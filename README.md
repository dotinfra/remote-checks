# remote-checks

A tiny Go web service that performs remote HTTP checks on demand and reports the result as JSON. Built and operated by [dotINFRA](https://www.dotinfra.fr).

Go implementation, standard library only — no framework, no external dependencies. Single static binary.

## Features

- `GET /checks?url=<target>[&proxy=<proxy-url>]` — fetches `<target>` over HTTP(S) and returns a JSON report with the response code, response time, and the target's `Cache-Control` header.
- `GET /` — redirects to [dotinfra.fr](https://www.dotinfra.fr).
- Optional outbound proxy: pass `proxy=http://host:port` to route the check through a proxy.
- Supports HTTP Basic Authentication via credentials embedded in the target URL (`https://user:pass@example.com/`).
- 10-second request timeout with clean JSON error responses.

## Requirements

- Go 1.24+

## Usage

Run locally:

```sh
go run ./cmd/server
```

The server listens on `:8080` by default (override with the `PORT` environment variable).

Run a check:

```sh
curl 'http://localhost:8080/checks?url=https://example.com'
```

Example response:

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

### Parameters

| Parameter | Required | Description                                          |
|-----------|----------|------------------------------------------------------|
| `url`     | yes      | Target URL to check. May embed Basic Auth credentials. |
| `proxy`   | no       | Proxy URL used for the outbound request.             |

### Errors

Errors are returned as `{"error": "<message>"}`:

| Status | Meaning                                            |
|--------|----------------------------------------------------|
| `400`  | Missing or invalid `url` / `proxy` parameter.        |
| `502`  | Target unreachable or the request failed.           |
| `504`  | The request timed out (10s limit).                  |

## Build

```sh
CGO_ENABLED=0 go build -ldflags="-s -w" -o remote-checks ./cmd/server
```

## Docker

```sh
docker build -t remote-checks .
docker run -p 8080:8080 remote-checks
```

The final image is `FROM scratch` (static binary), suitable for serverless containers (Scaleway, OVHcloud) and minimal deployments.

## Tests

```sh
go test ./...
```

## License

None specified.
