# Contributing to clipfolio

Thank you for your interest in contributing. clipfolio is self-hosted business video hosting with
analytics, open source under the MIT license.

## Getting started

```bash
git clone https://github.com/<your-username>/clipfolio.git   # your fork, see below
cd clipfolio
go mod download
make frontend   # builds the dashboard and embeddable player (needs Node 20+)
make build
```

## Requirements

- Go 1.26+
- Node 20+ (only needed to build/change the dashboard or embeddable player)
- `ffmpeg` and `ffprobe` on your `PATH` - required for the transcode-pipeline tests and for running
  the server for real; unit tests that don't touch ffmpeg still pass without it
- Docker, to run Postgres for the integration tests (see below) and to run the full stack via
  `docker compose up`

## Contribution workflow

The `master` branch is protected: every change lands through a pull request, required status checks
must pass, and protection is enforced for everyone - including the maintainer. There are no direct
pushes to `master`.

1. Fork the repo on GitHub, then clone your fork (command above).
2. Create a descriptively named feature branch from `master`.
3. Make your changes as small, focused commits, each leaving the tree buildable.
4. Run `make lint` and `make test` - both must pass. See below for the database `make test` needs.
5. If your change is user-facing (a feature, fix, or behavior change), add one bullet under the
   `Unreleased` heading in [CHANGELOG.md](CHANGELOG.md).
6. Push the branch to your fork.
7. Open a pull request against `master` here.

A PR can merge only when the `Test` and `Lint` checks pass and all conversation threads are
resolved.

### Running the integration tests

`internal/db` and `internal/api` run real integration tests against Postgres (no mocked database
layer) - they're skipped automatically when `CLIPFOLIO_TEST_DATABASE_URL` isn't set, so a plain
`go test ./...` still passes without Postgres running. To run them:

```bash
docker run -d --name clipfolio-test-pg -e POSTGRES_PASSWORD=postgres -p 5433:5432 postgres:17-alpine
CLIPFOLIO_TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable" make test
```

`make test` always runs with `-p 1` (packages sequential, not parallel) - `internal/db` and
`internal/api` both truncate the same shared test database between test runs, and running those
packages concurrently corrupts each other's fixtures.

### Frontend testing (known gap)

`web/dashboard` and `web/player` - the React dashboard and the embeddable player that render the
CTA overlay and lead-capture gate a viewer actually sees - have no automated tests and no `test`
script in either `package.json`. The 1,107 lines of Go tests above cover the backend the frontend
talks to, but not the frontend's own rendering logic. Until that's addressed, verify frontend
changes manually against the flow in [the demo section below](#reproducing-the-readme-demo).

### Reproducing the README demo

The screenshot and GIF in the README are real captures, not mockups. To reproduce them yourself:

```bash
docker compose up --build
```

Then create the admin account, upload any short video, add a CTA and a lead-capture gate from the
video's page, and play it back through the embed preview - the dashboard's analytics update live as
you watch.

## Releases

Releases are cut by pushing a tag; GitHub Actions does the rest (`.github/workflows/release.yml`):

1. Make sure every user-facing change since the last release has a bullet under `Unreleased` in
   [CHANGELOG.md](CHANGELOG.md) (step 5 of the workflow above).
2. Give the release its own changelog section: insert `## [x.y.z] - YYYY-MM-DD` above the (now
   empty) `## [Unreleased]` heading, following the format of the existing sections, and update the
   compare links at the bottom of the file - add
   `[x.y.z]: https://github.com/Laaaaksh/clipfolio/compare/v<prev>...vx.y.z` and repoint
   `[Unreleased]` at `compare/vx.y.z...HEAD`.
3. Land those changelog edits on `master` through a pull request (see the contribution workflow
   above), then tag and push:

   ```bash
   git tag vx.y.z && git push origin vx.y.z
   ```

The release workflow extracts the tagged version's CHANGELOG section as the GitHub release notes and
fails rather than publishing empty notes if that section is missing. It also builds and pushes the
`clipfolio` Docker image tagged with the version.

## Code style

- Standard `gofmt` formatting (enforced by CI); TypeScript formatted per the dashboard/player's own
  `tsc` strict-mode checks (also enforced by CI).
- Package layout: HTTP handlers and routing in `internal/api`, database access in `internal/db`,
  pure business logic (analytics math, transcode command construction) kept dependency-free and
  unit-testable in their own packages (`internal/analytics`, `internal/transcode`).
- Every exported identifier gets a doc comment (enforced by `golangci-lint`'s `revive` rule) -
  explain *why* it exists or a non-obvious constraint, not just what the name already says.
- The Go API's JSON field names are camelCase by explicit `json:` tags on every `internal/db`
  model - never add a field without one; a missing tag falls back to Go's PascalCase default and
  silently breaks every JS/TS consumer (the dashboard and the embeddable player). See
  `TestJSONFieldsAreCamelCase` in `internal/api` for the regression test this guards.

## Reporting issues

Please open a GitHub issue before starting large changes or proposing new features, so scope and
approach can be settled before code is written. Bug reports should include:
- clipfolio version/commit
- How you're running it (Docker Compose, source build) and which storage backend
- Steps to reproduce
- What you expected vs what happened
