// Package client is a thin, hand-written HTTP client for the Coolify v4 REST API.
//
// The API is documented by an OpenAPI 3.1 document published at
// https://raw.githubusercontent.com/coollabsio/coolify/main/openapi.json and served
// interactively at <endpoint>/docs/api. Every endpoint lives under /api/v1 and is
// authenticated with a bearer token created in Coolify under
// "Keys & Tokens" > "API tokens".
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultEndpoint is Coolify Cloud. Self-hosted instances pass their own base URL.
	DefaultEndpoint = "https://app.coolify.io"

	// apiPrefix is appended to the endpoint. Coolify has only ever shipped v1.
	apiPrefix = "/api/v1"

	defaultTimeout = 60 * time.Second
	maxRetries     = 3
)

// Client talks to a single Coolify instance as a single API token.
type Client struct {
	baseURL    *url.URL
	token      string
	userAgent  string
	httpClient *http.Client
}

// Option customises a Client at construction time.
type Option func(*Client)

// WithHTTPClient replaces the underlying *http.Client. Used by tests and by the
// provider to inject a client with TLS verification disabled.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithUserAgent sets the User-Agent header sent on every request.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

// New builds a Client for the given endpoint. The endpoint is the instance root
// (for example https://coolify.example.com); /api/v1 is appended automatically,
// and is tolerated if the caller already included it.
func New(endpoint, token string, opts ...Option) (*Client, error) {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	if token == "" {
		return nil, errors.New("an API token is required")
	}

	trimmed := strings.TrimSuffix(strings.TrimSpace(endpoint), "/")
	trimmed = strings.TrimSuffix(trimmed, apiPrefix)

	u, err := url.Parse(trimmed + apiPrefix)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint %q: %w", endpoint, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid endpoint %q: scheme must be http or https", endpoint)
	}

	c := &Client{
		baseURL:    u,
		token:      token,
		userAgent:  "terraform-provider-coolify",
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// Endpoint returns the fully-qualified API base URL, for logging and diagnostics.
func (c *Client) Endpoint() string { return c.baseURL.String() }

// do performs an authenticated request. body is JSON-encoded when non-nil; out is
// JSON-decoded from the response when non-nil. A nil out discards the body.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
	}

	// Retries are only safe because every retried status (429, 5xx) means the
	// request was rejected before it took effect.
	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff(attempt, lastErr)):
			}
		}

		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}

		req, err := http.NewRequestWithContext(ctx, method, c.baseURL.String()+path, reader)
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Transport errors are not retried: they may have reached the server.
			return fmt.Errorf("%s %s: %w", method, path, err)
		}

		raw, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return fmt.Errorf("%s %s: reading response: %w", method, path, readErr)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out == nil || len(bytes.TrimSpace(raw)) == 0 {
				return nil
			}
			if err := json.Unmarshal(raw, out); err != nil {
				return fmt.Errorf("%s %s: decoding response: %w (body: %s)", method, path, err, truncate(raw))
			}
			return nil
		}

		apiErr := parseError(method, path, resp, raw)
		if !apiErr.Retryable() {
			return apiErr
		}
		lastErr = apiErr
	}

	return fmt.Errorf("%s %s: giving up after %d attempts: %w", method, path, maxRetries, lastErr)
}

// backoff waits progressively longer, honouring Retry-After on 429 responses.
func backoff(attempt int, lastErr error) time.Duration {
	var apiErr *Error
	if errors.As(lastErr, &apiErr) && apiErr.RetryAfter > 0 {
		return apiErr.RetryAfter
	}
	return time.Duration(1<<attempt) * time.Second
}

func truncate(b []byte) string {
	const limit = 512
	if len(b) <= limit {
		return string(b)
	}
	return string(b[:limit]) + "…"
}

func parseError(method, path string, resp *http.Response, raw []byte) *Error {
	apiErr := &Error{
		Method:     method,
		Path:       path,
		StatusCode: resp.StatusCode,
		Body:       truncate(raw),
	}

	var decoded struct {
		Message string              `json:"message"`
		Errors  map[string][]string `json:"errors"`
	}
	if err := json.Unmarshal(raw, &decoded); err == nil {
		apiErr.Message = decoded.Message
		apiErr.Validation = decoded.Errors
	}
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}

	if after := resp.Header.Get("Retry-After"); after != "" {
		if secs, err := strconv.Atoi(after); err == nil && secs > 0 {
			apiErr.RetryAfter = time.Duration(secs) * time.Second
		}
	}

	return apiErr
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

func (c *Client) patch(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPatch, path, body, out)
}

func (c *Client) delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// deleteWithQuery issues DELETE with query parameters (used by the resource
// deletion endpoints that accept cleanup flags).
func (c *Client) deleteWithQuery(ctx context.Context, path string, query url.Values) error {
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// waitForDeletion polls checkFn with exponential backoff (500ms → 5s, capped at
// deadline) until it reports 404. Coolify deletes resources asynchronously:
// the DELETE call only queues the removal, so a destroy immediately followed
// by a re-create can collide on names, domains or networks still held by the
// dying containers.
func (c *Client) waitForDeletion(ctx context.Context, deadline time.Duration, checkFn func(context.Context) error) error {
	waitCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	delay := 500 * time.Millisecond
	for {
		// Still-present (nil error) and transient errors both re-check on the
		// next tick; only a 404 means the teardown finished.
		if err := checkFn(waitCtx); IsNotFound(err) {
			return nil
		}

		select {
		case <-waitCtx.Done():
			return fmt.Errorf("resource still present after %s: deletion is queued but not finished", deadline)
		case <-time.After(delay):
		}
		if delay < 5*time.Second {
			delay *= 2
			if delay > 5*time.Second {
				delay = 5 * time.Second
			}
		}
	}
}

// boolQuery renders the standard cleanup flags shared by the application,
// database and service delete endpoints. Nil pointers keep the API defaults
// (all true).
func deletionQuery(deleteConfigurations, deleteVolumes, dockerCleanup, deleteConnectedNetworks *bool) url.Values {
	q := url.Values{}
	set := func(name string, v *bool) {
		if v != nil {
			q.Set(name, strconv.FormatBool(*v))
		}
	}
	set("delete_configurations", deleteConfigurations)
	set("delete_volumes", deleteVolumes)
	set("docker_cleanup", dockerCleanup)
	set("delete_connected_networks", deleteConnectedNetworks)
	return q
}
