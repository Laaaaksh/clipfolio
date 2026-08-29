package db

import (
	"context"
	"fmt"
)

// CreateVideo creates a video record awaiting upload.
func (s *Store) CreateVideo(ctx context.Context, ownerID int64, title string) (Video, error) {
	var v Video
	err := s.pool.QueryRow(ctx,
		`INSERT INTO videos (owner_id, title, status) VALUES ($1, $2, $3)
		 RETURNING id, owner_id, title, status, error, duration_seconds, source_key, playlist_key, thumbnail_key, webhook_url, created_at, updated_at`,
		ownerID, title, VideoStatusUploading,
	).Scan(&v.ID, &v.OwnerID, &v.Title, &v.Status, &v.Error, &v.DurationSeconds, &v.SourceKey, &v.PlaylistKey, &v.ThumbnailKey, &v.WebhookURL, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return Video{}, fmt.Errorf("create video: %w", err)
	}
	return v, nil
}

// GetVideo fetches a video by id.
func (s *Store) GetVideo(ctx context.Context, id int64) (Video, error) {
	var v Video
	err := s.pool.QueryRow(ctx,
		`SELECT id, owner_id, title, status, error, duration_seconds, source_key, playlist_key, thumbnail_key, webhook_url, created_at, updated_at
		 FROM videos WHERE id = $1`,
		id,
	).Scan(&v.ID, &v.OwnerID, &v.Title, &v.Status, &v.Error, &v.DurationSeconds, &v.SourceKey, &v.PlaylistKey, &v.ThumbnailKey, &v.WebhookURL, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return Video{}, wrapNotFound(err)
	}
	return v, nil
}

// ListVideos returns an owner's videos, newest first.
func (s *Store) ListVideos(ctx context.Context, ownerID int64) ([]Video, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, owner_id, title, status, error, duration_seconds, source_key, playlist_key, thumbnail_key, webhook_url, created_at, updated_at
		 FROM videos WHERE owner_id = $1 ORDER BY created_at DESC`,
		ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("list videos: %w", err)
	}
	defer rows.Close()

	videos := []Video{}
	for rows.Next() {
		var v Video
		if err := rows.Scan(&v.ID, &v.OwnerID, &v.Title, &v.Status, &v.Error, &v.DurationSeconds, &v.SourceKey, &v.PlaylistKey, &v.ThumbnailKey, &v.WebhookURL, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan video: %w", err)
		}
		videos = append(videos, v)
	}
	return videos, rows.Err()
}

// SetVideoSourceKey records where the uploaded source landed and marks the
// video as transcoding.
func (s *Store) SetVideoSourceKey(ctx context.Context, id int64, sourceKey string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE videos SET source_key = $2, status = $3, updated_at = now() WHERE id = $1`,
		id, sourceKey, VideoStatusTranscoding,
	)
	if err != nil {
		return fmt.Errorf("set video source key: %w", err)
	}
	return nil
}

// MarkVideoReady records a completed transcode's outputs.
func (s *Store) MarkVideoReady(ctx context.Context, id int64, durationSeconds float64, playlistKey, thumbnailKey string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE videos SET status = $2, duration_seconds = $3, playlist_key = $4, thumbnail_key = $5, updated_at = now() WHERE id = $1`,
		id, VideoStatusReady, durationSeconds, playlistKey, thumbnailKey,
	)
	if err != nil {
		return fmt.Errorf("mark video ready: %w", err)
	}
	return nil
}

// MarkVideoFailed records why a video's transcode failed.
func (s *Store) MarkVideoFailed(ctx context.Context, id int64, errMsg string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE videos SET status = $2, error = $3, updated_at = now() WHERE id = $1`,
		id, VideoStatusFailed, errMsg,
	)
	if err != nil {
		return fmt.Errorf("mark video failed: %w", err)
	}
	return nil
}

// SetVideoWebhook sets the URL notified when this video captures a lead.
func (s *Store) SetVideoWebhook(ctx context.Context, id int64, url string) error {
	_, err := s.pool.Exec(ctx, `UPDATE videos SET webhook_url = $2, updated_at = now() WHERE id = $1`, id, url)
	if err != nil {
		return fmt.Errorf("set video webhook: %w", err)
	}
	return nil
}

// DeleteVideo removes a video and its related rows (CTAs, leads, viewer
// sessions cascade via foreign keys).
func (s *Store) DeleteVideo(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM videos WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete video: %w", err)
	}
	return nil
}
