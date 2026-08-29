-- Initial schema for clipfolio.

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE videos (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'uploading', -- uploading | transcoding | ready | failed
    error TEXT NOT NULL DEFAULT '',
    duration_seconds DOUBLE PRECISION NOT NULL DEFAULT 0,
    source_key TEXT NOT NULL DEFAULT '',
    playlist_key TEXT NOT NULL DEFAULT '', -- storage key of the master HLS playlist
    thumbnail_key TEXT NOT NULL DEFAULT '',
    webhook_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ctas (
    id BIGSERIAL PRIMARY KEY,
    video_id BIGINT NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    trigger TEXT NOT NULL, -- 'timestamp' | 'end'
    timestamp_seconds DOUBLE PRECISION NOT NULL DEFAULT 0,
    label TEXT NOT NULL,
    url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE lead_gates (
    video_id BIGINT PRIMARY KEY REFERENCES videos(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    position TEXT NOT NULL DEFAULT 'before', -- 'before' | 'timestamp'
    timestamp_seconds DOUBLE PRECISION NOT NULL DEFAULT 0,
    headline TEXT NOT NULL DEFAULT 'Enter your email to keep watching',
    require_name BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE viewer_sessions (
    id TEXT PRIMARY KEY, -- client-generated UUID, one per embed load
    video_id BIGINT NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    max_time_seconds DOUBLE PRECISION NOT NULL DEFAULT 0,
    played BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_viewer_sessions_video ON viewer_sessions(video_id);

CREATE TABLE leads (
    id BIGSERIAL PRIMARY KEY,
    video_id BIGINT NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL,
    email TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_leads_video ON leads(video_id);

CREATE TABLE cta_clicks (
    id BIGSERIAL PRIMARY KEY,
    cta_id BIGINT NOT NULL REFERENCES ctas(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
