package web

import (
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/Elagoht/collage/pkg/collage"
)

// clientIP is the rate limiter's key for r.
func clientIP(r *http.Request) netip.Addr { return collage.ClientIP(r).Unmap() }

// Throttle is called before checking a guessed secret (passphrase, recovery
// key, setup code). When r's client is backing off it returns the message for
// a 429 page and false. Otherwise it waits out the global delay and returns
// true; the caller then reports the outcome with AttemptFailed or
// AttemptSucceeded.
func (d *Deps) Throttle(r *http.Request) (refusal string, ok bool) {
	if wait, ok := d.Limiter.Allow(clientIP(r)); !ok {
		return RetryMessage(wait), false
	}
	time.Sleep(d.Limiter.GlobalDelay())
	return "", true
}

// AttemptFailed records a wrong guess from r's client.
func (d *Deps) AttemptFailed(r *http.Request) { d.Limiter.Fail(clientIP(r)) }

// AttemptSucceeded clears r's client's failures.
func (d *Deps) AttemptSucceeded(r *http.Request) { d.Limiter.Succeed(clientIP(r)) }

// RetryMessage tells a client backing off how long to wait, in whole seconds
// rounded up.
func RetryMessage(wait time.Duration) string {
	n := (wait + time.Second - 1) / time.Second
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
