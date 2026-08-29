package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/laaaaksh/clipfolio/internal/db"
	"github.com/laaaaksh/clipfolio/internal/webhook"
)

// getPublicVideo loads a video for the public embed surface. Unlike
// getOwnedVideo this has no auth requirement - anyone with a video id can
// embed it, matching how Wistia/Vimeo embeds work - but it only ever returns
// a video that has finished transcoding, so a viewer never lands on a
// half-uploaded asset.
func (s *Server) getPublicVideo(r *http.Request) (db.Video, error) {
	id, err := videoIDFromRequest(r)
	if err != nil {
		return db.Video{}, db.ErrNotFound
	}
	video, err := s.store.GetVideo(r.Context(), id)
	if err != nil {
		return db.Video{}, err
	}
	if video.Status != db.VideoStatusReady {
		return db.Video{}, db.ErrNotFound
	}
	return video, nil
}

type embedManifest struct {
	ID              int64       `json:"id"`
	Title           string      `json:"title"`
	PlaylistURL     string      `json:"playlistUrl"`
	ThumbnailURL    string      `json:"thumbnailUrl,omitempty"`
	DurationSeconds float64     `json:"durationSeconds"`
	CTAs            []db.CTA    `json:"ctas"`
	LeadGate        db.LeadGate `json:"leadGate"`
}

func (s *Server) handleEmbedManifest(w http.ResponseWriter, r *http.Request) {
	video, err := s.getPublicVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found or not ready")
		return
	}

	ctas, err := s.store.ListCTAs(r.Context(), video.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load CTAs")
		return
	}

	gate, err := s.store.GetLeadGate(r.Context(), video.ID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "failed to load lead gate")
		return
	}

	manifest := embedManifest{
		ID:              video.ID,
		Title:           video.Title,
		PlaylistURL:     s.storage.PublicURL(video.PlaylistKey),
		DurationSeconds: video.DurationSeconds,
		CTAs:            ctas,
		LeadGate:        gate,
	}
	if video.ThumbnailKey != "" {
		manifest.ThumbnailURL = s.storage.PublicURL(video.ThumbnailKey)
	}

	writeJSON(w, http.StatusOK, manifest)
}

func (s *Server) handleEmbedProgress(w http.ResponseWriter, r *http.Request) {
	video, err := s.getPublicVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found or not ready")
		return
	}

	var req struct {
		SessionID      string  `json:"sessionId"`
		MaxTimeSeconds float64 `json:"maxTimeSeconds"`
		Played         bool    `json:"played"`
	}
	if err := decodeJSON(r, &req); err != nil || req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "sessionId is required")
		return
	}
	if req.MaxTimeSeconds < 0 {
		req.MaxTimeSeconds = 0
	}

	if err := s.store.UpsertViewerProgress(r.Context(), req.SessionID, video.ID, req.MaxTimeSeconds, req.Played); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record progress")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleEmbedLead(w http.ResponseWriter, r *http.Request) {
	video, err := s.getPublicVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found or not ready")
		return
	}

	var req struct {
		SessionID string `json:"sessionId"`
		Email     string `json:"email"`
		Name      string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil || req.SessionID == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "sessionId and email are required")
		return
	}

	lead, err := s.store.CreateLead(r.Context(), video.ID, req.SessionID, req.Email, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record lead")
		return
	}

	if video.WebhookURL != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = s.webhook.SendLead(ctx, video.WebhookURL, webhook.LeadPayload{
				Event:      "lead.captured",
				VideoID:    video.ID,
				SessionID:  lead.SessionID,
				Email:      lead.Email,
				Name:       lead.Name,
				CapturedAt: lead.CreatedAt.Format(time.RFC3339),
			})
		}()
	}

	writeJSON(w, http.StatusCreated, lead)
}

func (s *Server) handleEmbedCTAClick(w http.ResponseWriter, r *http.Request) {
	video, err := s.getPublicVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found or not ready")
		return
	}
	_ = video

	ctaID, err := strconv.ParseInt(chi.URLParam(r, "ctaID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid CTA id")
		return
	}

	var req struct {
		SessionID string `json:"sessionId"`
	}
	_ = decodeJSON(r, &req)

	if err := s.store.RecordCTAClick(r.Context(), ctaID, req.SessionID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record click")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
