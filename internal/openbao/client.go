// Package openbao talks to OpenBao over its HTTP API for the feed service:
// logging in through a Login (the Kubernetes auth method in a cluster, a
// static token for a local run) and reading and writing one KV v2 secret.
// It needs no SDK, so the image this builds into stays standard library
// only.
//
// No token ever travels into an error, a log line or a formatted value: it
// is held as a Token, which prints as "[redacted]", and converted to a plain
// string only for the X-Vault-Token header.
package openbao

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// maxResponseBytes bounds every response; secrets and login answers are small.
	maxResponseBytes = 1 << 20
	// assumedTTL is the lease OpenBao is assumed to have granted when a
	// login answer carries no lease duration at all.
	assumedTTL = 90 * time.Minute
)

var (
	// ErrNotFound means the KV path holds no secret, including a version
	// that was deleted: OpenBao answers both a missing path and a deleted
	// version with data the caller cannot tell apart without this.
	ErrNotFound = errors.New("openbao: secret not found")
	// ErrCASMismatch means another writer changed the secret since it was read.
	ErrCASMismatch = errors.New("openbao: check-and-set version mismatch")
	// ErrForbidden means OpenBao refused a static token with 403: it expired,
	// was revoked or lacks the policy. It carries neither the token nor
	// OpenBao's response body, and the request is not retried, since logging
	// in again cannot yield a different static token.
	ErrForbidden = errors.New("openbao: permission denied for the configured token")
)

// Secret is the data of one KV v2 secret and its version.
type Secret struct {
	Data    map[string]any
	Version int
}

// String returns a string property, or "" when it is missing or not a string.
func (s Secret) String(key string) string {
	v, _ := s.Data[key].(string)
	return v
}

// Client logs in through a Login and keeps the token fresh.
//
// Everything that identifies it to OpenBao, the address and the login
// method with its credentials, is fixed at construction and never exported:
// the cached token and its renewal time depend on both, so letting a caller
// change one later would risk a token built for one identity being reused
// under another. http and now stay package private too; a test in this
// package reaches them directly instead of through an exported seam the
// rest of the world could also reach for.
//
// Client prints as its address only (see Format), so a Client in a format
// string or a struct dump never shows the cached token.
type Client struct {
	addr   string
	login  Login
	http   *http.Client
	logger *slog.Logger
	now    func() time.Time

	mu      sync.Mutex
	token   Token
	static  bool
	renewAt time.Time
}

// New returns a client for addr that authenticates through login, with a 10
// second per-request timeout. logger is the caller's own, never built here.
func New(addr string, login Login, logger *slog.Logger) *Client {
	return &Client{
		addr:   strings.TrimRight(addr, "/"),
		login:  login,
		http:   &http.Client{Timeout: 10 * time.Second},
		logger: logger,
		now:    time.Now,
	}
}

// Format implements fmt.Formatter: a Client prints as its address only,
// under every verb, never with its cached token.
func (c *Client) Format(f fmt.State, _ rune) {
	_, _ = fmt.Fprintf(f, "openbao.Client(%s)", c.addr)
}

// currentToken returns a valid token, logging in first when there is none or
// when two thirds of its TTL have passed. Logging in again instead of
// renewing also survives a token that reached its max TTL. A static token is
// never renewed: it is returned as is for the life of the process.
func (c *Client) currentToken(ctx context.Context) (Token, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && (c.static || c.now().Before(c.renewAt)) {
		return c.token, c.static, nil
	}
	return c.loginLocked(ctx)
}

// invalidate drops token unless another request already replaced it.
func (c *Client) invalidate(token Token) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == token {
		c.token = ""
	}
}

// loginLocked authenticates and caches the result. Callers hold mu.
func (c *Client) loginLocked(ctx context.Context) (Token, bool, error) {
	send := func(ctx context.Context, method, path string, body []byte) (int, []byte, error) {
		return c.send(ctx, method, path, "", body)
	}
	auth, err := c.login.Login(ctx, send)
	if err != nil {
		return "", false, err
	}
	if auth.Token == "" {
		return "", false, errors.New("openbao login: no client token")
	}
	c.token, c.static = auth.Token, auth.Static
	if auth.Static {
		c.logger.Info("openbao: using the configured token, it is never renewed")
		return c.token, true, nil
	}
	ttl := auth.Lease
	if ttl <= 0 {
		// A token without a lease duration never expires on its own;
		// still log in again on the same schedule an hour and a half
		// lease would give, rather than trusting it forever.
		ttl = assumedTTL
	}
	c.renewAt = c.now().Add(ttl * 2 / 3)
	c.logger.Info("openbao: logged in", "lease", ttl.String())
	return c.token, false, nil
}

// send performs one request and returns its status and body. The token
// travels only in a header, never in a logged or returned string.
func (c *Client) send(ctx context.Context, method, path string, token Token, body []byte) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.addr+"/v1/"+path, reader)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		// The one place a token becomes a plain string.
		req.Header.Set("X-Vault-Token", string(token))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, data, nil
}

// do sends an authenticated request. With a login method that can issue a
// new token, a 403 means the token was revoked or expired early, so it logs
// in once more and retries; a second 403 is returned to the caller rather
// than retried again. A static token cannot be replaced by logging in, so
// its 403 is returned at once as ErrForbidden, with no retry and without
// OpenBao's response body.
func (c *Client) do(ctx context.Context, method, path string, body []byte) (int, []byte, error) {
	for attempt := 0; ; attempt++ {
		token, static, err := c.currentToken(ctx)
		if err != nil {
			return 0, nil, err
		}
		status, data, err := c.send(ctx, method, path, token, body)
		if err != nil {
			return 0, nil, err
		}
		if status == http.StatusForbidden && static {
			return 0, nil, ErrForbidden
		}
		if status == http.StatusForbidden && attempt == 0 {
			c.logger.Warn("openbao: request rejected, logging in again", "method", method, "path", path)
			c.invalidate(token)
			continue
		}
		return status, data, nil
	}
}

func cleanPath(path string) (string, error) {
	path = strings.Trim(path, "/")
	if path == "" || strings.Contains(path, "..") {
		return "", fmt.Errorf("openbao: invalid path %q", path)
	}
	return path, nil
}

type readResponse struct {
	Data struct {
		Data     map[string]any `json:"data"`
		Metadata struct {
			Version int `json:"version"`
		} `json:"metadata"`
	} `json:"data"`
}

// Read returns the KV v2 secret at path, a full data path such as
// mount/data/name.
func (c *Client) Read(ctx context.Context, path string) (Secret, error) {
	path, err := cleanPath(path)
	if err != nil {
		return Secret{}, err
	}
	status, data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return Secret{}, fmt.Errorf("openbao read %s: %w", path, err)
	}
	switch status {
	case http.StatusOK:
	case http.StatusNotFound:
		return Secret{}, fmt.Errorf("openbao read %s: %w", path, ErrNotFound)
	default:
		return Secret{}, fmt.Errorf("openbao read %s: unexpected status %d%s", path, status, apiErrors(data))
	}
	var resp readResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return Secret{}, fmt.Errorf("openbao read %s: decode response: %w", path, err)
	}
	if resp.Data.Data == nil {
		// A deleted version answers with data null, the same shape a
		// missing path would have to the caller.
		return Secret{}, fmt.Errorf("openbao read %s: %w", path, ErrNotFound)
	}
	return Secret{Data: resp.Data.Data, Version: resp.Data.Metadata.Version}, nil
}

// Write replaces the KV v2 secret at path. A cas of 0 or more is sent as the
// check-and-set version, so a concurrent writer makes it fail with
// ErrCASMismatch; a negative cas omits the option and writes unconditionally.
func (c *Client) Write(ctx context.Context, path string, values map[string]any, cas int) error {
	path, err := cleanPath(path)
	if err != nil {
		return err
	}
	payload := map[string]any{"data": values}
	if cas >= 0 {
		payload["options"] = map[string]int{"cas": cas}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("openbao write %s: %w", path, err)
	}
	status, data, err := c.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return fmt.Errorf("openbao write %s: %w", path, err)
	}
	switch {
	case status == http.StatusOK || status == http.StatusNoContent:
		return nil
	case status == http.StatusBadRequest && strings.Contains(string(data), "check-and-set"):
		return fmt.Errorf("openbao write %s: %w", path, ErrCASMismatch)
	default:
		return fmt.Errorf("openbao write %s: unexpected status %d%s", path, status, apiErrors(data))
	}
}

// apiErrors extracts OpenBao's own error list, which never holds a secret
// value, only names or statuses. Capped, since it travels into an error
// string this package returns.
func apiErrors(data []byte) string {
	var resp struct {
		Errors []string `json:"errors"`
	}
	if json.Unmarshal(data, &resp) != nil || len(resp.Errors) == 0 {
		return ""
	}
	msg := strings.Join(resp.Errors, "; ")
	if len(msg) > 300 {
		msg = msg[:300]
	}
	return ": " + msg
}
