package pow

import (
	"errors"
	"testing"
	"time"
)

var key = []byte("test-key")

func newTestService(now *time.Time) *Service {
	s := NewService(1000, 30*time.Minute, 3*time.Second)
	s.Now = func() time.Time { return *now }
	return s
}

// solve finds the number the way the browser worker does.
func solve(t *testing.T, c Challenge) Solution {
	t.Helper()
	for n := int64(0); n <= c.MaxNumber; n++ {
		if Hash(c.Salt, n) == c.Challenge {
			return Solution{Algorithm: c.Algorithm, Challenge: c.Challenge, Number: n, Salt: c.Salt, Signature: c.Signature}
		}
	}
	t.Fatal("no solution found")
	return Solution{}
}

// TestKnownVector pins the algorithm; the browser worker's own test vectors
// use the same values.
func TestKnownVector(t *testing.T) {
	t.Parallel()

	salt := "00112233445566778899aabb?expires=1790000000&issued=1789998200"
	if got := Hash(salt, 4242); got != knownChallenge {
		t.Errorf("Hash = %s, want %s", got, knownChallenge)
	}
	if got := Sign([]byte("test-key"), knownChallenge); got != knownSignature {
		t.Errorf("Sign = %s, want %s", got, knownSignature)
	}
}

func TestValidSolution(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	s := newTestService(&now)
	c, err := s.Create(key)
	if err != nil {
		t.Fatal(err)
	}
	if c.Algorithm != "SHA-256" || c.MaxNumber != 1000 || len(c.Challenge) != 64 || len(c.Signature) != 64 {
		t.Fatalf("challenge = %+v", c)
	}
	sol := solve(t, c)
	now = now.Add(5 * time.Second)
	if err := s.Verify([][]byte{key}, sol); err != nil {
		t.Fatalf("valid solution rejected: %v", err)
	}
	if s.UsedCount() != 1 {
		t.Errorf("used = %d", s.UsedCount())
	}
}

func TestRejections(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		after  time.Duration
		keys   [][]byte
		mutate func(*Solution)
		want   error
	}{
		{"wrong number", 5 * time.Second, [][]byte{key}, func(s *Solution) { s.Number = (s.Number + 1) % 1001 }, ErrInvalid},
		{"negative number", 5 * time.Second, [][]byte{key}, func(s *Solution) { s.Number = -1 }, ErrInvalid},
		{"bad signature", 5 * time.Second, [][]byte{key}, func(s *Solution) { s.Signature = Sign([]byte("other"), s.Challenge) }, ErrInvalid},
		{"other key", 5 * time.Second, [][]byte{[]byte("rotated")}, func(*Solution) {}, ErrInvalid},
		{"tampered salt", 5 * time.Second, [][]byte{key}, func(s *Solution) { s.Salt += "0" }, ErrInvalid},
		{"wrong algorithm", 5 * time.Second, [][]byte{key}, func(s *Solution) { s.Algorithm = "SHA-1" }, ErrInvalid},
		{"expired", 31 * time.Minute, [][]byte{key}, func(*Solution) {}, ErrExpired},
		{"too fast", time.Second, [][]byte{key}, func(*Solution) {}, ErrTooFast},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			now := start
			s := newTestService(&now)
			c, err := s.Create(key)
			if err != nil {
				t.Fatal(err)
			}
			sol := solve(t, c)
			tc.mutate(&sol)
			now = now.Add(tc.after)
			if err := s.Verify(tc.keys, sol); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if s.UsedCount() != 0 {
				t.Error("rejected solution was marked used")
			}
		})
	}
}

func TestPreviousKeyAccepted(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	s := newTestService(&now)
	c, _ := s.Create(key)
	sol := solve(t, c)
	now = now.Add(time.Minute)
	if err := s.Verify([][]byte{[]byte("new-key"), key}, sol); err != nil {
		t.Fatalf("solution signed with the previous key rejected: %v", err)
	}
}

func TestReplay(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	s := newTestService(&now)
	c, _ := s.Create(key)
	sol := solve(t, c)
	now = now.Add(10 * time.Second)
	if err := s.Verify([][]byte{key}, sol); err != nil {
		t.Fatal(err)
	}
	if err := s.Verify([][]byte{key}, sol); !errors.Is(err, ErrReplay) {
		t.Fatalf("replay: err = %v, want ErrReplay", err)
	}
}

func TestReplaySetBounded(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	s := newTestService(&now)
	s.MaxUsed = 2
	var sols []Solution
	for range 3 {
		c, _ := s.Create(key)
		sols = append(sols, solve(t, c))
	}
	now = now.Add(10 * time.Second)
	for _, sol := range sols[:2] {
		if err := s.Verify([][]byte{key}, sol); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Verify([][]byte{key}, sols[2]); !errors.Is(err, ErrBusy) {
		t.Fatalf("full set: err = %v, want ErrBusy", err)
	}

	// Once the first two expired, their entries make room again.
	now = now.Add(31 * time.Minute)
	c, _ := s.Create(key)
	fresh := solve(t, c)
	now = now.Add(10 * time.Second)
	if err := s.Verify([][]byte{key}, fresh); err != nil {
		t.Fatalf("after expiry: %v", err)
	}
	if s.UsedCount() != 1 {
		t.Errorf("used = %d, want 1 after pruning", s.UsedCount())
	}
}

func TestCreateNeedsKey(t *testing.T) {
	t.Parallel()

	now := time.Now()
	if _, err := newTestService(&now).Create(nil); err == nil {
		t.Fatal("challenge created without a key")
	}
}
