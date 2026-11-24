// Package session keeps unlocked vault sessions (and their DEK) in memory only.
//
// It is the server half of a hybrid session. The elagoht/session plugin owns
// the browser's cookie (named CookieName), its lifetime and the guards; the
// cookie holds nothing but a session's Handle, an opaque random string. The
// Manager maps a handle to its DEK, with the same idle and absolute limits as
// the cookie, and wipes the DEK when the session expires, is deleted or is
// rotated. The DEK never leaves server memory: the plugin keeps its data in
// the cookie, which is why it cannot hold the key itself.
package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"passenger-go/internal/vault"
)

// CookieName is the name of the session cookie the elagoht/session plugin
// writes.
const CookieName = "passenger_session"

// Session is a snapshot of a stored session. Values handed out by the Manager
// are copies; the DEK inside a copy is not wiped when the stored session expires.
type Session struct {
	// Handle is the session's opaque random name, the only thing the cookie
	// holds.
	Handle   string
	DEK      vault.DEK
	Created  time.Time
	LastSeen time.Time
}

type Manager struct {
	mu       sync.Mutex
	sessions map[string]*Session
	idle     time.Duration
	max      time.Duration
	now      func() time.Time
}

func NewManager(idle, max time.Duration, now func() time.Time) *Manager {
	return &Manager{sessions: map[string]*Session{}, idle: idle, max: max, now: now}
}

func newHandle() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (m *Manager) expiresAt(s *Session) time.Time {
	idleEnd := s.LastSeen.Add(m.idle)
	maxEnd := s.Created.Add(m.max)
	if maxEnd.Before(idleEnd) {
		return maxEnd
	}
	return idleEnd
}

// ExpiresAt returns min(LastSeen+idle, Created+max).
func (m *Manager) ExpiresAt(s *Session) time.Time { return m.expiresAt(s) }

// Remaining is how long s has left by the manager's clock, never below zero.
func (m *Manager) Remaining(s *Session) time.Duration {
	if d := m.expiresAt(s).Sub(m.now()); d > 0 {
		return d
	}
	return 0
}

func (m *Manager) expiredLocked(s *Session) bool {
	return m.now().After(m.expiresAt(s))
}

func (m *Manager) removeLocked(handle string) {
	if s, ok := m.sessions[handle]; ok {
		s.DEK.Wipe()
		delete(m.sessions, handle)
	}
}

func (m *Manager) Create(dek vault.DEK) (*Session, error) {
	handle, err := newHandle()
	if err != nil {
		return nil, err
	}
	now := m.now()
	s := &Session{Handle: handle, DEK: dek, Created: now, LastSeen: now}
	m.mu.Lock()
	m.sessions[handle] = s
	m.mu.Unlock()
	cp := *s
	return &cp, nil
}

func (m *Manager) lookup(handle string, extend bool) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[handle]
	if !ok {
		return nil, false
	}
	if m.expiredLocked(s) {
		m.removeLocked(handle)
		return nil, false
	}
	if extend {
		s.LastSeen = m.now()
	}
	cp := *s
	return &cp, true
}

// Touch validates the session and slides its idle window.
func (m *Manager) Touch(handle string) (*Session, bool) { return m.lookup(handle, true) }

// Peek validates the session without extending it.
func (m *Manager) Peek(handle string) (*Session, bool) { return m.lookup(handle, false) }

func (m *Manager) Delete(handle string) {
	m.mu.Lock()
	m.removeLocked(handle)
	m.mu.Unlock()
}

// Rotate gives the session a new handle (same DEK, same creation time) and
// deletes every other session.
func (m *Manager) Rotate(handle string) (*Session, error) {
	fresh, err := newHandle()
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.sessions[handle]
	if !ok || m.expiredLocked(old) {
		m.removeLocked(handle)
		return nil, errors.New("session: unknown or expired handle")
	}
	s := &Session{Handle: fresh, DEK: old.DEK, Created: old.Created, LastSeen: m.now()}
	for t := range m.sessions {
		m.removeLocked(t)
	}
	m.sessions[fresh] = s
	cp := *s
	return &cp, nil
}

func (m *Manager) DeleteAll() {
	m.mu.Lock()
	for t := range m.sessions {
		m.removeLocked(t)
	}
	m.mu.Unlock()
}

// Sweep removes expired sessions.
func (m *Manager) Sweep() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for t, s := range m.sessions {
		if m.expiredLocked(s) {
			m.removeLocked(t)
		}
	}
}

// Start runs Sweep every minute until ctx is done.
func (m *Manager) Start(ctx context.Context) {
	go func() {
		tick := time.NewTicker(time.Minute)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				m.Sweep()
			}
		}
	}()
}

type ctxKey struct{}

// WithSession returns ctx carrying s, the request's copy of its session.
func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, ctxKey{}, s)
}

// FromContext returns the session WithSession put in ctx.
func FromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(ctxKey{}).(*Session)
	return s, ok && s != nil
}

// Len is how many sessions the manager holds, expired ones not yet swept
// included.
func (m *Manager) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}
