package main

import (
	"context"
	"errors"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/astrazstudio/pushnotify/server/internal/models"
	"github.com/astrazstudio/pushnotify/server/internal/store"
	"github.com/astrazstudio/pushnotify/server/pkg/crypto"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// runSeed populates the database with a demo account, workspace, subscriber,
// API key and template. It is idempotent on the demo email.
func runSeed(ctx context.Context, pool *pgxpool.Pool, log *zap.Logger) {
	st := store.New(pool)

	const email = "demo@pushnotify.dev"
	const password = "demo1234"

	user, err := st.GetUserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			log.Fatal("seed: lookup user", zap.Error(err))
		}
		hash, herr := crypto.HashPassword(password)
		if herr != nil {
			log.Fatal("seed: hash", zap.Error(herr))
		}
		user, err = st.CreateUser(ctx, email, "Demo User", hash)
		if err != nil {
			log.Fatal("seed: create user", zap.Error(err))
		}
	}

	ws, err := st.GetWorkspaceByOwner(ctx, user.ID)
	if err != nil {
		priv, pub, gerr := webpush.GenerateVAPIDKeys()
		if gerr != nil {
			log.Fatal("seed: vapid", zap.Error(gerr))
		}
		ws, err = st.CreateWorkspace(ctx, user.ID, "Demo Workspace", "demo-workspace", pub, priv)
		if err != nil {
			log.Fatal("seed: create workspace", zap.Error(err))
		}
	}

	plain, keyHash, prefix, err := crypto.GenerateAPIKey()
	if err != nil {
		log.Fatal("seed: api key", zap.Error(err))
	}
	if _, err := st.CreateAPIKey(ctx, ws.ID, "seed-key", keyHash, prefix); err != nil {
		log.Warn("seed: create api key", zap.Error(err))
	}

	email2 := "subscriber@example.com"
	if _, err := st.UpsertSubscriber(ctx, ws.ID, "user_123", &email2, nil, nil, []byte(`{"plan":"pro"}`)); err != nil {
		log.Warn("seed: subscriber", zap.Error(err))
	}

	tmpl := &models.Template{
		WorkspaceID:   ws.ID,
		Name:          "Order Confirmed",
		Slug:          "order-confirmed",
		Channels:      []string{"websocket", "email"},
		Subject:       "Order {{order_id}} confirmed",
		BodyWebSocket: "Your order {{order_id}} is confirmed, {{name}}!",
		BodyEmail:     "Hi {{name}}, your order {{order_id}} has been confirmed.",
		Variables:     []string{"name", "order_id"},
	}
	if _, err := st.CreateTemplate(ctx, tmpl); err != nil {
		log.Warn("seed: template (may already exist)", zap.Error(err))
	}

	log.Info("seed complete",
		zap.String("login_email", email),
		zap.String("login_password", password),
		zap.String("workspace", ws.Slug),
		zap.String("api_key", plain),
	)
}
