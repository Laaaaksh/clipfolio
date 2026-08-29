package db

import "time"

// User is a dashboard admin account.
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Session is an opaque, cookie-carried dashboard login session.
type Session struct {
	Token     string    `json:"token"`
	UserID    int64     `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Video status values, tracking its lifecycle from upload through transcode.
const (
	VideoStatusUploading   = "uploading"
	VideoStatusTranscoding = "transcoding"
	VideoStatusReady       = "ready"
	VideoStatusFailed      = "failed"
)

// Video is one hosted video and its transcode/embed state.
type Video struct {
	ID              int64     `json:"id"`
	OwnerID         int64     `json:"ownerId"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	Error           string    `json:"error"`
	DurationSeconds float64   `json:"durationSeconds"`
	SourceKey       string    `json:"sourceKey"`
	PlaylistKey     string    `json:"playlistKey"`
	ThumbnailKey    string    `json:"thumbnailKey"`
	WebhookURL      string    `json:"webhookUrl"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// CTA trigger values: fire at a specific timestamp, or when the video ends.
const (
	CTATriggerTimestamp = "timestamp"
	CTATriggerEnd       = "end"
)

// CTA is a clickable overlay shown during or after video playback.
type CTA struct {
	ID               int64     `json:"id"`
	VideoID          int64     `json:"videoId"`
	Trigger          string    `json:"trigger"`
	TimestampSeconds float64   `json:"timestampSeconds"`
	Label            string    `json:"label"`
	URL              string    `json:"url"`
	CreatedAt        time.Time `json:"createdAt"`
}

// Lead gate position values: shown before playback starts, or at a timestamp.
const (
	LeadGatePositionBefore    = "before"
	LeadGatePositionTimestamp = "timestamp"
)

// LeadGate is a video's optional email-capture form configuration.
type LeadGate struct {
	VideoID          int64   `json:"videoId"`
	Enabled          bool    `json:"enabled"`
	Position         string  `json:"position"`
	TimestampSeconds float64 `json:"timestampSeconds"`
	Headline         string  `json:"headline"`
	RequireName      bool    `json:"requireName"`
}

// ViewerSession tracks one embed playback session's furthest-watched point.
type ViewerSession struct {
	ID             string    `json:"id"`
	VideoID        int64     `json:"videoId"`
	MaxTimeSeconds float64   `json:"maxTimeSeconds"`
	Played         bool      `json:"played"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// Lead is one email captured through a video's lead gate.
type Lead struct {
	ID        int64     `json:"id"`
	VideoID   int64     `json:"videoId"`
	SessionID string    `json:"sessionId"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}
