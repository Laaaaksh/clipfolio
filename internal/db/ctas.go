package db

import (
	"context"
	"fmt"
)

// CreateCTA adds a clickable overlay to a video.
func (s *Store) CreateCTA(ctx context.Context, videoID int64, trigger string, timestampSeconds float64, label, url string) (CTA, error) {
	var c CTA
	err := s.pool.QueryRow(ctx,
		`INSERT INTO ctas (video_id, trigger, timestamp_seconds, label, url) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, video_id, trigger, timestamp_seconds, label, url, created_at`,
		videoID, trigger, timestampSeconds, label, url,
	).Scan(&c.ID, &c.VideoID, &c.Trigger, &c.TimestampSeconds, &c.Label, &c.URL, &c.CreatedAt)
	if err != nil {
		return CTA{}, fmt.Errorf("create cta: %w", err)
	}
	return c, nil
}

// ListCTAs returns a video's CTAs ordered by when they fire.
func (s *Store) ListCTAs(ctx context.Context, videoID int64) ([]CTA, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, video_id, trigger, timestamp_seconds, label, url, created_at FROM ctas WHERE video_id = $1 ORDER BY timestamp_seconds ASC`,
		videoID,
	)
	if err != nil {
		return nil, fmt.Errorf("list ctas: %w", err)
	}
	defer rows.Close()

	ctas := []CTA{}
	for rows.Next() {
		var c CTA
		if err := rows.Scan(&c.ID, &c.VideoID, &c.Trigger, &c.TimestampSeconds, &c.Label, &c.URL, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan cta: %w", err)
		}
		ctas = append(ctas, c)
	}
	return ctas, rows.Err()
}

// DeleteCTA removes a CTA, scoped to videoID so a caller can never delete a
// CTA belonging to a video they don't own by guessing another CTA's id.
func (s *Store) DeleteCTA(ctx context.Context, videoID, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM ctas WHERE id = $1 AND video_id = $2`, id, videoID)
	if err != nil {
		return fmt.Errorf("delete cta: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RecordCTAClick logs a viewer clicking a CTA.
func (s *Store) RecordCTAClick(ctx context.Context, ctaID int64, sessionID string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO cta_clicks (cta_id, session_id) VALUES ($1, $2)`, ctaID, sessionID)
	if err != nil {
		return fmt.Errorf("record cta click: %w", err)
	}
	return nil
}

// UpsertLeadGate creates or replaces a video's lead-gate configuration.
func (s *Store) UpsertLeadGate(ctx context.Context, g LeadGate) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO lead_gates (video_id, enabled, position, timestamp_seconds, headline, require_name)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (video_id) DO UPDATE SET
		   enabled = excluded.enabled,
		   position = excluded.position,
		   timestamp_seconds = excluded.timestamp_seconds,
		   headline = excluded.headline,
		   require_name = excluded.require_name`,
		g.VideoID, g.Enabled, g.Position, g.TimestampSeconds, g.Headline, g.RequireName,
	)
	if err != nil {
		return fmt.Errorf("upsert lead gate: %w", err)
	}
	return nil
}

// GetLeadGate fetches a video's lead-gate configuration, defaulting to
// disabled if none has been set.
func (s *Store) GetLeadGate(ctx context.Context, videoID int64) (LeadGate, error) {
	var g LeadGate
	err := s.pool.QueryRow(ctx,
		`SELECT video_id, enabled, position, timestamp_seconds, headline, require_name FROM lead_gates WHERE video_id = $1`,
		videoID,
	).Scan(&g.VideoID, &g.Enabled, &g.Position, &g.TimestampSeconds, &g.Headline, &g.RequireName)
	if err != nil {
		return wrapNotFoundLeadGate(videoID), wrapNotFound(err)
	}
	return g, nil
}

// wrapNotFoundLeadGate returns the sensible default (disabled) lead gate for
// a video that has never had one configured, so callers can treat "not
// found" and "disabled" the same way.
func wrapNotFoundLeadGate(videoID int64) LeadGate {
	return LeadGate{VideoID: videoID, Enabled: false, Position: LeadGatePositionBefore, Headline: "Enter your email to keep watching"}
}
