// Command clipfolio runs the clipfolio server: the admin dashboard, the
// public embed/analytics API, and the background video transcode workers,
// all in a single process.
package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/laaaaksh/clipfolio/internal/api"
	"github.com/laaaaksh/clipfolio/internal/config"
	"github.com/laaaaksh/clipfolio/internal/db"
	"github.com/laaaaksh/clipfolio/internal/storage"
)

const transcodeWorkers = 2

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	store, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		return err
	}

	objectStore, err := storage.New(ctx, cfg)
	if err != nil {
		return err
	}

	server := api.NewServer(store, objectStore, cfg)
	server.Start(ctx, transcodeWorkers)

	httpServer := &http.Server{
		Addr:    cfg.Addr,
		Handler: server.Router(),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	log.Printf("clipfolio listening on %s", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
