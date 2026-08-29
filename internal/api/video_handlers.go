package api

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/laaaaksh/clipfolio/internal/db"
)

// maxUploadSize caps a single video upload at 4GiB - generous for a business
// demo/onboarding video, small enough to fail fast on an obvious mistake
// (someone pointing this at a movie file) rather than exhausting disk.
const maxUploadSize = 4 << 30

func videoIDFromRequest(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "videoID"), 10, 64)
}

func (s *Server) handleListVideos(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	videos, err := s.store.ListVideos(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list videos")
		return
	}
	writeJSON(w, http.StatusOK, videos)
}

func (s *Server) handleCreateVideo(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())

	var req struct {
		Title string `json:"title"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	video, err := s.store.CreateVideo(r.Context(), user.ID, req.Title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create video")
		return
	}
	writeJSON(w, http.StatusCreated, video)
}

func (s *Server) getOwnedVideo(r *http.Request) (db.Video, error) {
	user := userFromContext(r.Context())
	id, err := videoIDFromRequest(r)
	if err != nil {
		return db.Video{}, db.ErrNotFound
	}
	video, err := s.store.GetVideo(r.Context(), id)
	if err != nil {
		return db.Video{}, err
	}
	if video.OwnerID != user.ID {
		return db.Video{}, db.ErrNotFound
	}
	return video, nil
}

func (s *Server) handleGetVideo(w http.ResponseWriter, r *http.Request) {
	video, err := s.getOwnedVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found")
		return
	}
	writeJSON(w, http.StatusOK, video)
}

func (s *Server) handleDeleteVideo(w http.ResponseWriter, r *http.Request) {
	video, err := s.getOwnedVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found")
		return
	}
	if err := s.storage.DeletePrefix(r.Context(), videoStorageDir(video.ID)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete video assets")
		return
	}
	if err := s.store.DeleteVideo(r.Context(), video.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete video")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUploadVideo(w http.ResponseWriter, r *http.Request) {
	video, err := s.getOwnedVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found")
		return
	}
	if video.Status != db.VideoStatusUploading {
		writeError(w, http.StatusConflict, "video has already been uploaded")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or malformed upload")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer func() { _ = file.Close() }()

	tmp, err := os.CreateTemp("", "clipfolio-upload-*.mp4")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to buffer upload")
		return
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, file); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		writeError(w, http.StatusInternalServerError, "failed to save upload")
		return
	}
	_ = tmp.Close()

	if err := s.store.SetVideoSourceKey(r.Context(), video.ID, videoStorageDir(video.ID)+"/source.mp4"); err != nil {
		_ = os.Remove(tmpPath)
		writeError(w, http.StatusInternalServerError, "failed to record upload")
		return
	}

	s.jobs.Enqueue(TranscodeJob{VideoID: video.ID, LocalSourcePath: tmpPath})

	writeJSON(w, http.StatusAccepted, map[string]string{"status": db.VideoStatusTranscoding})
}

func (s *Server) handleVideoAnalytics(w http.ResponseWriter, r *http.Request) {
	video, err := s.getOwnedVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found")
		return
	}
	summary, err := s.computeAnalytics(r.Context(), video)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to compute analytics")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleListLeads(w http.ResponseWriter, r *http.Request) {
	video, err := s.getOwnedVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found")
		return
	}
	leads, err := s.store.ListLeads(r.Context(), video.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list leads")
		return
	}
	writeJSON(w, http.StatusOK, leads)
}

func (s *Server) handleSetWebhook(w http.ResponseWriter, r *http.Request) {
	video, err := s.getOwnedVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found")
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := s.store.SetVideoWebhook(r.Context(), video.ID, req.URL); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to set webhook")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListCTAs(w http.ResponseWriter, r *http.Request) {
	video, err := s.getOwnedVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found")
		return
	}
	ctas, err := s.store.ListCTAs(r.Context(), video.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list CTAs")
		return
	}
	writeJSON(w, http.StatusOK, ctas)
}

func (s *Server) handleCreateCTA(w http.ResponseWriter, r *http.Request) {
	video, err := s.getOwnedVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found")
		return
	}

	var req struct {
		Trigger          string  `json:"trigger"`
		TimestampSeconds float64 `json:"timestampSeconds"`
		Label            string  `json:"label"`
		URL              string  `json:"url"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Trigger != db.CTATriggerTimestamp && req.Trigger != db.CTATriggerEnd {
		writeError(w, http.StatusBadRequest, "trigger must be 'timestamp' or 'end'")
		return
	}
	if req.Label == "" || req.URL == "" {
		writeError(w, http.StatusBadRequest, "label and url are required")
		return
	}

	cta, err := s.store.CreateCTA(r.Context(), video.ID, req.Trigger, req.TimestampSeconds, req.Label, req.URL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create CTA")
		return
	}
	writeJSON(w, http.StatusCreated, cta)
}

func (s *Server) handleDeleteCTA(w http.ResponseWriter, r *http.Request) {
	video, err := s.getOwnedVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found")
		return
	}
	ctaID, err := strconv.ParseInt(chi.URLParam(r, "ctaID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid CTA id")
		return
	}
	if err := s.store.DeleteCTA(r.Context(), video.ID, ctaID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "CTA not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete CTA")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetLeadGate(w http.ResponseWriter, r *http.Request) {
	video, err := s.getOwnedVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found")
		return
	}
	gate, err := s.store.GetLeadGate(r.Context(), video.ID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "failed to load lead gate")
		return
	}
	writeJSON(w, http.StatusOK, gate)
}

func (s *Server) handleSetLeadGate(w http.ResponseWriter, r *http.Request) {
	video, err := s.getOwnedVideo(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "video not found")
		return
	}

	var req db.LeadGate
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Position != db.LeadGatePositionBefore && req.Position != db.LeadGatePositionTimestamp {
		writeError(w, http.StatusBadRequest, "position must be 'before' or 'timestamp'")
		return
	}
	req.VideoID = video.ID

	if err := s.store.UpsertLeadGate(r.Context(), req); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save lead gate")
		return
	}
	writeJSON(w, http.StatusOK, req)
}
