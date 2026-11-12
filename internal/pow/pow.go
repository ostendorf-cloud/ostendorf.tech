// Package pow issues and verifies ALTCHA style proof of work challenges: the
// browser searches the number n with sha256(salt + n) == challenge, and an
// HMAC over the challenge proves that the server issued it. The salt carries
// the expiry and the issue time, and the hash binds both to the challenge.
//
// Ported from old_simon-ostendorf.de's internal/pow, unchanged: the algorithm
// and the wire shape are a contract with the browser worker that solves it,
// and changing either here without changing the worker would just break the
// form.
package pow

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Algorithm is the only hash algorithm offered and accepted.
const Algorithm = "SHA-256"

var (
	// ErrInvalid covers a malformed payload, a wrong number and a bad signature.
	ErrInvalid = errors.New("pow: invalid solution")
	// ErrExpired means the challenge is past its expiry.
	ErrExpired = errors.New("pow: challenge expired")
	// ErrTooFast means the form was sent sooner than a person could fill it.
	ErrTooFast = errors.New("pow: submitted too fast")
	// ErrReplay means the solution was used before.
	ErrReplay = errors.New("pow: solution already used")
	// ErrBusy means the replay set is full; the caller should retry later.
	ErrBusy = errors.New("pow: too many solutions in flight")
)

// Challenge is what GET /api/v1/challenge returns.
type Challenge struct {
	Algorithm string `json:"algorithm"`
	Challenge string `json:"challenge"`
	MaxNumber int64  `json:"maxnumber"`
	Salt      string `json:"salt"`
	Signature string `json:"signature"`
}

// Solution is what the browser sends back with the form.
type Solution struct {
	Algorithm string `json:"algorithm"`
	Challenge string `json:"challenge"`
	Number    int64  `json:"number"`
	Salt      string `json:"salt"`
	Signature string `json:"signature"`
}

// Service issues challenges and remembers used ones until they expire.
type Service struct {
	// MaxNumber is the upper bound of the secret number; it sets the work.
	MaxNumber int64
	// TTL is how long a challenge stays valid.
	TTL time.Duration
	// MinAge is the minimum time between issue and submission.
	MinAge time.Duration
	// MaxUsed bounds the replay set.
	MaxUsed int
	// Now is the clock; tests replace it.
	Now func() time.Time
	// Rand is the randomness source; tests may replace it.
	Rand io.Reader

	mu   sync.Mutex
	used map[string]time.Time
}

// NewService returns a service with the given work and timing.
func NewService(maxNumber int64, ttl, minAge time.Duration) *Service {
	return &Service{
		MaxNumber: maxNumber,
		TTL:       ttl,
		MinAge:    minAge,
		MaxUsed:   100_000,
		Now:       time.Now,
		Rand:      rand.Reader,
		used:      map[string]time.Time{},
	}
}

// Hash returns hex(sha256(salt + number)), the value the browser searches for.
func Hash(salt string, number int64) string {
	sum := sha256.Sum256([]byte(salt + strconv.FormatInt(number, 10)))
	return hex.EncodeToString(sum[:])
}

// Sign returns hex(HMAC-SHA-256(key, challenge)).
func Sign(key []byte, challenge string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(challenge))
	return hex.EncodeToString(mac.Sum(nil))
}

// Create issues a new challenge signed with key.
func (s *Service) Create(key []byte) (Challenge, error) {
	if len(key) == 0 {
		return Challenge{}, errors.New("pow: no key")
	}
	var nonce [12]byte
	if _, err := io.ReadFull(s.Rand, nonce[:]); err != nil {
		return Challenge{}, fmt.Errorf("pow: salt: %w", err)
	}
	n, err := rand.Int(s.Rand, big.NewInt(s.MaxNumber+1))
	if err != nil {
		return Challenge{}, fmt.Errorf("pow: number: %w", err)
	}
	now := s.Now()
	salt := fmt.Sprintf("%x?expires=%d&issued=%d", nonce, now.Add(s.TTL).Unix(), now.Unix())
	challenge := Hash(salt, n.Int64())
	return Challenge{
		Algorithm: Algorithm,
		Challenge: challenge,
		MaxNumber: s.MaxNumber,
		Salt:      salt,
		Signature: Sign(key, challenge),
	}, nil
}

// saltTimes reads expires and issued from the salt's query part.
func saltTimes(salt string) (expires, issued time.Time, err error) {
	_, query, ok := strings.Cut(salt, "?")
	if !ok {
		return time.Time{}, time.Time{}, ErrInvalid
	}
	params, err := url.ParseQuery(query)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalid
	}
	exp, err1 := strconv.ParseInt(params.Get("expires"), 10, 64)
	iss, err2 := strconv.ParseInt(params.Get("issued"), 10, 64)
	if err1 != nil || err2 != nil {
		return time.Time{}, time.Time{}, ErrInvalid
	}
	return time.Unix(exp, 0), time.Unix(iss, 0), nil
}

func isHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// Verify checks sol against any of keys (the current key first, then older
// ones during a rotation) and marks it used. It is called only after the
// rest of the form is valid, so a typo does not burn the solution.
func (s *Service) Verify(keys [][]byte, sol Solution) error {
	if sol.Algorithm != Algorithm || !isHex64(sol.Challenge) || !isHex64(sol.Signature) ||
		len(sol.Salt) == 0 || len(sol.Salt) > 128 || sol.Number < 0 || sol.Number > s.MaxNumber {
		return ErrInvalid
	}
	signed := false
	for _, key := range keys {
		if len(key) > 0 && hmac.Equal([]byte(Sign(key, sol.Challenge)), []byte(sol.Signature)) {
			signed = true
			break
		}
	}
	if !signed {
		return ErrInvalid
	}
	if !hmac.Equal([]byte(Hash(sol.Salt, sol.Number)), []byte(sol.Challenge)) {
		return ErrInvalid
	}
	expires, issued, err := saltTimes(sol.Salt)
	if err != nil {
		return err
	}
	now := s.Now()
	if !now.Before(expires) {
		return ErrExpired
	}
	if now.Sub(issued) < s.MinAge {
		return ErrTooFast
	}
	return s.markUsed(sol.Challenge, expires, now)
}

func (s *Service) markUsed(challenge string, expires, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.used == nil {
		s.used = map[string]time.Time{}
	}
	if _, seen := s.used[challenge]; seen {
		return ErrReplay
	}
	if len(s.used) >= s.MaxUsed {
		for c, exp := range s.used {
			if !now.Before(exp) {
				delete(s.used, c)
			}
		}
		if len(s.used) >= s.MaxUsed {
			// Refusing is safe: forgetting an entry would allow a replay.
			return ErrBusy
		}
	}
	s.used[challenge] = expires
	return nil
}

// UsedCount returns the size of the replay set, for tests.
func (s *Service) UsedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.used)
}
