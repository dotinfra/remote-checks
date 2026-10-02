# AGENTS.md

Agent-facing context for working in this repository. Keep this file in sync with reality when the code changes.

## What this project is

`remote-checks` is a minimal Sinatra web service that performs remote HTTP checks on demand. It is a single-file app (`app/main.rb`) with no database, no background jobs, and no build step. The live service backs checks for dotINFRA.

- Rack endpoint: `config.ru`
- Application: `app/main.rb` (class `NetChecks < Sinatra::Base`, ~45 lines)
- Server stack: Sinatra + Thin (see `Gemfile`)
- Ruby version pinned to `2.7.0` in the `Gemfile` (EOL — see "Known issues")

## Endpoints

- `GET /` — 302 redirect to `https://www.dotinfra.fr`.
- `GET /checks?url=<target>[&proxy=<proxy-url>]` — performs an HTTP GET against `url` and returns a JSON report.

The JSON response has the shape:

```json
{
  "HTTP_Check": {
    "request_to": "<url>",
    "response_code": "<code>",
    "response_time": "<seconds>",
    "cachecontrol": "<cache-control header value>"
  }
}
```

Behavioral details an agent must know (verified in `app/main.rb`):

- Both `url` and `proxy` are passed to `URI(...)`. If `proxy` is omitted, `URI(nil)` raises, so **in practice `proxy` is required**.
- HTTP Basic Auth credentials embedded in the `url` (`https://user:pass@host`) are forwarded to the target request.
- `response_time` is a `Benchmark.realtime` wall-clock duration, serialized as a string.
- The `Cache-Control` header of the target response is read with `.first.to_s`; a response without that header yields an empty string (via `nil.to_s` on missing header — note `get_fields` returns `nil` when absent, which would raise `NoMethodError`).
- The target is always fetched with HTTP GET using User-Agent `Mozilla/5.0 (dotINFRA; remote check)`.
- There is no timeout handling and no error handling: an unreachable URL, an invalid URI, or a connection error will surface as a Sinatra 500.
- No TLS/HTTPS-specific configuration is done; `Net::HTTP` defaults apply.

## Commands

```sh
bundle install          # install dependencies (Gemfile.lock is committed)
bundle exec rackup      # run the app locally (config.ru, default port 9292)
bundle exec ruby -c app/main.rb   # syntax check only — there is no test suite
```

There is no test suite, no linter config, and no CI workflow in this repo. Verification is manual: run the app and curl the endpoints.

```sh
curl -i http://localhost:9292/                       # expect 302 to dotinfra.fr
curl -s 'http://localhost:9292/checks?url=https://example.com&proxy=http://localhost:3128'
```

## Repository conventions

- Commits are short imperative one-liners (`Update ruby & deps`, `Change wording`).
- Structure is flat: application code lives in `app/`, entry point is `config.ru`, version lives in `VERSION` (currently `0.1.0`).
- No comments policy beyond what exists; keep new code consistent with the current terse style.
- Dependencies are minimal (sinatra, thin, json). Do not add gems without a strong reason.

## Known issues (documented only — not yet fixed)

These are intentional follow-ups; do not "fix" them incidentally when asked to do something else:

1. `proxy` param is effectively mandatory; omitting it crashes with `URI(nil)` (ArgumentError).
2. No error handling for invalid/unreachable URLs, DNS failures, or timeouts — results in HTTP 500.
3. Missing `Cache-Control` header on the target response raises `NoMethodError` on `nil.first`.
4. Ruby pinned to 2.7.0 (EOL since 2023) and dated dependencies.
5. No tests, no CI.
6. Open proxy usage: any caller can make the service issue requests toward arbitrary targets via an arbitrary proxy (SSRF-ish surface), no rate limiting or allowlist.
