package hub

import (
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 30 * time.Second
	maxMessageSize = 8192
)

// Client is a single WebSocket connection owned by the hub.
type Client struct {
	hub          *Hub
	conn         *websocket.Conn
	send         chan []byte
	WorkspaceID  string
	SubscriberID string
	UserAgent    string
	ConnectedAt  time.Time
}

// NewClient wraps an upgraded WebSocket connection.
func NewClient(h *Hub, conn *websocket.Conn, workspaceID, subscriberID, userAgent string) *Client {
	return &Client{
		hub:          h,
		conn:         conn,
		send:         make(chan []byte, 64),
		WorkspaceID:  workspaceID,
		SubscriberID: subscriberID,
		UserAgent:    userAgent,
		ConnectedAt:  time.Now(),
	}
}

// Start registers the client and launches its read/write pumps.
func (c *Client) Start() {
	c.hub.RegisterClient(c)
	go c.writePump()
	go c.readPump()
}

// readPump drains inbound frames (mostly pongs/heartbeats) and triggers
// cleanup on disconnect. Inbound application messages are ignored: this is a
// one-way delivery channel.
func (c *Client) readPump() {
	defer func() {
		c.hub.UnregisterClient(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.log.Debug("ws read closed", zap.Error(err))
			}
			return
		}
	}
}

// writePump fans out hub messages and keeps the connection alive with pings.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
