// Package api wires clipfolio's HTTP surface: admin dashboard endpoints
// (auth-gated) and viewer-facing embed endpoints (public, rate-limit-free by
// design since a lead-capture form or CTA click must never require login).
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/laaaaksh/clipfolio/internal/config"
	"github.com/laaaaksh/clipfolio/internal/db"
	"github.com/laaaaksh/clipfolio/internal/webhook"
)

// ObjectStore is the subset of storage.Store the API and transcode queue
// depend on. Declared here (consumer side) so tests can substitute an
// in-memory fake instead of standing up a real S3-compatible bucket.
type ObjectStore interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Get(ctx context.Context, key string) ([]byte, error)
	DeletePrefix(ctx context.Context, prefix string) error
	PublicURL(key string) string
}

// Server holds clipfolio's shared dependencies and builds its HTTP router.
type Server struct {
	store   *db.Store
	storage ObjectStore
	webhook *webhook.Sender
	cfg     config.Config
	jobs    *TranscodeQueue
}

// NewServer builds a Server ready to have Start and Router called on it.
func NewServer(store *db.Store, objectStore ObjectStore, cfg config.Config) *Server {
	s := &Server{
		store:   store,
		storage: objectStore,
		webhook: webhook.NewSender(),
		cfg:     cfg,
	}
	s.jobs = NewTranscodeQueue(store, objectStore)
	return s
}

// Start launches the background transcode worker pool. Call once before
// serving traffic.
func (s *Server) Start(ctx context.Context, workers int) {
	s.jobs.Start(ctx, workers)
}

// Router builds clipfolio's full HTTP handler: dashboard API, public embed
// API, the embeddable player script, and the dashboard SPA itself.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/api", func(r chi.Router) {
		r.Get("/setup", s.handleSetupStatus)
		r.Post("/setup", s.handleSetup)
		r.Post("/login", s.handleLogin)
		r.Post("/logout", s.handleLogout)

		// Public, viewer-facing embed endpoints - no auth, hit from any site
		// embedding a clipfolio player.
		r.Route("/embed/{videoID}", func(r chi.Router) {
			r.Get("/", s.handleEmbedManifest)
			r.Post("/progress", s.handleEmbedProgress)
			r.Post("/leads", s.handleEmbedLead)
			r.Post("/cta-click/{ctaID}", s.handleEmbedCTAClick)
		})

		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Get("/me", s.handleMe)

			r.Route("/videos", func(r chi.Router) {
				r.Get("/", s.handleListVideos)
				r.Post("/", s.handleCreateVideo)
				r.Route("/{videoID}", func(r chi.Router) {
					r.Get("/", s.handleGetVideo)
					r.Delete("/", s.handleDeleteVideo)
					r.Post("/upload", s.handleUploadVideo)
					r.Get("/analytics", s.handleVideoAnalytics)
					r.Get("/leads", s.handleListLeads)
					r.Put("/webhook", s.handleSetWebhook)

					r.Route("/ctas", func(r chi.Router) {
						r.Get("/", s.handleListCTAs)
						r.Post("/", s.handleCreateCTA)
						r.Delete("/{ctaID}", s.handleDeleteCTA)
					})

					r.Route("/lead-gate", func(r chi.Router) {
						r.Get("/", s.handleGetLeadGate)
						r.Put("/", s.handleSetLeadGate)
					})
				})
			})
		})
	})

	r.Get("/player.js", s.handlePlayerJS)
	r.Handle("/*", s.dashboardHandler())

	return r
}
