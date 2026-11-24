// Package headertest holds the security headers the single website image
// sent before it was split into a frontend and an API image (#119), and the
// comparison both images' tests run against them. The values were captured
// from that image's handler, for a page, an API route and a 404 of each.
//
// What may differ: X-Request-Id is per request. In the policy, a 'nonce-'
// source is per response and a 'sha256-' source is the hash of the theme
// boot script, which was the site's own script and is govue's now; both are
// dropped before the comparison. The API serves no page, so its script-src
// carries no hash at all, which the normalisation also covers.
package headertest

import (
	"net/http"
	"slices"
	"strings"
	"testing"
)

// Policy is the Content-Security-Policy of every response of the former
// image, with its nonce and hash sources dropped.
const Policy = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; " +
	"font-src 'self'; connect-src 'self'; worker-src 'self'; form-action 'self'; " +
	"frame-ancestors 'none'; base-uri 'none'; object-src 'none'"

// Security is every security header of the former image but the policy,
// the same on a page, an API route and a 404.
var Security = map[string]string{
	"Cross-Origin-Opener-Policy":   "same-origin",
	"Cross-Origin-Resource-Policy": "same-origin",
	"Permissions-Policy": "accelerometer=(), browsing-topics=(), camera=(), display-capture=(), " +
		"geolocation=(), gyroscope=(), magnetometer=(), microphone=(), midi=(), payment=(), usb=()",
	"Referrer-Policy":           "strict-origin-when-cross-origin",
	"Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
	"X-Content-Type-Options":    "nosniff",
	"X-Frame-Options":           "DENY",
}

// NormalizePolicy drops the 'nonce-' and 'sha256-' sources of a policy.
func NormalizePolicy(policy string) string {
	directives := strings.Split(policy, ";")
	out := make([]string, 0, len(directives))
	for _, d := range directives {
		fields := slices.DeleteFunc(strings.Fields(d), func(f string) bool {
			return strings.HasPrefix(f, "'nonce-") || strings.HasPrefix(f, "'sha256-")
		})
		if len(fields) > 0 {
			out = append(out, strings.Join(fields, " "))
		}
	}
	return strings.Join(out, "; ")
}

// Equal fails t when h does not carry exactly the former security headers:
// every value of Security, the normalised Policy and a request id.
func Equal(t *testing.T, what string, h http.Header) {
	t.Helper()
	for name, want := range Security {
		if got := h.Values(name); len(got) != 1 || got[0] != want {
			t.Errorf("%s: %s = %q, want %q once", what, name, got, want)
		}
	}
	csp := h.Values("Content-Security-Policy")
	if len(csp) != 1 || NormalizePolicy(csp[0]) != Policy {
		t.Errorf("%s: Content-Security-Policy = %q, want %q after dropping nonce and hash sources",
			what, csp, Policy)
	}
	if h.Get("X-Request-Id") == "" {
		t.Errorf("%s: no X-Request-Id", what)
	}
}
