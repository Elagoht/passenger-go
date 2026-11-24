// Package session keeps unlocked vault sessions (and their DEK) in memory only.
package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/vault"
)

const CookieName = "passenger_session"

// Session is a snapshot of a stored session. Values handed out by the Manager
// are copies; the DEK inside a copy is not wiped when the stored session expires.
type Session struct {
	Token    string
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

func newToken() (string, error) {
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

func (m *Manager) removeLocked(token string) {
	if s, ok := m.sessions[token]; ok {
		s.DEK.Wipe()
		delete(m.sessions, token)
	}
}

func (m *Manager) Create(dek vault.DEK) (*Session, error) {
	token, err := newToken()
	if err != nil {
		return nil, err
	}
	now := m.now()
	s := &Session{Token: token, DEK: dek, Created: now, LastSeen: now}
	m.mu.Lock()
	m.sessions[token] = s
	m.mu.Unlock()
	cp := *s
	return &cp, nil
}

func (m *Manager) lookup(token string, extend bool) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[token]
	if !ok {
		return nil, false
	}
	if m.expiredLocked(s) {
		m.removeLocked(token)
		return nil, false
	}
	if extend {
		s.LastSeen = m.now()
	}
	cp := *s
	return &cp, true
}

// Touch validates the session and slides its idle window.
func (m *Manager) Touch(token string) (*Session, bool) { return m.lookup(token, true) }

// Peek validates the session without extending it.
func (m *Manager) Peek(token string) (*Session, bool) { return m.lookup(token, false) }

func (m *Manager) Delete(token string) {
	m.mu.Lock()
	m.removeLocked(token)
	m.mu.Unlock()
}

// Rotate gives the session a new token (same DEK, same creation time) and
// deletes every other session.
func (m *Manager) Rotate(token string) (*Session, error) {
	newTok, err := newToken()
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.sessions[token]
	if !ok || m.expiredLocked(old) {
		m.removeLocked(token)
		return nil, errors.New("session: unknown or expired token")
	}
	s := &Session{Token: newTok, DEK: old.DEK, Created: old.Created, LastSeen: m.now()}
	for t := range m.sessions {
		m.removeLocked(t)
	}
	m.sessions[newTok] = s
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

// Cookie is the session cookie for token (browser-session lifetime; validity
// is server-side): HttpOnly, SameSite=Strict, Path=/, Secure over TLS.
func Cookie(r *http.Request, token string) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   collage.IsHTTPS(r),
	}
}

// ExpiredCookie is the session cookie that tells the browser to drop it.
func ExpiredCookie(r *http.Request) *http.Cookie {
	c := Cookie(r, "")
	c.MaxAge = -1
	return c
}
