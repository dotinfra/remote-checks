package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestHandleRootRedirects(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handleRoot(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if got := rec.Header().Get("Location"); got != redirectTarget {
		t.Fatalf("Location = %q, want %q", got, redirectTarget)
	}
}

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok\n" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "ok\n")
	}
}

func TestHandleChecksMissingURL(t *testing.T) {
	s := &server{}
	req := httptest.NewRequest(http.MethodGet, "/checks", nil)
	rec := httptest.NewRecorder()

	s.handleChecks(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleChecksInvalidURL(t *testing.T) {
	s := &server{}
	for _, raw := range []string{"", "not-a-url", "ftp://", "://host"} {
		req := httptest.NewRequest(http.MethodGet, "/checks?url="+url.QueryEscape(raw), nil)
		rec := httptest.NewRecorder()
		s.handleChecks(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("url %q: status = %d, want %d", raw, rec.Code, http.StatusBadRequest)
		}
	}
}

func TestHandleChecksInvalidProxy(t *testing.T) {
	s := &server{}
	req := httptest.NewRequest(http.MethodGet, "/checks?url=https://example.com&proxy=%3A%2F%2F", nil)
	rec := httptest.NewRecorder()

	s.handleChecks(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCheckSuccess(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != userAgent {
			t.Errorf("User-Agent = %q, want %q", r.Header.Get("User-Agent"), userAgent)
		}
		w.Header().Set("Cache-Control", "max-age=604800")
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	s := &server{}
	targetURL, _ := url.Parse(target.URL)

	result, err := s.check(context.Background(), targetURL, nil)
	if err != nil {
		t.Fatalf("check() error: %v", err)
	}
	if result.ResponseCode != "200" {
		t.Fatalf("ResponseCode = %q, want \"200\"", result.ResponseCode)
	}
	if result.CacheControl != "max-age=604800" {
		t.Fatalf("CacheControl = %q, want %q", result.CacheControl, "max-age=604800")
	}
	if result.RequestTo != target.URL {
		t.Fatalf("RequestTo = %q, want %q", result.RequestTo, target.URL)
	}
	if _, err := time.ParseDuration(result.ResponseTime + "s"); err != nil {
		t.Errorf("ResponseTime %q not parseable as duration: %v", result.ResponseTime, err)
	}
}

func TestCheckMissingCacheControlHeader(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	s := &server{}
	targetURL, _ := url.Parse(target.URL)

	result, err := s.check(context.Background(), targetURL, nil)
	if err != nil {
		t.Fatalf("check() error: %v", err)
	}
	if result.CacheControl != "" {
		t.Fatalf("CacheControl = %q, want empty", result.CacheControl)
	}
}

func TestCheckBasicAuthForwarded(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "user" || password != "pass" {
			t.Errorf("basic auth = %q/%q ok=%v, want user/pass", username, password, ok)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	s := &server{}
	targetURL, _ := url.Parse(target.URL)
	targetURL.User = url.UserPassword("user", "pass")

	result, err := s.check(context.Background(), targetURL, nil)
	if err != nil {
		t.Fatalf("check() error: %v", err)
	}
	if result.ResponseCode != "200" {
		t.Fatalf("ResponseCode = %q, want 200", result.ResponseCode)
	}
}

func TestCheckUnreachableReturnsError(t *testing.T) {
	s := &server{}
	badURL, _ := url.Parse("http://127.0.0.1:1")

	if _, err := s.check(context.Background(), badURL, nil); err == nil {
		t.Fatal("check() with unreachable host: want error, got nil")
	}
}

func TestHandleChecksErrorJSONShape(t *testing.T) {
	s := &server{}
	req := httptest.NewRequest(http.MethodGet, "/checks?url=http://127.0.0.1:1", nil)
	rec := httptest.NewRecorder()

	s.handleChecks(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("error field empty: %v", body)
	}
}
