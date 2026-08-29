# Changelog

All notable changes to clipfolio are documented in this file. Format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.1.0] - 2026-08-30

### Added
- Video upload with background transcoding to adaptive-bitrate HLS (360p/720p/1080p, skipping
  upscaling past the source resolution).
- An embeddable player (`player.js` + a `<div data-clipfolio-video>` snippet) with HLS playback,
  CTA overlays (at a timestamp or at video end), and an optional email-capture lead gate.
- Per-video analytics: impressions, plays, play rate, average watch percentage, and a drop-off
  (retention) curve, computed from real viewer-session heartbeats.
- Lead capture with a per-video webhook, so captured leads can be wired into any CRM without a
  native integration.
- A dashboard (React + TypeScript) to upload videos, configure CTAs and the lead gate, view
  analytics, and copy the embed snippet.
- Docker Compose setup bundling Postgres and MinIO, so `docker compose up --build` runs the whole
  stack with no cloud account required to try it.

[Unreleased]: https://github.com/Laaaaksh/clipfolio/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Laaaaksh/clipfolio/compare/305597b...v0.1.0
