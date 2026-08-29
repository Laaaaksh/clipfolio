package api

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dashboarddist
var dashboardEmbed embed.FS

//go:embed playerdist/player.js
var playerJS []byte

func (s *Server) handlePlayerJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(playerJS)
}

// dashboardHandler serves the built React dashboard as a single-page app:
// any path that isn't a real static asset falls back to index.html so
// client-side routing works on a hard refresh.
func (s *Server) dashboardHandler() http.Handler {
	sub, err := fs.Sub(dashboardEmbed, "dashboarddist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(sub, r.URL.Path[1:]); err != nil {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
