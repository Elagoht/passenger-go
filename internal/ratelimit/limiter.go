// Package ratelimit throttles secret guesses (passphrase, recovery key, setup
// code) with a per-IP exponential backoff plus a global ceiling.
package ratelimit

import (
	"net/netip"
	"sync"
	"time"
)

const (
	freeAttempts = 4 // failures tolerated before backoff starts (the 5th blocks)
	baseBackoff  = time.Second
	maxBackoff   = 5 * time.Minute

	entryTTL = 24 * time.Hour
	// sweepEvery bounds how often the whole per-IP map is scanned for stale entries.
	sweepEvery = time.Minute

	globalWindow = time.Hour
	globalLimit  = 100
	globalDelay  = 2 * time.Second
	// maxGlobalEntries bounds the timestamp slice; it only needs to exceed globalLimit.
	maxGlobalEntries = 1024
)

type entry struct {
	fails        int
	lastFail     time.Time
	blockedUntil time.Time
}

// Limiter is safe for concurrent use.
type Limiter struct {
	now       func() time.Time
	mu        sync.Mutex
	ips       map[netip.Addr]*entry
	global    []time.Time // failure times of all IPs within globalWindow, ascending
	lastSweep time.Time
}

// New returns a Limiter reading time from now.
func New(now func() time.Time) *Limiter {
	return &Limiter{now: now, ips: make(map[netip.Addr]*entry)}
}

// sweep drops entries untouched for 24h. Caller holds l.mu.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < sweepEvery {
		return
	}
	l.lastSweep = now
	for ip, e := range l.ips {
		if now.Sub(e.lastFail) >= entryTTL {
			delete(l.ips, ip)
		}
	}
}

// Allow reports whether ip may attempt now; if not, wait is the remaining backoff.
func (l *Limiter) Allow(ip netip.Addr) (wait time.Duration, ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	e := l.ips[ip]
	if e == nil || now.Sub(e.lastFail) >= entryTTL {
		return 0, true
	}
	if wait = e.blockedUntil.Sub(now); wait > 0 {
		return wait, false
	}
	return 0, true
}

// Fail records a failed attempt. From the 5th consecutive failure n on, ip is
// blocked until now + 1s*2^(n-5), capped at 5m.
func (l *Limiter) Fail(ip netip.Addr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	l.failLocked(ip, now)
}

// Begin starts an attempt from ip. When ip is backing off it returns the
// remaining wait and false. Otherwise it records the attempt as a failure up
// front, exactly as Fail would, and returns true; the caller calls Succeed if
// the guess turns out right. Checking and recording under one lock means
// parallel attempts cannot all slip past the backoff before any failure is
// counted.
func (l *Limiter) Begin(ip netip.Addr) (wait time.Duration, ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	if e := l.ips[ip]; e != nil && now.Sub(e.lastFail) < entryTTL {
		if wait = e.blockedUntil.Sub(now); wait > 0 {
			return wait, false
		}
	}
	l.failLocked(ip, now)
	return 0, true
}

// failLocked records a failure of ip at now. Caller holds l.mu.
func (l *Limiter) failLocked(ip netip.Addr, now time.Time) {
	e := l.ips[ip]
	if e == nil || now.Sub(e.lastFail) >= entryTTL {
		e = &entry{}
		l.ips[ip] = e
	}
	e.fails++
	e.lastFail = now
	if e.fails > freeAttempts {
		e.blockedUntil = now.Add(backoff(e.fails - freeAttempts - 1))
	}
	l.pruneGlobal(now)
	l.global = append(l.global, now)
	if len(l.global) > maxGlobalEntries {
		l.global = append(l.global[:0], l.global[len(l.global)-maxGlobalEntries:]...)
	}
}

// backoff returns 1s*2^exp capped at 5m, without overflowing for large exp.
func backoff(exp int) time.Duration {
	d := baseBackoff
	for i := 0; i < exp && d < maxBackoff; i++ {
		d *= 2
	}
	return min(d, maxBackoff)
}

// Succeed resets ip's failure history.
func (l *Limiter) Succeed(ip netip.Addr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.ips, ip)
}

// GlobalDelay is 2s when more than 100 failures (all IPs) happened in the
// trailing hour, else 0.
func (l *Limiter) GlobalDelay() time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneGlobal(l.now())
	if len(l.global) > globalLimit {
		return globalDelay
	}
	return 0
}

// pruneGlobal drops timestamps at or before now-1h. Caller holds l.mu.
func (l *Limiter) pruneGlobal(now time.Time) {
	cutoff := now.Add(-globalWindow)
	i := 0
	for i < len(l.global) && !l.global[i].After(cutoff) {
		i++
	}
	if i > 0 {
		l.global = append(l.global[:0], l.global[i:]...)
	}
}
