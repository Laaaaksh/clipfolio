# Security Policy

## Supported versions

clipfolio is a young project. Security fixes are made against the **latest
release** and `main` only.

| Version        | Supported |
| -------------- | --------- |
| latest release | yes       |
| older releases | no        |

## Reporting a vulnerability

Please do **not** open a public GitHub issue for anything you believe is a
security problem.

Use GitHub's private vulnerability reporting instead:

> https://github.com/Laaaaksh/clipfolio/security/advisories/new

That link reaches the maintainer privately - the report, follow-up discussion,
and any fix coordination stay confidential until a patched release ships.

When reporting, please include:

- The affected version/commit
- How you're running clipfolio (Docker Compose, source build, which storage backend)
- Clear steps to reproduce

## What belongs in a report

clipfolio is a self-hosted server that accepts video uploads, shells out to
`ffmpeg`/`ffprobe`, stores objects in an S3-compatible bucket, and serves a
public, unauthenticated embed API. Things worth reporting:

- A path where an uploaded file, its filename, or its metadata reaches an
  `ffmpeg`/`ffprobe` invocation (or any other subprocess) in a way that could
  execute unintended commands or read/write files outside the intended
  temp/storage paths.
- Session or authentication bypass on the dashboard API (`/api/videos/*` and
  friends) - anything that lets a request act as an admin without a valid
  session cookie.
- A way for the **public, unauthenticated** embed API (`/api/embed/*`) to read
  or modify data belonging to a video other than the one addressed by its ID,
  or to affect the dashboard's authenticated surface.
- Stored XSS via any field a viewer or admin can set (video titles, CTA
  labels/URLs, lead names) that renders unescaped in the dashboard or the
  embeddable player.
- Object storage credentials or the database connection string leaking into a
  response, log line, or client-visible payload.

Out of scope:

- Missing rate limiting on the public embed endpoints - documented as a known
  gap, not a vulnerability, until v1's threat model changes.
- Issues that require an attacker who already has valid dashboard admin
  credentials or direct database/bucket access - that's already full trust.
- Vulnerabilities in `ffmpeg`, Postgres, or your S3-compatible storage
  provider itself - please report those upstream.

## Credits

Reporters who wish to be credited in a fix's release notes may say so in the
private report; otherwise reports are handled without attribution.
