# AGENTS.md

Agent-facing context for working in this repository. Keep this file in sync with reality when the code changes.

## What this project is

`remote-checks` is a minimal HTTP web service that performs remote HTTP checks on demand and reports the result as JSON. It backs checks for dotINFRA.

The service was migrated from Ruby/Sinatra to Go; the legacy Ruby implementation has been removed.

- **Go (canonical)**: `cmd/server/main.go` — stdlib only (`net/http`, `encoding/json`), no framework, no external dependencies. `go.mod` pins `go 1.24`.

## Endpoints (Go implementation)

- `GET /` — 302 redirect to `https://www.dotinfra.fr`.
- `GET /healthz` — 200 health probe (used by CI smoke test and deployment platforms).
- `GET /checks?url=<target>[&proxy=<proxy-url>]` — performs an HTTP GET against `url` and returns a JSON report.

Success response:

```json
{
  "HTTP_Check": {
    "request_to": "https://example.com",
    "response_code": "200",
    "response_time": "0.184213",
    "cachecontrol": "max-age=604800"
  }
}
```

Error responses are `{"error": "<message>"}` with a meaningful status code:

- `400` — missing/invalid `url` or `proxy` parameter
- `502` — target unreachable or request failed
- `504` — request timed out (10s limit)

Behavioral details an agent must know (verified in `cmd/server/main.go`):

- `url` is required and must have a scheme and host; `proxy` is optional.
- HTTP Basic Auth credentials embedded in `url` are forwarded to the target.
- A missing `Cache-Control` header yields an empty string.
- Requests are issued with User-Agent `Mozilla/5.0 (dotINFRA; remote check)` and carry a 10s overall timeout plus a 5s TLS handshake timeout.
- `response_time` is the wall-clock duration of `client.Do`, serialized as a fractional-seconds string.
- Query routing uses Go 1.22+ method patterns (`GET /checks`); other methods get 405.

## Commands

```sh
go build ./...     # build
go vet ./...       # static analysis
gofmt -l .         # formatting check
go test ./...      # run tests (httptest-based, no network needed)
go run ./cmd/server            # run locally, listens on :8080 (override with PORT)
CGO_ENABLED=0 go build -ldflags="-s -w" -o remote-checks ./cmd/server   # static binary
docker build -t remote-checks .   # image FROM scratch (final binary is static)
```

CI workflow: `.github/workflows/ci.yml` (fmt, vet, tests, static build, Docker build, container smoke test) on every PR and push to master. Deploy workflow: `.github/workflows/deploy.yml` (build/push GHCR + Scaleway Serverless Container rollout) on push to master — see `DEPLOYMENT.md` for setup and secrets (`SCW_SECRET_KEY`, `SCW_CONTAINER_ID`, optional `SCW_REGION`). Locally, still run vet + gofmt + test before pushing.

## Repository conventions

- Commits are short imperative one-liners.
- Go code: stdlib only — do not add external modules without a strong reason. Keep the terse style; no comments except doc comments on exported/non-obvious helpers.
- Tests live next to the code (`main_test.go`), use `httptest`, and must not require network access.

## Known issues (documented only — not yet fixed)

1. Open proxy usage: any caller can make the service issue requests toward arbitrary targets via an arbitrary proxy (SSRF-ish surface), no rate limiting or allowlist.
2. No graceful shutdown handling (`http.ErrServerClosed` is tolerated but SIGTERM is not trapped).
3. Deploy workflow cannot run until the one-time Scaleway setup in `DEPLOYMENT.md` is done (namespace, container, GitHub secrets).

## History

This service was originally Ruby/Sinatra (`NetChecks < Sinatra::Base`, Thin server, Ruby 2.7.0). It was rewritten in Go and the Ruby implementation removed; the rewrite fixed several legacy bugs (mandatory `proxy` crashing with `URI(nil)`, `NoMethodError` on missing `Cache-Control`, no error handling/timeouts).
