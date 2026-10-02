# remote-checks

A tiny Sinatra web service that performs remote HTTP checks on demand and reports the result as JSON. Built and operated by [dotINFRA](https://www.dotinfra.fr).

## Features

- `GET /checks?url=<target>&proxy=<proxy-url>` — fetches `<target>` over HTTP(S) and returns a JSON report with the response code, total response time, and the target's `Cache-Control` header.
- `GET /` — redirects to [dotinfra.fr](https://www.dotinfra.fr).
- Supports HTTP Basic Authentication via credentials embedded in the target URL (`https://user:pass@example.com/`).
- Optional outbound proxy: pass `proxy=http://host:port` to route the check through a proxy.

## Requirements

- Ruby 2.7.0 (as pinned in the `Gemfile`)
- Bundler

Dependencies: `sinatra`, `thin`, `json`.

## Installation

```sh
bundle install
```

## Usage

Start the server:

```sh
bundle exec rackup
```

Run a check (through a proxy):

```sh
curl 'http://localhost:9292/checks?url=https://example.com&proxy=http://localhost:3128'
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

The service always issues a `GET` request with the User-Agent `Mozilla/5.0 (dotINFRA; remote check)`.

### Parameters

| Parameter | Description |
|-----------|-------------|
| `url`     | Target URL to check. May embed Basic Auth credentials. |
| `proxy`   | Proxy URL used for the outbound request (e.g. `http://host:port`). |

## API

### `GET /checks`

Returns `application/json`:

| Field           | Description                                          |
|-----------------|------------------------------------------------------|
| `request_to`    | The requested URL.                                    |
| `response_code` | HTTP status code returned by the target.              |
| `response_time` | Wall-clock request duration in seconds (as string).  |
| `cachecontrol`  | `Cache-Control` header of the target response.       |

Errors (invalid URL, unreachable host, missing proxy) currently surface as an HTTP 500.

## License

None specified.
