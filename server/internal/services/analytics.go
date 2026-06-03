package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/hub"
	"github.com/astrazstudio/pushnotify/server/internal/store"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Overview is the dashboard summary payload.
type Overview struct {
	TotalSent24h   int64                     `json:"total_sent_24h"`
	TotalSent7d    int64                     `json:"total_sent_7d"`
	TotalSent30d   int64                     `json:"total_sent_30d"`
	TotalDelivered int64                     `json:"total_delivered"`
	TotalFailed    int64                     `json:"total_failed"`
	DeliveryRate   float64                   `json:"delivery_rate"`
	ConnectedNow   int                       `json:"connected_now"`
	ByChannel      []store.ChannelTotals     `json:"by_channel"`
	HourlyChart    []store.TimeBucket        `json:"hourly_chart"`
	TopFailing     []store.FailingSubscriber `json:"top_failing"`
}

// AnalyticsService computes dashboard metrics with a short Redis cache.
type AnalyticsService struct {
	store *store.Store
	rdb   *redis.Client
	hub   *hub.Hub
	log   *zap.Logger
}

// NewAnalyticsService constructs the service.
func NewAnalyticsService(st *store.Store, rdb *redis.Client, h *hub.Hub, log *zap.Logger) *AnalyticsService {
	return &AnalyticsService{store: st, rdb: rdb, hub: h, log: log}
}

// GetOverview returns the dashboard overview, cached in Redis for 60s.
func (s *AnalyticsService) GetOverview(ctx context.Context, workspaceID string) (*Overview, error) {
	cacheKey := "pushnotify:analytics:overview:" + workspaceID
	if cached, err := s.rdb.Get(ctx, cacheKey).Bytes(); err == nil {
		var ov Overview
		if json.Unmarshal(cached, &ov) == nil {
			// Connection count is live; never serve it from cache.
			ov.ConnectedNow = s.hub.GetConnectedCount(workspaceID)
			return &ov, nil
		}
	}

	now := time.Now()
	since24h := now.Add(-24 * time.Hour)
	since7d := now.Add(-7 * 24 * time.Hour)
	since30d := now.Add(-30 * 24 * time.Hour)

	t24, err := s.store.CountDeliveries(ctx, workspaceID, since24h)
	if err != nil {
		return nil, err
	}
	t7, err := s.store.CountDeliveries(ctx, workspaceID, since7d)
	if err != nil {
		return nil, err
	}
	t30, err := s.store.CountDeliveries(ctx, workspaceID, since30d)
	if err != nil {
		return nil, err
	}
	byChannel, err := s.store.CountDeliveriesByChannel(ctx, workspaceID, since24h)
	if err != nil {
		return nil, err
	}
	hourly, err := s.store.DeliveriesTimeSeries(ctx, workspaceID, since24h, "hour")
	if err != nil {
		return nil, err
	}
	topFailing, err := s.store.TopFailingSubscribers(ctx, workspaceID, since7d, 5)
	if err != nil {
		return nil, err
	}

	rate := 0.0
	if t24.Sent > 0 {
		rate = float64(t24.Delivered) / float64(t24.Sent) * 100
	}

	ov := &Overview{
		TotalSent24h:   t24.Sent,
		TotalSent7d:    t7.Sent,
		TotalSent30d:   t30.Sent,
		TotalDelivered: t24.Delivered,
		TotalFailed:    t24.Failed,
		DeliveryRate:   rate,
		ConnectedNow:   s.hub.GetConnectedCount(workspaceID),
		ByChannel:      byChannel,
		HourlyChart:    hourly,
		TopFailing:     topFailing,
	}

	if data, err := json.Marshal(ov); err == nil {
		if err := s.rdb.Set(ctx, cacheKey, data, 60*time.Second).Err(); err != nil {
			s.log.Debug("cache overview failed", zap.Error(err))
		}
	}
	return ov, nil
}

// GetAnalytics returns a period-scoped time series ("24h", "7d", "30d").
func (s *AnalyticsService) GetAnalytics(ctx context.Context, workspaceID, period string) (map[string]interface{}, error) {
	var since time.Time
	trunc := "hour"
	now := time.Now()
	switch period {
	case "30d":
		since = now.Add(-30 * 24 * time.Hour)
		trunc = "day"
	case "7d":
		since = now.Add(-7 * 24 * time.Hour)
		trunc = "day"
	default:
		period = "24h"
		since = now.Add(-24 * time.Hour)
	}

	totals, err := s.store.CountDeliveries(ctx, workspaceID, since)
	if err != nil {
		return nil, err
	}
	series, err := s.store.DeliveriesTimeSeries(ctx, workspaceID, since, trunc)
	if err != nil {
		return nil, err
	}
	byChannel, err := s.store.CountDeliveriesByChannel(ctx, workspaceID, since)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"period":     period,
		"totals":     totals,
		"series":     series,
		"by_channel": byChannel,
	}, nil
}
