package channels

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/astrazstudio/pushnotify/server/internal/config"
	"github.com/astrazstudio/pushnotify/server/internal/hub"
	"github.com/astrazstudio/pushnotify/server/internal/models"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func TestWebSocketChannel_OfflineSubscriber(t *testing.T) {
	// No Redis configured -> falls back to the in-memory hub, which has no
	// connections, so the subscriber is offline.
	h := hub.New(zap.NewNop(), nil)
	go h.Run()
	ch := NewWebSocketChannel(h)

	res, err := ch.Send(context.Background(), DeliveryJob{
		WorkspaceID: "ws_1",
		Subscriber:  &models.Subscriber{ExternalID: "ghost"},
		Body:        "hi",
	})
	if err != nil {
		t.Fatalf("websocket is best-effort; expected nil error, got %v", err)
	}
	if res.Success || res.Error != "offline" {
		t.Fatalf("expected offline result, got %+v", res)
	}
}

func TestWebSocketChannel_OnlineSubscriber(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	ctx := context.Background()
	h := hub.New(zap.NewNop(), rdb)
	ch := NewWebSocketChannel(h)

	// Mark the subscriber present (as a live connection would) and subscribe
	// to the workspace channel the bridge publishes on.
	if err := rdb.HSet(ctx, "pushnotify:presence:ws_1", "user_123", 1).Err(); err != nil {
		t.Fatalf("hset presence: %v", err)
	}
	pubsub := rdb.Subscribe(ctx, "pushnotify:ws:ws_1")
	defer pubsub.Close()
	if _, err := pubsub.Receive(ctx); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	res, err := ch.Send(ctx, DeliveryJob{
		WorkspaceID: "ws_1",
		Subscriber:  &models.Subscriber{ExternalID: "user_123"},
		Subject:     "Title",
		Body:        "hello world",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success for online subscriber, got %+v", res)
	}

	select {
	case msg := <-pubsub.Channel():
		var env struct {
			SubscriberID string `json:"subscriber_id"`
			Payload      []byte `json:"payload"`
		}
		if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
			t.Fatalf("decode envelope: %v", err)
		}
		if env.SubscriberID != "user_123" {
			t.Errorf("wrong subscriber: %s", env.SubscriberID)
		}
		if !strings.Contains(string(env.Payload), "hello world") {
			t.Errorf("payload missing body: %s", env.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for published payload")
	}
}

func TestEmailChannel_MissingConfig(t *testing.T) {
	ch := NewEmailChannel(config.EmailConfig{}) // empty host
	if err := ch.Validate(); err == nil {
		t.Error("expected Validate to fail without SMTP host")
	}
	res, err := ch.Send(context.Background(), DeliveryJob{
		Subscriber: &models.Subscriber{Email: strptr("a@b.com")},
		Body:       "hi",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success || !strings.Contains(res.Error, "smtp") {
		t.Fatalf("expected descriptive smtp config error, got %+v", res)
	}
}

func TestWebPushChannel_InvalidSubscription(t *testing.T) {
	ch := NewWebPushChannel()
	res, err := ch.Send(context.Background(), DeliveryJob{
		Subscriber: &models.Subscriber{ExternalID: "user_123"}, // no web_push_subscription
		Body:       "hi",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success || !strings.Contains(res.Error, "no web push subscription") {
		t.Fatalf("expected no-subscription error, got %+v", res)
	}
}

func strptr(s string) *string { return &s }
