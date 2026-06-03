package hub

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Redis key/channel helpers for cross-process WebSocket routing.
const wsChannelPrefix = "pushnotify:ws:"

func presenceKey(workspaceID string) string { return "pushnotify:presence:" + workspaceID }
func wsChannel(workspaceID string) string   { return wsChannelPrefix + workspaceID }

// bridgeEnvelope is the pub/sub message used to route a payload to a
// subscriber's live connections on whichever API node holds them.
type bridgeEnvelope struct {
	SubscriberID string `json:"subscriber_id"`
	Payload      []byte `json:"payload"`
}

// Message targets a single subscriber within a workspace.
type Message struct {
	WorkspaceID  string
	SubscriberID string
	Payload      []byte
}

// WorkspaceMessage targets every connection in a workspace.
type WorkspaceMessage struct {
	WorkspaceID string
	Payload     []byte
}

// Hub is the central registry of live WebSocket connections, keyed by
// workspace and subscriber. All mutation happens on the Run goroutine via
// channels; reads of the map are guarded by the RWMutex.
type Hub struct {
	rooms      map[string]map[string]map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	send       chan *Message
	broadcast  chan *WorkspaceMessage
	mu         sync.RWMutex
	log        *zap.Logger
	rdb        *redis.Client
}

// New creates a Hub. Pass a Redis client to enable cross-process delivery and
// presence tracking. Call Run in a goroutine to start it.
func New(log *zap.Logger, rdb *redis.Client) *Hub {
	return &Hub{
		rooms:      make(map[string]map[string]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		send:       make(chan *Message, 256),
		broadcast:  make(chan *WorkspaceMessage, 256),
		log:        log,
		rdb:        rdb,
	}
}

// Run is the central event loop. It owns all writes to the rooms map.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.addClient(c)
		case c := <-h.unregister:
			h.removeClient(c)
		case m := <-h.send:
			h.deliver(m)
		case b := <-h.broadcast:
			h.deliverBroadcast(b)
		}
	}
}

func (h *Hub) addClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ws, ok := h.rooms[c.WorkspaceID]
	if !ok {
		ws = make(map[string]map[*Client]struct{})
		h.rooms[c.WorkspaceID] = ws
	}
	subs, ok := ws[c.SubscriberID]
	if !ok {
		subs = make(map[*Client]struct{})
		ws[c.SubscriberID] = subs
	}
	subs[c] = struct{}{}
	if h.rdb != nil {
		// Refcount presence so a subscriber with multiple tabs stays online
		// until the last one disconnects.
		_ = h.rdb.HIncrBy(context.Background(), presenceKey(c.WorkspaceID), c.SubscriberID, 1).Err()
	}
	h.log.Debug("ws client registered",
		zap.String("workspace", c.WorkspaceID),
		zap.String("subscriber", c.SubscriberID))
}

func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ws, ok := h.rooms[c.WorkspaceID]
	if !ok {
		return
	}
	subs, ok := ws[c.SubscriberID]
	if !ok {
		return
	}
	if _, ok := subs[c]; ok {
		delete(subs, c)
		close(c.send)
		if h.rdb != nil {
			ctx := context.Background()
			n, err := h.rdb.HIncrBy(ctx, presenceKey(c.WorkspaceID), c.SubscriberID, -1).Result()
			if err == nil && n <= 0 {
				_ = h.rdb.HDel(ctx, presenceKey(c.WorkspaceID), c.SubscriberID).Err()
			}
		}
	}
	if len(subs) == 0 {
		delete(ws, c.SubscriberID)
	}
	if len(ws) == 0 {
		delete(h.rooms, c.WorkspaceID)
	}
}

func (h *Hub) deliver(m *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ws, ok := h.rooms[m.WorkspaceID]
	if !ok {
		return
	}
	subs, ok := ws[m.SubscriberID]
	if !ok {
		return
	}
	for c := range subs {
		h.trySend(c, m.Payload)
	}
}

func (h *Hub) deliverBroadcast(b *WorkspaceMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ws, ok := h.rooms[b.WorkspaceID]
	if !ok {
		return
	}
	for _, subs := range ws {
		for c := range subs {
			h.trySend(c, b.Payload)
		}
	}
}

// trySend performs a non-blocking send; slow clients are skipped rather than
// stalling the hub. The reader/writer pumps handle eventual cleanup.
func (h *Hub) trySend(c *Client, payload []byte) {
	select {
	case c.send <- payload:
	default:
		h.log.Warn("ws client send buffer full, dropping message",
			zap.String("subscriber", c.SubscriberID))
	}
}

// RegisterClient registers a new connection with the hub.
func (h *Hub) RegisterClient(c *Client) { h.register <- c }

// UnregisterClient removes a connection from the hub.
func (h *Hub) UnregisterClient(c *Client) { h.unregister <- c }

// SendToSubscriber delivers a payload to every connection a subscriber holds.
// It reports whether the subscriber had at least one live connection.
func (h *Hub) SendToSubscriber(workspaceID, subscriberID string, payload []byte) bool {
	online := h.GetSubscriberConnections(workspaceID, subscriberID) > 0
	h.send <- &Message{WorkspaceID: workspaceID, SubscriberID: subscriberID, Payload: payload}
	return online
}

// BroadcastToWorkspace delivers a payload to every connection in a workspace.
func (h *Hub) BroadcastToWorkspace(workspaceID string, payload []byte) {
	h.broadcast <- &WorkspaceMessage{WorkspaceID: workspaceID, Payload: payload}
}

// GetConnectedCount returns the number of live connections in a workspace.
func (h *Hub) GetConnectedCount(workspaceID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ws, ok := h.rooms[workspaceID]
	if !ok {
		return 0
	}
	n := 0
	for _, subs := range ws {
		n += len(subs)
	}
	return n
}

// GetSubscriberConnections returns the number of live connections a single
// subscriber currently holds on this node.
func (h *Hub) GetSubscriberConnections(workspaceID, subscriberID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ws, ok := h.rooms[workspaceID]
	if !ok {
		return 0
	}
	return len(ws[subscriberID])
}

// PublishToSubscriber routes a payload to a subscriber across all API nodes via
// Redis pub/sub, returning whether the subscriber is currently online (per the
// Redis presence map). Workers use this instead of the in-memory hub. When no
// Redis client is configured it falls back to local delivery.
func (h *Hub) PublishToSubscriber(ctx context.Context, workspaceID, subscriberID string, payload []byte) (bool, error) {
	if h.rdb == nil {
		return h.SendToSubscriber(workspaceID, subscriberID, payload), nil
	}
	online, err := h.rdb.HExists(ctx, presenceKey(workspaceID), subscriberID).Result()
	if err != nil {
		return false, err
	}
	env, err := json.Marshal(bridgeEnvelope{SubscriberID: subscriberID, Payload: payload})
	if err != nil {
		return online, err
	}
	if err := h.rdb.Publish(ctx, wsChannel(workspaceID), env).Err(); err != nil {
		return online, err
	}
	return online, nil
}

// RunBridge subscribes to the Redis WebSocket channels and forwards inbound
// envelopes to this node's local connections. Call it in a goroutine on API
// nodes that own WebSocket connections. It returns when ctx is cancelled.
func (h *Hub) RunBridge(ctx context.Context) {
	if h.rdb == nil {
		return
	}
	sub := h.rdb.PSubscribe(ctx, wsChannelPrefix+"*")
	defer func() { _ = sub.Close() }()
	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			workspaceID := strings.TrimPrefix(msg.Channel, wsChannelPrefix)
			var env bridgeEnvelope
			if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
				h.log.Warn("ws bridge decode failed", zap.Error(err))
				continue
			}
			// Deliver to local connections only.
			h.send <- &Message{WorkspaceID: workspaceID, SubscriberID: env.SubscriberID, Payload: env.Payload}
		}
	}
}
