package transcode

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectRenditions_SkipsUpscaling(t *testing.T) {
	got := SelectRenditions(480, DefaultRenditions)
	if len(got) != 1 || got[0].Name != "360p" {
		t.Fatalf("expected only 360p for a 480p source, got %+v", got)
	}
}

func TestSelectRenditions_IncludesAllBelowSource(t *testing.T) {
	got := SelectRenditions(1080, DefaultRenditions)
	if len(got) != 3 {
		t.Fatalf("expected all 3 renditions for a 1080p source, got %+v", got)
	}
}

func TestSelectRenditions_TinySourceStillGetsOneRendition(t *testing.T) {
	got := SelectRenditions(144, DefaultRenditions)
	if len(got) != 1 || got[0].Name != "360p" {
		t.Fatalf("expected the smallest rendition as a fallback, got %+v", got)
	}
}

func TestBuildFFmpegArgs_OneMasterPlaylistNoUpscaleFilters(t *testing.T) {
	renditions := []Rendition{
		{Name: "360p", Height: 360, VideoBitrate: "800k", AudioBitrate: "96k"},
		{Name: "720p", Height: 720, VideoBitrate: "2800k", AudioBitrate: "128k"},
	}
	args := BuildFFmpegArgs("in.mp4", "/tmp/out", renditions)
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "scale=-2:360") {
		t.Errorf("expected a scale filter for 360p, got: %s", joined)
	}
	if !strings.Contains(joined, "scale=-2:720") {
		t.Errorf("expected a scale filter for 720p, got: %s", joined)
	}
	if !strings.Contains(joined, "-master_pl_name master.m3u8") {
		t.Errorf("expected a master playlist name, got: %s", joined)
	}
	if !strings.Contains(joined, "v:0,a:0,name:360p") || !strings.Contains(joined, "v:1,a:1,name:720p") {
		t.Errorf("expected var_stream_map entries for both renditions, got: %s", joined)
	}
}

func TestAvailable(t *testing.T) {
	// Sanity-check the detection function against whatever is actually
	// installed - it should never panic, and should agree with exec.LookPath.
	_, wantFFmpeg := exec.LookPath("ffmpeg")
	got := Available()
	if wantFFmpeg == nil && !got {
		t.Fatalf("ffmpeg is on PATH but Available() returned false")
	}
}

// TestToHLS_RealFFmpeg exercises the actual ffmpeg pipeline end-to-end
// against a tiny generated test clip. It's skipped when ffmpeg isn't
// installed, mirroring how a self-hoster's CI might not have it either.
func TestToHLS_RealFFmpeg(t *testing.T) {
	if !Available() {
		t.Skip("ffmpeg/ffprobe not on PATH")
	}

	dir := t.TempDir()
	input := filepath.Join(dir, "in.mp4")

	// Generate a 2-second synthetic clip (color bars + a tone) instead of
	// committing a binary fixture to the repo.
	genCmd := exec.Command("ffmpeg", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=15",
		"-f", "lavfi", "-i", "sine=frequency=1000:duration=2",
		"-c:v", "libx264", "-c:a", "aac", "-shortest",
		input,
	)
	if out, err := genCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate test clip: %v\n%s", err, out)
	}

	duration, height, err := Probe(context.Background(), input)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if duration < 1.5 || duration > 2.5 {
		t.Fatalf("expected ~2s duration, got %v", duration)
	}
	if height != 240 {
		t.Fatalf("expected height 240, got %v", height)
	}

	renditions := SelectRenditions(height, DefaultRenditions)
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := ToHLS(context.Background(), input, outDir, renditions); err != nil {
		t.Fatalf("ToHLS: %v", err)
	}

	master := filepath.Join(outDir, "master.m3u8")
	if _, err := os.Stat(master); err != nil {
		t.Fatalf("expected master playlist at %s: %v", master, err)
	}
	variant := filepath.Join(outDir, renditions[0].Name+".m3u8")
	if _, err := os.Stat(variant); err != nil {
		t.Fatalf("expected variant playlist at %s: %v", variant, err)
	}
}
