package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/channels"
	"github.com/astrazstudio/pushnotify/server/internal/config"
	"github.com/astrazstudio/pushnotify/server/internal/database"
	"github.com/astrazstudio/pushnotify/server/internal/handlers"
	"github.com/astrazstudio/pushnotify/server/internal/hub"
	"github.com/astrazstudio/pushnotify/server/internal/queue"
	"github.com/astrazstudio/pushnotify/server/internal/router"
	"github.com/astrazstudio/pushnotify/server/internal/services"
	"github.com/astrazstudio/pushnotify/server/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("load config", zap.Error(err))
	}

	mode := "server"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	ctx := context.Background()

	pool, err := database.NewPostgres(ctx, cfg.Database)
	if err != nil {
		log.Fatal("connect postgres", zap.Error(err))
	}
	defer pool.Close()

	rdb, err := database.NewRedis(ctx, cfg.Redis)
	if err != nil {
		log.Fatal("connect redis", zap.Error(err))
	}
	defer func() { _ = rdb.Close() }()

	switch mode {
	case "migrate":
		if err := database.RunMigrations(ctx, pool); err != nil {
			log.Fatal("migrate", zap.Error(err))
		}
		log.Info("migrations applied")
	case "seed":
		if err := database.RunMigrations(ctx, pool); err != nil {
			log.Fatal("migrate", zap.Error(err))
		}
		runSeed(ctx, pool, log)
	case "worker":
		runWorker(ctx, cfg, pool, rdb, log)
	default:
		// Apply migrations on boot so a single command brings the API up.
		if err := database.RunMigrations(ctx, pool); err != nil {
			log.Fatal("migrate", zap.Error(err))
		}
		runServer(ctx, cfg, pool, rdb, log)
	}
}

// app holds the shared, wired components.
type app struct {
	store    *store.Store
	hub      *hub.Hub
	producer *queue.RedisProducer
	notifier *services.NotificationService
	handlers *handlers.Handlers
}

func buildApp(cfg *config.Config, pool *pgxpool.Pool, rdb *redis.Client, log *zap.Logger) *app {
	st := store.New(pool)
	h := hub.New(log, rdb)

	producer := queue.NewProducer(rdb)
	tmpl := services.NewTemplateService()
	webhookSvc := services.NewWebhookService(st, log)

	chans := map[string]channels.Channel{
		channels.ChannelWebSocket: channels.NewWebSocketChannel(h),
		channels.ChannelWebPush:   channels.NewWebPushChannel(),
		channels.ChannelEmail:     channels.NewEmailChannel(cfg.Email),
		channels.ChannelWhatsApp:  channels.NewWhatsAppChannel(cfg.WhatsApp),
		channels.ChannelSMS:       channels.NewSMSChannel(cfg.SMS),
	}

	notifier := services.NewNotificationService(st, producer, tmpl, webhookSvc, chans, log)
	subscriberSvc := services.NewSubscriberService(st)
	analyticsSvc := services.NewAnalyticsService(st, rdb, h, log)
	billingSvc := services.NewBillingService(st, cfg.Billing, cfg.Server.AppURL, log)

	hs := &handlers.Handlers{
		Cfg:         cfg,
		Store:       st,
		Redis:       rdb,
		Hub:         h,
		Notifier:    notifier,
		Subscribers: subscriberSvc,
		Analytics:   analyticsSvc,
		Templates:   tmpl,
		Billing:     billingSvc,
		Log:         log,
	}

	return &app{store: st, hub: h, producer: producer, notifier: notifier, handlers: hs}
}

// newConsumer builds a consumer with the notification handler registered for
// every channel.
func newConsumer(cfg *config.Config, rdb *redis.Client, producer *queue.RedisProducer, notifier *services.NotificationService, log *zap.Logger) *queue.RedisConsumer {
	consumer := queue.NewConsumer(rdb, producer, log, cfg.Queue.Workers, cfg.Queue.RetryBackoffBase)
	for _, name := range []string{
		channels.ChannelWebSocket, channels.ChannelWebPush, channels.ChannelEmail,
		channels.ChannelWhatsApp, channels.ChannelSMS,
	} {
		consumer.SetHandler(name, notifier.HandleDelivery)
	}
	return consumer
}

func runServer(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, rdb *redis.Client, log *zap.Logger) {
	a := buildApp(cfg, pool, rdb, log)

	bridgeCtx, cancelBridge := context.WithCancel(ctx)
	go a.hub.Run()
	go a.hub.RunBridge(bridgeCtx)

	// Run an in-process consumer so `pushnotify` works end-to-end alone.
	consumer := newConsumer(cfg, rdb, a.producer, a.notifier, log)
	consumer.Start(ctx)

	handler := router.New(a.handlers, rdb, cfg.JWT.Secret, log)
	srv := &http.Server{
		Addr:              cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("http server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("http server", zap.Error(err))
		}
	}()

	waitForSignal(log)

	log.Info("shutting down gracefully")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Warn("http shutdown", zap.Error(err))
	}
	consumer.Stop() // drains in-flight jobs
	cancelBridge()
}

func runWorker(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, rdb *redis.Client, log *zap.Logger) {
	a := buildApp(cfg, pool, rdb, log)
	consumer := newConsumer(cfg, rdb, a.producer, a.notifier, log)
	consumer.Start(ctx)
	log.Info("worker started", zap.Int("workers", cfg.Queue.Workers))

	waitForSignal(log)
	log.Info("worker draining queue before exit")
	consumer.Stop()
}

func waitForSignal(log *zap.Logger) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Info("signal received", zap.String("signal", s.String()))
}
