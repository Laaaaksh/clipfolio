package db

import (
	"context"
	"fmt"
)

// UpsertViewerProgress records or advances a viewer session's furthest
// reached playback time. It never lets max_time_seconds go backward (a
// heartbeat racing a seek-back must not undercount a viewer's real progress)
// and "played" latches true forever once set.
func (s *Store) UpsertViewerProgress(ctx context.Context, sessionID string, videoID int64, maxTimeSeconds float64, played bool) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO viewer_sessions (id, video_id, max_time_seconds, played)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (id) DO UPDATE SET
		   max_time_seconds = GREATEST(viewer_sessions.max_time_seconds, excluded.max_time_seconds),
		   played = viewer_sessions.played OR excluded.played,
		   updated_at = now()`,
		sessionID, videoID, maxTimeSeconds, played,
	)
	if err != nil {
		return fmt.Errorf("upsert viewer progress: %w", err)
	}
	return nil
}

// ViewerSessionsForVideo returns every viewer session's max reached time and
// played flag, the raw material analytics.Summarize turns into a summary.
func (s *Store) ViewerSessionsForVideo(ctx context.Context, videoID int64) ([]ViewerSession, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, video_id, max_time_seconds, played, created_at, updated_at FROM viewer_sessions WHERE video_id = $1`,
		videoID,
	)
	if err != nil {
		return nil, fmt.Errorf("list viewer sessions: %w", err)
	}
	defer rows.Close()

	sessions := []ViewerSession{}
	for rows.Next() {
		var v ViewerSession
		if err := rows.Scan(&v.ID, &v.VideoID, &v.MaxTimeSeconds, &v.Played, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan viewer session: %w", err)
		}
		sessions = append(sessions, v)
	}
	return sessions, rows.Err()
}

// CreateLead records an email captured through a video's lead gate.
func (s *Store) CreateLead(ctx context.Context, videoID int64, sessionID, email, name string) (Lead, error) {
	var l Lead
	err := s.pool.QueryRow(ctx,
		`INSERT INTO leads (video_id, session_id, email, name) VALUES ($1, $2, $3, $4)
		 RETURNING id, video_id, session_id, email, name, created_at`,
		videoID, sessionID, email, name,
	).Scan(&l.ID, &l.VideoID, &l.SessionID, &l.Email, &l.Name, &l.CreatedAt)
	if err != nil {
		return Lead{}, fmt.Errorf("create lead: %w", err)
	}
	return l, nil
}

// ListLeads returns a video's captured leads, newest first.
func (s *Store) ListLeads(ctx context.Context, videoID int64) ([]Lead, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, video_id, session_id, email, name, created_at FROM leads WHERE video_id = $1 ORDER BY created_at DESC`,
		videoID,
	)
	if err != nil {
		return nil, fmt.Errorf("list leads: %w", err)
	}
	defer rows.Close()

	leads := []Lead{}
	for rows.Next() {
		var l Lead
		if err := rows.Scan(&l.ID, &l.VideoID, &l.SessionID, &l.Email, &l.Name, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan lead: %w", err)
		}
		leads = append(leads, l)
	}
	return leads, rows.Err()
}
