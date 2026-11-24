package session

import (
	"context"
	"testing"
	"time"

	"passenger-go/internal/vault"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func setup() (*Manager, *clock) {
	c := &clock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	return NewManager(15*time.Minute, 12*time.Hour, c.now), c
}

func testDEK(b byte) vault.DEK {
	var d vault.DEK
	for i := range d {
		d[i] = b
	}
	return d
}

func TestSlidingIdle(t *testing.T) {
	m, c := setup()
	s, err := m.Create(testDEK(1))
	if err != nil {
		t.Fatal(err)
	}
	c.advance(14 * time.Minute)
	if _, ok := m.Touch(s.Handle); !ok {
		t.Fatal("touch at +14m should be ok")
	}
	c.advance(14 * time.Minute)
	if _, ok := m.Touch(s.Handle); !ok {
		t.Fatal("touch at +28m should be ok (sliding)")
	}
	c.advance(16 * time.Minute)
	if _, ok := m.Touch(s.Handle); ok {
		t.Fatal("idle exceeded should fail")
	}
	if _, ok := m.Peek(s.Handle); ok {
		t.Fatal("session should be gone")
	}
}

func TestAbsoluteMax(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(1))
	for i := 0; i < 72; i++ { // touch every 10m up to +12h
		c.advance(10 * time.Minute)
		if _, ok := m.Touch(s.Handle); !ok {
			t.Fatalf("unexpected expiry at step %d", i)
		}
	}
	c.advance(time.Minute) // +12h01m
	if _, ok := m.Touch(s.Handle); ok {
		t.Fatal("absolute max exceeded should fail")
	}
}

func TestPeekDoesNotExtend(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(1))
	c.advance(10 * time.Minute)
	if _, ok := m.Peek(s.Handle); !ok {
		t.Fatal("peek should be ok")
	}
	c.advance(6 * time.Minute)
	if _, ok := m.Touch(s.Handle); ok {
		t.Fatal("touch at +16m should fail after peek-only")
	}
}

func TestExpiresAtIdleBound(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(1))
	if got := m.ExpiresAt(s); !got.Equal(c.t.Add(15 * time.Minute)) {
		t.Fatalf("idle bound: %v", got)
	}
}

func TestExpiresAtAbsoluteBound(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(1))
	created := c.t
	for i := 0; i < 71; i++ {
		c.advance(10 * time.Minute)
		m.Touch(s.Handle)
	}
	c.advance(5 * time.Minute) // +11h55m
	cur, ok := m.Touch(s.Handle)
	if !ok {
		t.Fatal("expected valid")
	}
	if got := m.ExpiresAt(cur); !got.Equal(created.Add(12 * time.Hour)) {
		t.Fatalf("absolute bound: %v", got)
	}
}

func TestRotate(t *testing.T) {
	m, _ := setup()
	a, _ := m.Create(testDEK(7))
	b, _ := m.Create(testDEK(9))
	n, err := m.Rotate(a.Handle)
	if err != nil {
		t.Fatal(err)
	}
	if n.Handle == a.Handle {
		t.Fatal("handle must change")
	}
	if _, ok := m.Peek(a.Handle); ok {
		t.Fatal("old token valid")
	}
	if _, ok := m.Peek(b.Handle); ok {
		t.Fatal("other session valid")
	}
	got, ok := m.Peek(n.Handle)
	if !ok || got.DEK != testDEK(7) {
		t.Fatal("rotated session missing or DEK differs")
	}
	if _, err := m.Rotate("nope"); err == nil {
		t.Fatal("rotate of unknown token should error")
	}
}

func TestExpiredDEKZeroed(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(5))
	m.mu.Lock()
	internal := m.sessions[s.Handle]
	m.mu.Unlock()
	c.advance(16 * time.Minute)
	if _, ok := m.Touch(s.Handle); ok {
		t.Fatal("should be expired")
	}
	if internal.DEK != (vault.DEK{}) {
		t.Fatal("DEK not zeroed")
	}
}

func TestSweepAndDelete(t *testing.T) {
	m, c := setup()
	a, _ := m.Create(testDEK(1))
	c.advance(10 * time.Minute)
	b, _ := m.Create(testDEK(2))
	c.advance(6 * time.Minute)
	m.Sweep()
	m.mu.Lock()
	_, hasA := m.sessions[a.Handle]
	_, hasB := m.sessions[b.Handle]
	m.mu.Unlock()
	if hasA || !hasB {
		t.Fatalf("sweep wrong: a=%v b=%v", hasA, hasB)
	}
	m.Delete(b.Handle)
	if _, ok := m.Peek(b.Handle); ok {
		t.Fatal("deleted session valid")
	}
	m.Create(testDEK(3))
	m.DeleteAll()
	if len(m.sessions) != 0 {
		t.Fatal("DeleteAll left sessions")
	}
}

func TestCopyIsolated(t *testing.T) {
	m, c := setup()
	s, _ := m.Create(testDEK(4))
	cp, _ := m.Touch(s.Handle)
	c.advance(20 * time.Minute)
	m.Sweep()
	if cp.DEK != testDEK(4) {
		t.Fatal("returned copy was wiped by sweep")
	}
}

func TestLen(t *testing.T) {
	m, c := setup()
	m.Create(testDEK(1))
	m.Create(testDEK(2))
	if n := m.Len(); n != 2 {
		t.Fatalf("Len = %d, want 2", n)
	}
	c.advance(16 * time.Minute)
	m.Sweep()
	if n := m.Len(); n != 0 {
		t.Fatalf("Len after sweep = %d, want 0", n)
	}
}

func TestContext(t *testing.T) {
	if _, ok := FromContext(context.Background()); ok {
		t.Fatal("empty context has a session")
	}
	s := &Session{Handle: "h", DEK: testDEK(3)}
	got, ok := FromContext(WithSession(context.Background(), s))
	if !ok || got != s {
		t.Fatal("WithSession/FromContext lost the session")
	}
}
