// Package transcode shells out to ffmpeg/ffprobe to turn an uploaded source
// video into adaptive-bitrate HLS: one variant playlist per rendition plus a
// master playlist, ready to upload to object storage and stream with any
// HLS-capable player.
package transcode

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Rendition is one HLS output quality level.
type Rendition struct {
	Name         string // e.g. "360p", used in the output filename
	Height       int    // target vertical resolution
	VideoBitrate string // ffmpeg -b:v value, e.g. "800k"
	AudioBitrate string // ffmpeg -b:a value, e.g. "96k"
}

// DefaultRenditions are ordered lowest to highest quality. A source video
// narrower than a rendition's height is skipped by SelectRenditions rather
// than upscaled.
var DefaultRenditions = []Rendition{
	{Name: "360p", Height: 360, VideoBitrate: "800k", AudioBitrate: "96k"},
	{Name: "720p", Height: 720, VideoBitrate: "2800k", AudioBitrate: "128k"},
	{Name: "1080p", Height: 1080, VideoBitrate: "5000k", AudioBitrate: "192k"},
}

// SelectRenditions returns the renditions that don't upscale past the
// source's own height. If the source is smaller than every candidate, the
// single smallest rendition is used so there is always at least one output.
func SelectRenditions(sourceHeight int, candidates []Rendition) []Rendition {
	var selected []Rendition
	for _, r := range candidates {
		if r.Height <= sourceHeight {
			selected = append(selected, r)
		}
	}
	if len(selected) == 0 && len(candidates) > 0 {
		selected = []Rendition{candidates[0]}
	}
	return selected
}

// BuildFFmpegArgs constructs the ffmpeg argument list for producing one HLS
// variant per rendition plus a master playlist named "master.m3u8" in
// outputDir. Kept as a pure function (no process execution) so the command
// shape is unit-testable without ffmpeg installed.
func BuildFFmpegArgs(inputPath, outputDir string, renditions []Rendition) []string {
	args := []string{"-y", "-i", inputPath}

	// One -map pair per rendition so a single ffmpeg invocation produces every
	// variant in one pass instead of re-reading the source file per rendition.
	var varStreamMap []string
	for i, r := range renditions {
		args = append(args,
			"-map", "0:v:0", "-map", "0:a:0",
			fmt.Sprintf("-filter:v:%d", i), fmt.Sprintf("scale=-2:%d", r.Height),
			fmt.Sprintf("-b:v:%d", i), r.VideoBitrate,
			fmt.Sprintf("-b:a:%d", i), r.AudioBitrate,
		)
		varStreamMap = append(varStreamMap, fmt.Sprintf("v:%d,a:%d,name:%s", i, i, r.Name))
	}

	args = append(args,
		"-c:v", "h264", "-c:a", "aac",
		"-f", "hls",
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", outputDir+"/%v_%03d.ts",
		"-master_pl_name", "master.m3u8",
		"-var_stream_map", strings.Join(varStreamMap, " "),
		outputDir+"/%v.m3u8",
	)
	return args
}

// ToHLS runs ffmpeg to produce HLS output for every rendition into outputDir.
func ToHLS(ctx context.Context, inputPath, outputDir string, renditions []Rendition) error {
	if len(renditions) == 0 {
		return fmt.Errorf("no renditions selected")
	}
	args := BuildFFmpegArgs(inputPath, outputDir, renditions)
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w\n%s", err, truncate(string(out), 4000))
	}
	return nil
}

// Probe returns the source video's duration in seconds and vertical
// resolution in pixels, via ffprobe.
func Probe(ctx context.Context, inputPath string) (durationSeconds float64, height int, err error) {
	durArgs := []string{"-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", inputPath}
	durOut, err := exec.CommandContext(ctx, "ffprobe", durArgs...).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe duration: %w", err)
	}
	durationSeconds, err = strconv.ParseFloat(strings.TrimSpace(string(durOut)), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse duration %q: %w", durOut, err)
	}

	heightArgs := []string{"-v", "error", "-select_streams", "v:0", "-show_entries", "stream=height", "-of", "default=noprint_wrappers=1:nokey=1", inputPath}
	heightOut, err := exec.CommandContext(ctx, "ffprobe", heightArgs...).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe height: %w", err)
	}
	height, err = strconv.Atoi(strings.TrimSpace(string(heightOut)))
	if err != nil {
		return 0, 0, fmt.Errorf("parse height %q: %w", heightOut, err)
	}

	return durationSeconds, height, nil
}

// Available reports whether ffmpeg and ffprobe are on PATH.
func Available() bool {
	_, ffmpegErr := exec.LookPath("ffmpeg")
	_, ffprobeErr := exec.LookPath("ffprobe")
	return ffmpegErr == nil && ffprobeErr == nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
