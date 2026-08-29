package api

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/laaaaksh/clipfolio/internal/db"
	"github.com/laaaaksh/clipfolio/internal/transcode"
)

// TranscodeJob is one unit of work for the background transcode workers: an
// uploaded source file, saved locally, waiting to become HLS.
type TranscodeJob struct {
	VideoID         int64
	LocalSourcePath string
}

// TranscodeQueue runs uploaded videos through ffmpeg on a small fixed pool of
// goroutines - no Redis or external job queue needed for a single-instance
// self-hosted deploy.
type TranscodeQueue struct {
	store   *db.Store
	storage ObjectStore
	jobs    chan TranscodeJob
}

// NewTranscodeQueue builds a queue ready to have Start called on it.
func NewTranscodeQueue(store *db.Store, storage ObjectStore) *TranscodeQueue {
	return &TranscodeQueue{
		store:   store,
		storage: storage,
		jobs:    make(chan TranscodeJob, 64),
	}
}

// Start launches the given number of worker goroutines, stopping them when
// ctx is done.
func (q *TranscodeQueue) Start(ctx context.Context, workers int) {
	if workers < 1 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		go q.worker(ctx)
	}
}

// Enqueue schedules job to run on the next free worker.
func (q *TranscodeQueue) Enqueue(job TranscodeJob) {
	q.jobs <- job
}

func (q *TranscodeQueue) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-q.jobs:
			if err := q.process(ctx, job); err != nil {
				log.Printf("transcode video %d failed: %v", job.VideoID, err)
				if markErr := q.store.MarkVideoFailed(ctx, job.VideoID, err.Error()); markErr != nil {
					log.Printf("failed to mark video %d as failed: %v", job.VideoID, markErr)
				}
			}
			_ = os.Remove(job.LocalSourcePath)
		}
	}
}

func (q *TranscodeQueue) process(ctx context.Context, job TranscodeJob) error {
	if !transcode.Available() {
		return fmt.Errorf("ffmpeg/ffprobe not installed on this host")
	}

	duration, height, err := transcode.Probe(ctx, job.LocalSourcePath)
	if err != nil {
		return fmt.Errorf("probe source: %w", err)
	}

	renditions := transcode.SelectRenditions(height, transcode.DefaultRenditions)

	outDir, err := os.MkdirTemp("", "clipfolio-hls-*")
	if err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(outDir) }()

	if err := transcode.ToHLS(ctx, job.LocalSourcePath, outDir, renditions); err != nil {
		return fmt.Errorf("transcode: %w", err)
	}

	thumbPath := filepath.Join(outDir, "thumbnail.jpg")
	thumbnailKey := ""
	if err := extractThumbnail(ctx, job.LocalSourcePath, thumbPath, duration); err != nil {
		log.Printf("thumbnail extraction failed for video %d: %v (continuing without one)", job.VideoID, err)
	} else {
		thumbnailKey, err = q.uploadFile(ctx, job.VideoID, thumbPath, "image/jpeg")
		if err != nil {
			log.Printf("thumbnail upload failed for video %d: %v (continuing without one)", job.VideoID, err)
			thumbnailKey = ""
		}
	}

	sourceData, err := os.ReadFile(job.LocalSourcePath)
	if err != nil {
		return fmt.Errorf("read source for upload: %w", err)
	}
	if err := q.storage.Put(ctx, videoStorageDir(job.VideoID)+"/source.mp4", sourceData, "video/mp4"); err != nil {
		return fmt.Errorf("upload source: %w", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		return fmt.Errorf("read output dir: %w", err)
	}
	var playlistKey string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "thumbnail.jpg" {
			continue
		}
		contentType := "application/octet-stream"
		switch filepath.Ext(entry.Name()) {
		case ".m3u8":
			contentType = "application/vnd.apple.mpegurl"
		case ".ts":
			contentType = "video/mp2t"
		}

		localPath := filepath.Join(outDir, entry.Name())
		key, err := q.uploadFile(ctx, job.VideoID, localPath, contentType)
		if err != nil {
			return fmt.Errorf("upload %s: %w", entry.Name(), err)
		}
		if entry.Name() == "master.m3u8" {
			playlistKey = key
		}
	}
	if playlistKey == "" {
		return fmt.Errorf("transcode produced no master playlist")
	}

	return q.store.MarkVideoReady(ctx, job.VideoID, duration, playlistKey, thumbnailKey)
}

func (q *TranscodeQueue) uploadFile(ctx context.Context, videoID int64, localPath, contentType string) (string, error) {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", localPath, err)
	}
	key := fmt.Sprintf("%s/%s", videoStorageDir(videoID), filepath.Base(localPath))
	if err := q.storage.Put(ctx, key, data, contentType); err != nil {
		return "", err
	}
	return key, nil
}

// extractThumbnail grabs a single frame roughly a quarter into the video (or
// at 1s for very short clips) as the video's cover image.
func extractThumbnail(ctx context.Context, inputPath, outputPath string, durationSeconds float64) error {
	seek := "1"
	if durationSeconds > 4 {
		seek = fmt.Sprintf("%.2f", durationSeconds/4)
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-ss", seek, "-i", inputPath, "-frames:v", "1", "-q:v", "3", outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg thumbnail: %w\n%s", err, out)
	}
	return nil
}
