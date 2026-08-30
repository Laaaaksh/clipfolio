# record-demo

Records the README's `docs/assets/demo.mp4` / `demo.gif` from a real, running clipfolio instance.
Dev-only tooling - this package is never installed as part of the product build (`make build` and
`make frontend` never touch it).

It drives the actual dashboard UI with Playwright: create the admin account, create a video, upload
a generated test clip, wait for it to transcode, add a CTA and a lead-capture gate, play through the
real embed preview as a viewer (submit the lead form, hit the CTA), then dwell on the resulting
analytics and captured lead. Nothing is mocked or staged - every screen it visits is the real app
talking to real Postgres/MinIO.

## Requirements

Same as running clipfolio itself: Docker, and `ffmpeg`/`ffprobe` on `PATH` (used both to generate the
license-clean synthetic source clip and to convert the recording afterwards).

## Usage

One command from the repo root, which boots a fresh stack, records, converts, and tears the stack
back down:

```bash
make demo
```

To iterate on the recorder itself instead of the whole pipeline:

```bash
cd scripts/record-demo
npm install
npx playwright install chromium   # first time only

# with clipfolio already running at http://localhost:8080 (docker compose up --build)
npm run clip      # (re)generates assets/sample-clip.mp4 via ffmpeg - only needed once
npm run record    # drives the real UI, writes assets/video/raw.webm
./convert.sh       # writes ../../docs/assets/demo.mp4 and demo.gif
```

`npm run record` expects a **freshly seeded** stack - it creates the admin account, which only
succeeds once per database. Reset with `docker compose down -v && docker compose up -d --build`
between runs. Set `CLIPFOLIO_BASE_URL` to point at a different instance, or `HEADLESS=false` to
watch the browser while debugging.

## Tuning

- `record.mjs`'s `page.waitForTimeout(...)` calls exist purely for pacing - long enough for a human
  to read each screen, short enough to keep the whole walkthrough in the README's 45-90s budget.
  Adjust them there if a step feels rushed or dragged out.
- `convert.sh` accepts `GIF_FPS` (default 12) if the GIF needs to shrink further to stay under the
  10MB budget GitHub imposes on inline images - lower the fps or shorten the recording rather than
  raising the size limit.
