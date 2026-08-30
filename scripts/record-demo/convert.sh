#!/usr/bin/env bash
# Converts the raw Playwright recording into the two assets the README ships:
# docs/assets/demo.mp4 (full quality) and docs/assets/demo.gif (embedded,
# must stay under 10MB - GitHub renders the GIF inline, not the MP4).
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

RAW="assets/video/raw.webm"
REPO_ROOT="../.."
MP4="$REPO_ROOT/docs/assets/demo.mp4"
GIF="$REPO_ROOT/docs/assets/demo.gif"
PALETTE="assets/video/palette.png"
GIF_FPS="${GIF_FPS:-12}"

if [ ! -f "$RAW" ]; then
  echo "missing $RAW - run \`npm run record\` first" >&2
  exit 1
fi

mkdir -p "$REPO_ROOT/docs/assets"

ffmpeg -y -i "$RAW" -vf "scale=1280:-2" -c:v libx264 -pix_fmt yuv420p -movflags +faststart -an "$MP4"

ffmpeg -y -i "$RAW" -vf "fps=${GIF_FPS},scale=960:-1:flags=lanczos,palettegen" -update 1 "$PALETTE"
ffmpeg -y -i "$RAW" -i "$PALETTE" -filter_complex "fps=${GIF_FPS},scale=960:-1:flags=lanczos[x];[x][1:v]paletteuse" "$GIF"

echo "mp4: $(du -h "$MP4" | cut -f1)"
echo "gif: $(du -h "$GIF" | cut -f1)"

GIF_BYTES=$(stat -f%z "$GIF" 2>/dev/null || stat -c%s "$GIF")
if [ "$GIF_BYTES" -gt 10485760 ]; then
  echo "ERROR: $GIF is $(($GIF_BYTES / 1024 / 1024))MB, over the 10MB budget - rerun with a lower GIF_FPS or shorten the recorded walkthrough" >&2
  exit 1
fi
