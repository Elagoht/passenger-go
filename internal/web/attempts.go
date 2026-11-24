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

// Throttle is called before checking a guessed secret (passphrase, recovery
// key, setup code). When r's client is backing off it returns the message for
// a 429 page and false. Otherwise the attempt is recorded as a failure up
// front (see ratelimit.Limiter.Begin), the global delay is waited out, and it
// returns true; the caller reports a right guess with AttemptSucceeded. The
// error is the request's context ending during the delay.
func (d *Deps) Throttle(r *http.Request) (refusal string, ok bool, err error) {
	if wait, ok := d.Limiter.Begin(clientIP(r)); !ok {
		return RetryMessage(wait), false, nil
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
// recorded for this attempt.
func (d *Deps) AttemptSucceeded(r *http.Request) { d.Limiter.Succeed(clientIP(r)) }

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
