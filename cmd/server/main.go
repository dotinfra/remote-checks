// Package main implements the remote-checks HTTP service: it performs
// remote HTTP checks on demand and reports the result as JSON.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	defaultTimeout = 10 * time.Second
	userAgent      = "Mozilla/5.0 (dotINFRA; remote check)"
	redirectTarget = "https://www.dotinfra.fr"
)

type checkResult struct {
	RequestTo    string `json:"request_to"`
	ResponseCode string `json:"response_code"`
	ResponseTime string `json:"response_time"`
	CacheControl string `json:"cachecontrol"`
}

type httpCheck struct {
	HTTPCheck checkResult `json:"HTTP_Check"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *server) handleChecks(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	rawProxy := r.URL.Query().Get("proxy")

	if rawURL == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter: url")
		return
	}
	target, err := url.Parse(rawURL)
	if err != nil || target.Scheme == "" || target.Host == "" {
		writeError(w, http.StatusBadRequest, "invalid url parameter")
		return
	}

	var proxyURL *url.URL
	if rawProxy != "" {
		proxyURL, err = url.Parse(rawProxy)
		if err != nil || proxyURL.Host == "" {
			writeError(w, http.StatusBadRequest, "invalid proxy parameter")
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultTimeout)
	defer cancel()

	result, err := s.check(ctx, target, proxyURL)
	if err != nil {
		var statusCode int
		if errors.Is(err, context.DeadlineExceeded) {
			statusCode = http.StatusGatewayTimeout
			err = fmt.Errorf("request timed out after %s", defaultTimeout)
		} else {
			statusCode = http.StatusBadGateway
		}
		writeError(w, statusCode, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, httpCheck{HTTPCheck: result})
}

// check performs a GET against target, optionally through proxy, and
// measures the wall-clock duration of the exchange.
func (s *server) check(ctx context.Context, target *url.URL, proxy *url.URL) (checkResult, error) {
	client := s.client(proxy)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return checkResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	if target.User != nil {
		password, _ := target.User.Password()
		req.SetBasicAuth(target.User.Username(), password)
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return checkResult{}, err
	}
	defer resp.Body.Close()

	return checkResult{
		RequestTo:    target.String(),
		ResponseCode: fmt.Sprintf("%d", resp.StatusCode),
		ResponseTime: fmt.Sprintf("%f", elapsed.Seconds()),
		CacheControl: resp.Header.Get("Cache-Control"),
	}, nil
}

// client returns an HTTP client configured with the outbound proxy and
// sensible defaults. Transport is not shared across requests because the
// proxy may change from one request to the next.
func (s *server) client(proxy *url.URL) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	if proxy != nil {
		transport.Proxy = http.ProxyURL(proxy)
	}
	return &http.Client{Transport: transport, Timeout: defaultTimeout}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, redirectTarget, http.StatusFound)
}

type server struct{}

func main() {
	s := &server{}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleRoot)
	mux.HandleFunc("GET /checks", s.handleChecks)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("remote-checks listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}
