package secrettest

import (
	"errors"
	"fmt"
	"testing"
)

// recorder stands in for *testing.T so a test can observe a failure the
// helper reports without failing itself. The embedded interface stays nil:
// the helper calls only the three methods below.
type recorder struct {
	testing.TB

	failed bool
}

func (r *recorder) Helper()               {}
func (r *recorder) Errorf(string, ...any) { r.failed = true }
func (r *recorder) Fatal(...any)          { r.failed = true }

func TestChanceProne(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"101":                  true,
		"987654321":            true,
		"deadBEEF":             true,
		"0x1e8f5572e210":       true,
		"0x":                   true,
		"+101":                 false,
		"-1000000007":          false,
		"off3":                 false,
		"cursorA":              false,
		"fifty":                false,
		"12345678901234567890": false,
	}
	for secret, want := range tests {
		if got := ChanceProne(secret); got != want {
			t.Errorf("ChanceProne(%q) = %v, want %v", secret, got, want)
		}
	}
}

func TestAssertHiddenRefusesAChanceProneSecret(t *testing.T) {
	t.Parallel()

	r := &recorder{}
	AssertHidden(r, nil, "101")
	if !r.failed {
		t.Error("AssertHidden accepted a secret that can occur in a pointer address")
	}
}

func TestAssertHiddenStillFindsASentinel(t *testing.T) {
	t.Parallel()

	const sentinel = "-1000000007"
	r := &recorder{}
	AssertHidden(r, fmt.Errorf("feedconfig: value %s refused", sentinel), sentinel)
	if !r.failed {
		t.Error("AssertHidden missed a sentinel that is in the error text")
	}
	r = &recorder{}
	AssertHidden(r, errors.New("feedconfig: value refused"), sentinel)
	if r.failed {
		t.Error("AssertHidden failed on an error without the sentinel")
	}
}
