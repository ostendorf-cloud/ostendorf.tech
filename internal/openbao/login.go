package openbao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// redacted is what a Token prints as, whatever the verb.
const redacted = "[redacted]"

// Token is an OpenBao client token. It prints as "[redacted]" under every
// fmt verb (%v, %+v, %#v, %q, %s and the rest) and in a slog attribute, so a
// token that ends up in a format string, a struct dump or a log line by
// mistake still does not leak. The only way to the value is an explicit
// string conversion, which this package does in exactly one place: the
// X-Vault-Token request header.
type Token string

// Format implements fmt.Formatter for every verb.
func (Token) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(redacted)) }

// String implements fmt.Stringer.
func (Token) String() string { return redacted }

// GoString implements fmt.GoStringer.
func (Token) GoString() string { return redacted }

// LogValue implements slog.LogValuer.
func (Token) LogValue() slog.Value { return slog.StringValue(redacted) }

// MarshalJSON implements json.Marshaler, which slog's JSON handler uses for a
// struct holding a Token. A token is never sent as JSON on purpose.
func (Token) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }

// Auth is the result of one login: the client token and its lease.
type Auth struct {
	// Token is sent as X-Vault-Token on every request until it is replaced.
	Token Token
	// Lease is how long OpenBao granted the token for. Zero means no lease
	// was reported; see Static for what the client does then.
	Lease time.Duration
	// Static means logging in again cannot yield a different token, as with
	// a token handed to the process from outside. The client then never
	// renews it and treats a 403 as final instead of logging in again.
	Static bool
}

// Sender sends one unauthenticated request to OpenBao's HTTP API, path
// relative to /v1/, and returns the status and the body. A Login uses it so
// it shares the client's address, transport and timeout.
type Sender func(ctx context.Context, method, path string, body []byte) (status int, data []byte, err error)

// Login is one OpenBao auth method. Another method is one more type
// implementing this interface; the client itself does not change.
//
// An implementation must never put the token, or any credential it reads to
// obtain one, into a returned error.
type Login interface {
	Login(ctx context.Context, send Sender) (Auth, error)
}

// ErrNoToken means the static token method was configured without a token.
var ErrNoToken = errors.New("openbao: no token configured")

// StaticToken is the token auth method: a token obtained outside this
// process, for a local run against a development OpenBao. It is used as is,
// never renewed and never replaced: once it expires or is revoked, every
// request answers ErrForbidden until the process is given a new one.
type StaticToken struct {
	Token Token
}

// Login returns the configured token unchanged; it sends no request.
func (s StaticToken) Login(context.Context, Sender) (Auth, error) {
	if s.Token == "" {
		return Auth{}, ErrNoToken
	}
	return Auth{Token: s.Token, Static: true}, nil
}

// Kubernetes is the Kubernetes auth method: it logs in at AuthPath as Role
// with the service account token in JWTFile. The file is read fresh on every
// login, because the kubelet rotates a projected token in place.
type Kubernetes struct {
	AuthPath string
	Role     string
	JWTFile  string
}

type loginResponse struct {
	Auth struct {
		ClientToken   string `json:"client_token"`
		LeaseDuration int64  `json:"lease_duration"`
	} `json:"auth"`
}

// Login posts the service account token to auth/<AuthPath>/login.
func (k Kubernetes) Login(ctx context.Context, send Sender) (Auth, error) {
	jwt, err := os.ReadFile(k.JWTFile)
	if err != nil {
		return Auth{}, fmt.Errorf("openbao login: read service account token: %w", err)
	}
	body, err := json.Marshal(map[string]string{"role": k.Role, "jwt": strings.TrimSpace(string(jwt))})
	if err != nil {
		return Auth{}, fmt.Errorf("openbao login: %w", err)
	}
	status, data, err := send(ctx, http.MethodPost, "auth/"+strings.Trim(k.AuthPath, "/")+"/login", body)
	if err != nil {
		return Auth{}, fmt.Errorf("openbao login: %w", err)
	}
	if status != http.StatusOK {
		return Auth{}, fmt.Errorf("openbao login: unexpected status %d%s", status, apiErrors(data))
	}
	var resp loginResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return Auth{}, fmt.Errorf("openbao login: decode response: %w", err)
	}
	if resp.Auth.ClientToken == "" {
		return Auth{}, errors.New("openbao login: no client token in response")
	}
	return Auth{
		Token: Token(resp.Auth.ClientToken),
		Lease: time.Duration(resp.Auth.LeaseDuration) * time.Second,
	}, nil
}
