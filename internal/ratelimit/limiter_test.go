package ratelimit

import (
	"net/netip"
	"sync"
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTest() (*Limiter, *fakeClock) {
	c := &fakeClock{t: time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)}
	return New(c.now), c
}

var ipA = netip.MustParseAddr("203.0.113.1")
var ipB = netip.MustParseAddr("203.0.113.2")

// beginN starts n attempts from ip without time passing, failing the test if
// one is refused.
func beginN(t *testing.T, l *Limiter, ip netip.Addr, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if wait, ok := l.Begin(ip); !ok {
			t.Fatalf("attempt %d refused (wait %v)", i+1, wait)
		}
	}
}

func wantRefused(t *testing.T, l *Limiter, ip netip.Addr, want time.Duration) {
	t.Helper()
	if wait, ok := l.Begin(ip); ok || wait != want {
		t.Fatalf("Begin = %v, %v; want %v, false", wait, ok, want)
	}
}

func TestFifthAttemptStillAllowed(t *testing.T) {
	l, _ := newTest()
	beginN(t, l, ipA, 5)
}

func TestFifthFailureBlocksOneSecond(t *testing.T) {
	l, c := newTest()
	beginN(t, l, ipA, 5)
	wantRefused(t, l, ipA, time.Second)
	c.advance(400 * time.Millisecond)
	wantRefused(t, l, ipA, 600*time.Millisecond)
	c.advance(600 * time.Millisecond)
	beginN(t, l, ipA, 1)
}

func TestSixthFailureBlocksTwoSecondsFromItsOwnTime(t *testing.T) {
	l, c := newTest()
	beginN(t, l, ipA, 5)
	c.advance(time.Second)
	beginN(t, l, ipA, 1)
	wantRefused(t, l, ipA, 2*time.Second)
}

func TestRefusedAttemptDoesNotCount(t *testing.T) {
	l, c := newTest()
	beginN(t, l, ipA, 5)
	for i := 0; i < 3; i++ {
		wantRefused(t, l, ipA, time.Second)
	}
	c.advance(time.Second)
	beginN(t, l, ipA, 1)                  // the 6th failure, not the 9th
	wantRefused(t, l, ipA, 2*time.Second) // 1s*2^(6-5)
}

func TestBackoffCappedAtFiveMinutes(t *testing.T) {
	l, c := newTest()
	for fails := 0; fails < 40; {
		if wait, ok := l.Begin(ipA); ok {
			fails++
		} else {
			c.advance(wait)
		}
	}
	wantRefused(t, l, ipA, 5*time.Minute)
}

func TestOtherIPUnaffected(t *testing.T) {
	l, _ := newTest()
	beginN(t, l, ipA, 5)
	wantRefused(t, l, ipA, time.Second)
	beginN(t, l, ipB, 1)
}

func TestSucceedResets(t *testing.T) {
	l, _ := newTest()
	beginN(t, l, ipA, 5)
	l.Succeed(ipA)
	beginN(t, l, ipA, 5) // the counter restarted: five more before a block
	wantRefused(t, l, ipA, time.Second)
}

func TestStaleEntriesPruned(t *testing.T) {
	l, c := newTest()
	beginN(t, l, ipA, 5)
	c.advance(25 * time.Hour)
	beginN(t, l, ipB, 1) // sweeps
	l.mu.Lock()
	_, kept := l.ips[ipA]
	l.mu.Unlock()
	if kept {
		t.Fatal("stale entry kept")
	}
	beginN(t, l, ipA, 5) // old failures no longer count
}

// distinctIP is the i-th of many different IPv4 addresses.
func distinctIP(i int) netip.Addr {
	return netip.AddrFrom4([4]byte{10, byte(i >> 16), byte(i >> 8), byte(i)})
}

func TestGlobalDelay(t *testing.T) {
	l, c := newTest()
	for i := 0; i < 100; i++ {
		l.Begin(distinctIP(i))
	}
	if d := l.GlobalDelay(); d != 0 {
		t.Fatalf("GlobalDelay at 100 = %v; want 0", d)
	}
	l.Begin(distinctIP(100))
	if d := l.GlobalDelay(); d != 2*time.Second {
		t.Fatalf("GlobalDelay at 101 = %v; want 2s", d)
	}
	c.advance(time.Hour)
	if d := l.GlobalDelay(); d != 0 {
		t.Fatalf("GlobalDelay after an hour = %v; want 0", d)
	}
}

func TestGlobalMemoryBounded(t *testing.T) {
	l, _ := newTest()
	for i := 0; i < 100000; i++ {
		l.Begin(distinctIP(i))
	}
	l.mu.Lock()
	n := len(l.global)
	l.mu.Unlock()
	if n > maxGlobalEntries {
		t.Fatalf("global entries = %d; want <= %d", n, maxGlobalEntries)
	}
	if d := l.GlobalDelay(); d != 2*time.Second {
		t.Fatalf("GlobalDelay = %v; want 2s", d)
	}
}

func TestConcurrent(t *testing.T) {
	l, _ := newTest()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				l.Begin(ipA)
				l.GlobalDelay()
				l.Succeed(ipB)
			}
		}()
	}
	wg.Wait()
}

func TestBeginCountsAsFailure(t *testing.T) {
	l, c := newTest()
	for i := 0; i < 5; i++ {
		if _, ok := l.Begin(ipA); !ok {
			t.Fatalf("Begin %d refused", i+1)
		}
	}
	if wait, ok := l.Begin(ipA); ok || wait != time.Second {
		t.Fatalf("6th Begin = %v, %v; want 1s, false", wait, ok)
	}
	c.advance(time.Second)
	if _, ok := l.Begin(ipA); !ok {
		t.Fatal("Begin after the backoff refused")
	}
	l.Succeed(ipA)
	if _, ok := l.Begin(ipA); !ok {
		t.Fatal("Succeed did not reset")
	}
	if _, ok := l.Begin(ipB); !ok {
		t.Fatal("other IP refused")
	}
}

func TestBeginFeedsGlobalWindow(t *testing.T) {
	l, _ := newTest()
	for i := 0; i <= globalLimit; i++ {
		l.Begin(netip.AddrFrom4([4]byte{10, 0, byte(i >> 8), byte(i)}))
	}
	if d := l.GlobalDelay(); d != globalDelay {
		t.Fatalf("GlobalDelay = %v, want %v", d, globalDelay)
	}
}

func TestBeginConcurrentSameIP(t *testing.T) {
	l, _ := newTest()
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := l.Begin(ipA); ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed > 5 {
		t.Fatalf("%d of 20 concurrent attempts allowed, want at most 5", allowed)
	}
}
