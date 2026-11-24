package web

import (
	"net/netip"
	"testing"
	"time"

	"passenger-go/internal/ratelimit"
)

// admitted is how many guesses the backoff lets one client make within d,
// guessing again the moment each wait ends and never guessing right: the
// most strikes SecretJail can see from it in d. Both are counted on one
// clock, the limiter's.
func admitted(d time.Duration) int {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := now.Add(d)
	l := ratelimit.New(func() time.Time { return now })
	ip := netip.MustParseAddr("192.0.2.1")
	n := 0
	for !now.After(end) {
		wait, ok := l.Begin(ip)
		if ok {
			n++
			continue
		}
		now = now.Add(wait)
	}
	return n
}

// SecretJail must be able to fire: a client the backoff admits as fast as it
// allows reaches MaxRetry strikes within FindTime. And it must be far above
// the backoff: an hour of guessing does not reach it.
func TestSecretJailIsReachable(t *testing.T) {
	j := SecretJailSettings
	window := time.Duration(j.FindTime)
	if n := admitted(window); n < j.MaxRetry {
		t.Fatalf("the backoff admits %d guesses in FindTime %v, under MaxRetry %d: the jail can never fire", n, window, j.MaxRetry)
	}
	if n := admitted(time.Hour); n >= j.MaxRetry {
		t.Fatalf("the backoff admits %d guesses in an hour, MaxRetry %d: the jail bans a mere typist", n, j.MaxRetry)
	}
}
