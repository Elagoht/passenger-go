package ratelimit

import (
	"fmt"
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

func failN(l *Limiter, ip netip.Addr, n int) {
	for i := 0; i < n; i++ {
		l.Fail(ip)
	}
}

func TestFourFailuresStillAllowed(t *testing.T) {
	l, _ := newTest()
	failN(l, ipA, 4)
	if wait, ok := l.Allow(ipA); !ok || wait != 0 {
		t.Fatalf("Allow = %v, %v; want 0, true", wait, ok)
	}
}

func TestFifthFailureBlocksOneSecond(t *testing.T) {
	l, c := newTest()
	failN(l, ipA, 5)
	if wait, ok := l.Allow(ipA); ok || wait != time.Second {
		t.Fatalf("Allow = %v, %v; want 1s, false", wait, ok)
	}
	c.advance(400 * time.Millisecond)
	if wait, ok := l.Allow(ipA); ok || wait != 600*time.Millisecond {
		t.Fatalf("Allow = %v, %v; want 600ms, false", wait, ok)
	}
	c.advance(600 * time.Millisecond)
	if wait, ok := l.Allow(ipA); !ok || wait != 0 {
		t.Fatalf("Allow after 1s = %v, %v; want 0, true", wait, ok)
	}
}

func TestSixthFailureBlocksTwoSecondsFromItsOwnTime(t *testing.T) {
	l, c := newTest()
	failN(l, ipA, 5)
	c.advance(time.Second)
	l.Fail(ipA)
	if wait, ok := l.Allow(ipA); ok || wait != 2*time.Second {
		t.Fatalf("Allow = %v, %v; want 2s, false", wait, ok)
	}
}

func TestFailWhileBlockedStillCounts(t *testing.T) {
	l, _ := newTest()
	failN(l, ipA, 6) // no time passes between failures
	if wait, ok := l.Allow(ipA); ok || wait != 2*time.Second {
		t.Fatalf("Allow = %v, %v; want 2s, false", wait, ok)
	}
}

func TestBackoffCappedAtFiveMinutes(t *testing.T) {
	l, _ := newTest()
	failN(l, ipA, 200)
	if wait, ok := l.Allow(ipA); ok || wait != 5*time.Minute {
		t.Fatalf("Allow = %v, %v; want 5m, false", wait, ok)
	}
}

func TestOtherIPUnaffected(t *testing.T) {
	l, _ := newTest()
	failN(l, ipA, 10)
	if _, ok := l.Allow(ipB); !ok {
		t.Fatal("other IP must stay allowed")
	}
}

func TestSucceedResets(t *testing.T) {
	l, _ := newTest()
	failN(l, ipA, 7)
	l.Succeed(ipA)
	if _, ok := l.Allow(ipA); !ok {
		t.Fatal("Succeed must clear the block")
	}
	failN(l, ipA, 4)
	if _, ok := l.Allow(ipA); !ok {
		t.Fatal("counter must restart after Succeed")
	}
}

func TestStaleEntriesPruned(t *testing.T) {
	l, c := newTest()
	failN(l, ipA, 8)
	c.advance(25 * time.Hour)
	if _, ok := l.Allow(ipB); !ok {
		t.Fatal("unexpected block")
	}
	l.mu.Lock()
	n := len(l.ips)
	l.mu.Unlock()
	if n != 0 {
		t.Fatalf("stale entries = %d; want 0", n)
	}
	failN(l, ipA, 4)
	if _, ok := l.Allow(ipA); !ok {
		t.Fatal("old failures must not count after 24h")
	}
}

func TestGlobalDelay(t *testing.T) {
	l, c := newTest()
	ip := func(i int) netip.Addr { return netip.MustParseAddr(fmt.Sprintf("10.0.%d.%d", i/250, i%250+1)) }
	for i := 0; i < 100; i++ {
		l.Fail(ip(i))
	}
	if d := l.GlobalDelay(); d != 0 {
		t.Fatalf("GlobalDelay at 100 = %v; want 0", d)
	}
	l.Fail(ip(100))
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
	failN(l, ipA, 100000)
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
				l.Fail(ipA)
				l.Allow(ipA)
				l.GlobalDelay()
				l.Succeed(ipB)
			}
		}()
	}
	wg.Wait()
}
