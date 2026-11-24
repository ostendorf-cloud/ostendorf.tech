// Package secrettest holds test helpers that prove a secret stays out of an
// error or a formatted value. It is imported by tests only. It follows
// govue's internal configtext/configtexttest.AssertHidden, which this
// module cannot import.
package secrettest

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// verbs are the formats a value can reach a log line, an error or a test
// failure message through.
var verbs = []string{"%v", "%+v", "%#v", "%q", "%s"}

// maxAddressText is the length of the longest pointer text %#v prints:
// "0x" and sixteen hex digits.
const maxAddressText = 18

// ChanceProne reports whether secret can occur inside a pointer address that
// %v, %+v or %#v prints, such as (*errors.joinError)(0x1e8f5572e210). Such a
// value makes a test flaky, because the address changes on every run. It is
// true for a short value made only of hex digits, with or without a leading
// "0x". A value with any other character, or a longer one, never occurs by
// chance.
func ChanceProne(secret string) bool {
	if len(secret) > maxAddressText {
		return false
	}
	digits := strings.TrimPrefix(secret, "0x")
	if digits == "" && secret != "" {
		return true
	}
	for _, r := range digits {
		isHex := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
		if !isHex {
			return false
		}
	}
	return true
}

// AssertHidden fails t when secret appears in err or in any error it wraps,
// through errors.Unwrap and Unwrap() []error alike, under every verb above.
// A nil err passes. The failure message never quotes the secret or the
// error text: it names only the verb and the depth. A secret that
// ChanceProne reports is refused, because its absence cannot be proven
// deterministically.
func AssertHidden(t testing.TB, err error, secret string) {
	t.Helper()
	if secret == "" {
		t.Fatal("secrettest: AssertHidden needs a non-empty secret")
	}
	if ChanceProne(secret) {
		t.Fatal("secrettest: AssertHidden needs a secret that cannot occur in a pointer address: add a non-hex character or make it longer than 18")
	}
	walk(t, err, secret, 0)
}

func walk(t testing.TB, err error, secret string, depth int) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("secrettest: the secret appears in Error() of an error at depth %d (%T)", depth, err)
	}
	for _, verb := range verbs {
		if strings.Contains(fmt.Sprintf(verb, err), secret) {
			t.Errorf("secrettest: the secret appears under %s of an error at depth %d (%T)", verb, depth, err)
		}
	}
	switch e := err.(type) { //nolint:errorlint // walking the tree by hand is the point
	case interface{ Unwrap() []error }:
		for _, inner := range e.Unwrap() {
			walk(t, inner, secret, depth+1)
		}
	default:
		walk(t, errors.Unwrap(err), secret, depth+1)
	}
}

// AssertFormatHidden fails t when secret appears in v under any verb above.
// The failure message never quotes the secret or the formatted text.
func AssertFormatHidden(t testing.TB, v any, secret string) {
	t.Helper()
	if secret == "" {
		t.Fatal("secrettest: AssertFormatHidden needs a non-empty secret")
	}
	if ChanceProne(secret) {
		t.Fatal("secrettest: AssertFormatHidden needs a secret that cannot occur in a pointer address: add a non-hex character or make it longer than 18")
	}
	for _, verb := range verbs {
		if strings.Contains(fmt.Sprintf(verb, v), secret) {
			t.Errorf("secrettest: the secret appears under %s of a %T", verb, v)
		}
	}
}
