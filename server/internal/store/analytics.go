package store

import (
	"context"
	"fmt"
	"time"
)

// DeliveryTotals aggregates delivery outcomes over a window.
type DeliveryTotals struct {
	Sent      int64 `json:"sent"`
	Delivered int64 `json:"delivered"`
	Failed    int64 `json:"failed"`
}

// CountDeliveries returns aggregate delivery totals since a point in time.
func (s *Store) CountDeliveries(ctx context.Context, workspaceID string, since time.Time) (DeliveryTotals, error) {
	var t DeliveryTotals
	err := s.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS sent,
			COUNT(*) FILTER (WHERE status = 'delivered') AS delivered,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed
		FROM deliveries
		WHERE workspace_id = $1 AND created_at >= $2`, workspaceID, since,
	).Scan(&t.Sent, &t.Delivered, &t.Failed)
	if err != nil {
		return t, fmt.Errorf("count deliveries: %w", err)
	}
	return t, nil
}

// ChannelTotals is a per-channel breakdown row.
type ChannelTotals struct {
	Channel   string `json:"channel"`
	Sent      int64  `json:"sent"`
	Delivered int64  `json:"delivered"`
	Failed    int64  `json:"failed"`
}

// CountDeliveriesByChannel breaks down delivery totals per channel.
func (s *Store) CountDeliveriesByChannel(ctx context.Context, workspaceID string, since time.Time) ([]ChannelTotals, error) {
	rows, err := s.db.Query(ctx, `
		SELECT channel,
			COUNT(*) AS sent,
			COUNT(*) FILTER (WHERE status = 'delivered') AS delivered,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed
		FROM deliveries
		WHERE workspace_id = $1 AND created_at >= $2
		GROUP BY channel`, workspaceID, since)
	if err != nil {
		return nil, fmt.Errorf("count by channel: %w", err)
	}
	defer rows.Close()
	var out []ChannelTotals
	for rows.Next() {
		var c ChannelTotals
		if err := rows.Scan(&c.Channel, &c.Sent, &c.Delivered, &c.Failed); err != nil {
			return nil, fmt.Errorf("scan channel totals: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// TimeBucket is one point in a time-series chart.
type TimeBucket struct {
	Bucket    time.Time `json:"bucket"`
	Sent      int64     `json:"sent"`
	Delivered int64     `json:"delivered"`
}

// DeliveriesTimeSeries buckets deliveries by the given interval (e.g. 'hour').
func (s *Store) DeliveriesTimeSeries(ctx context.Context, workspaceID string, since time.Time, trunc string) ([]TimeBucket, error) {
	// trunc is validated by the caller against a fixed allowlist; never
	// interpolate raw user input here.
	if trunc != "hour" && trunc != "day" {
		trunc = "hour"
	}
	rows, err := s.db.Query(ctx, fmt.Sprintf(`
		SELECT date_trunc('%s', created_at) AS bucket,
			COUNT(*) AS sent,
			COUNT(*) FILTER (WHERE status = 'delivered') AS delivered
		FROM deliveries
		WHERE workspace_id = $1 AND created_at >= $2
		GROUP BY bucket ORDER BY bucket`, trunc), workspaceID, since)
	if err != nil {
		return nil, fmt.Errorf("time series: %w", err)
	}
	defer rows.Close()
	var out []TimeBucket
	for rows.Next() {
		var b TimeBucket
		if err := rows.Scan(&b.Bucket, &b.Sent, &b.Delivered); err != nil {
			return nil, fmt.Errorf("scan time bucket: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// TopFailing returns subscribers with the most failed deliveries.
type FailingSubscriber struct {
	SubscriberID string `json:"subscriber_id"`
	ExternalID   string `json:"external_id"`
	Failures     int64  `json:"failures"`
}

// TopFailingSubscribers lists the subscribers with the most failures.
func (s *Store) TopFailingSubscribers(ctx context.Context, workspaceID string, since time.Time, limit int) ([]FailingSubscriber, error) {
	rows, err := s.db.Query(ctx, `
		SELECT s.id, s.external_id, COUNT(*) AS failures
		FROM deliveries d
		JOIN notifications n ON n.id = d.notification_id
		JOIN subscribers s ON s.id = n.subscriber_id
		WHERE d.workspace_id = $1 AND d.status = 'failed' AND d.created_at >= $2
		GROUP BY s.id, s.external_id
		ORDER BY failures DESC LIMIT $3`, workspaceID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("top failing: %w", err)
	}
	defer rows.Close()
	var out []FailingSubscriber
	for rows.Next() {
		var f FailingSubscriber
		if err := rows.Scan(&f.SubscriberID, &f.ExternalID, &f.Failures); err != nil {
			return nil, fmt.Errorf("scan failing: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
