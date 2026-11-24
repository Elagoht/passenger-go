package web

import (
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/Elagoht/collage/pkg/collage"
)

// clientIP is the rate limiter's key for r.
func clientIP(r *http.Request) netip.Addr { return limiterKey(collage.ClientIP(r)) }

// limiterKey groups addresses the way one client holds them: an IPv4 address
// (IPv4-mapped IPv6 included) is its own key, while an IPv6 address counts as
// its /64, which a single host usually gets whole and can rotate through.
func limiterKey(ip netip.Addr) netip.Addr {
	ip = ip.Unmap().WithZone("")
	if !ip.Is6() {
		return ip
	}
	return netip.PrefixFrom(ip, 64).Masked().Addr()
}

// SecretJail is the elagoht/fail2ban jail every secret guess is struck in.
// Its limit (see plugins.go) is far above the backoff's, so it only bans a
// client that keeps guessing for a long time.
const SecretJail = "secret"

// Throttle is called before checking a guessed secret (passphrase, recovery
// key, setup code). When r's client is backing off it returns the message for
// a 429 page and false. Otherwise the attempt is recorded as a failure up
// front (see ratelimit.Limiter.Begin), the global delay is waited out, and it
// returns true; the caller reports a right guess with AttemptSucceeded. The
// error is the request's context ending during the delay.
//
// The backoff is ours, not elagoht/fail2ban's, although fail2ban has a strike
// API (Report): fail2ban checks its bans in a middleware when a request
// arrives and strikes after the fact, so guesses sent in parallel are all
// admitted before the first one is counted, where Begin checks and records
// under one lock; it answers a ban with a plain site-wide 403, where a
// reader backing off gets the form again with how many seconds to wait; it
// has no ceiling across all clients (GlobalDelay); and it does nothing in
// development mode. elagoht/ratelimit counts requests, not failures. So each
// attempt is also struck in fail2ban's SecretJail, as a long-horizon ban on
// top of this backoff, and forgiven there when it succeeds.
func (d *Deps) Throttle(r *http.Request) (refusal string, ok bool, err error) {
	if wait, ok := d.Limiter.Begin(clientIP(r)); !ok {
		return RetryMessage(wait), false, nil
	}
	if d.Bans != nil {
		d.Bans.Report(r, SecretJail)
	}
	if delay := d.Limiter.GlobalDelay(); delay > 0 {
		t := time.NewTimer(delay)
		defer t.Stop()
		select {
		case <-t.C:
		case <-r.Context().Done():
			return "", false, r.Context().Err()
		}
	}
	return "", true, nil
}

// AttemptSucceeded clears r's client's failures, including the one Throttle
// recorded for this attempt. Forgiving the fail2ban strikes is safe here, as
// its README asks: the vault has one secret of each kind, so a right guess
// proves the guesser is its owner.
func (d *Deps) AttemptSucceeded(r *http.Request) {
	d.Limiter.Succeed(clientIP(r))
	if d.Bans != nil {
		d.Bans.Forgive(r, SecretJail)
	}
}

// RetryMessage tells a client backing off how long to wait, in whole seconds
// rounded up.
func RetryMessage(wait time.Duration) string {
	n := (wait + time.Second - 1) / time.Second
	if n == 1 {
		return "Too many attempts. Try again in 1 second."
	}
	return fmt.Sprintf("Too many attempts. Try again in %d seconds.", n)
}

// CookieHeader is a response header setting cookies, for ActionResult.Header.
func CookieHeader(cookies ...*http.Cookie) http.Header {
	h := http.Header{}
	for _, c := range cookies {
		h.Add("Set-Cookie", c.String())
	}
	return h
}
