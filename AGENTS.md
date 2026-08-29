# Project agent memory

This file is the project's committed home for project-intrinsic agent knowledge: build, test, release, architecture, and sharp-edge notes that should travel with the code.

## Architecture

Single Go binary (`cmd/clipfolio`) embeds the built React dashboard (`internal/api/dashboarddist`)
and the vanilla-TS embeddable player bundle (`internal/api/playerdist`) via `go:embed`. Postgres for
metadata, any S3-compatible bucket (via the `ObjectStore` interface in `internal/api`) for video
assets - see README's Configuration section for the env vars. `ffmpeg`/`ffprobe` are shelled out to
directly (`internal/transcode`), not vendored.

## Build

`make frontend` builds the dashboard (Vite) and player (esbuild) into `internal/api/{dashboarddist,playerdist}`
- both are gitignored generated artifacts. `make build`/`make test`/`make lint` depend on
`ensure-frontend-stub` instead, which creates a minimal placeholder there ONLY if nothing exists yet
(so plain Go work never needs Node installed). Run `make frontend` yourself before testing any actual
dashboard/player behavior in a browser - the stub is a "run `make frontend`" placeholder, not a real UI.

## Testing

`internal/db` and `internal/api` are real integration tests against Postgres (`internal/testutil.OpenTestStore`),
skipped when `CLIPFOLIO_TEST_DATABASE_URL` is unset. They truncate shared tables at the start of each
test, so **always run with `-p 1`** (`make test` already does) - running those two packages
concurrently corrupts each other's fixtures via the same database.

`internal/transcode` has a real end-to-end test that shells out to `ffmpeg` to generate a synthetic
clip and transcode it - skipped if `ffmpeg`/`ffprobe` aren't on `PATH`.

## Sharp edges found by dogfooding (all now covered by regression tests - don't reintroduce)

- **Every `internal/db` model needs an explicit camelCase `json:` tag.** A struct field with no tag
  serializes as Go's PascalCase default, which the dashboard and the embeddable player (both written
  assuming camelCase, the normal JS/JSON convention) silently read as `undefined`. This exact bug
  shipped once: CTAs and the lead gate never fired in the real player because `cta.trigger`/
  `gate.enabled` were `undefined` client-side. Guarded by `TestJSONFieldsAreCamelCase` in
  `internal/api`.
- **A Go list handler must never return a nil slice.** `encoding/json` marshals `nil` as `null`, not
  `[]`, and the dashboard's `.length` calls on a `null` response crash the whole page (no error
  boundary saved it before one was added - see `components/ErrorBoundary.tsx`). Every `List*` method
  in `internal/db` initializes with `:= []T{}`, not `var x []T`. Guarded by
  `TestListEndpoints_EmptyListsAreJSONArraysNotNull`.
- **A child React component that saves its own state must call an `onChange` prop back to its
  parent**, or the parent's copy of that state goes stale forever after the first edit. This bit the
  lead-gate editor specifically: the dashboard's live preview is `key`-ed on `ctas.length` +
  `leadGate.*` so editing either remounts the embed with a fresh manifest fetch, but the lead gate's
  edits weren't propagated up, so the preview kept playing against the *original* (disabled) gate no
  matter what was saved. Both `CTAManager` and `LeadGateEditor` take an `onChange` for this reason.
- **Chrome's native `<video controls>` does not reliably start playback from a synthetic
  `element.click()`** (headless or not) - it does from a focused element + a `Space` keypress. Only
  matters for browser-automation/demo scripts, not real users.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
