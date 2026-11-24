package main

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Elagoht/collage/pkg/collage"
	"github.com/Elagoht/collage/pkg/collagetest"

	"passenger-go/internal/vault"
	"passenger-go/internal/web"
)

// fakeClock is a controllable time source shared by the session manager, the
// rate limiter and the store.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// testEnv is what a test needs besides the browser: the clock and the
// services the app was built with.
type testEnv struct {
	Clock *fakeClock
	Deps  *web.Deps
	App   *collage.App
}

// testKDF keeps Argon2id cheap in tests.
var testKDF = vault.KDFParams{Alg: "argon2id", T: 1, M: 1024, P: 1}

// newTestApp builds the real app (through buildApp, as newApp does) on a temp
// database, with a fast KDF and a fake clock, and returns a browser for it.
func newTestApp(t *testing.T) (*collagetest.Client, *testEnv) {
	t.Helper()
	t.Setenv("CSRF_KEY", strings.Repeat("k", 32))
	t.Setenv("DATABASE_PATH", filepath.Join(t.TempDir(), "passenger.db"))

	clock := &fakeClock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	kdf := testKDF
	kdfOverride, nowOverride = &kdf, clock.Now
	t.Cleanup(func() { kdfOverride, nowOverride = nil, nil })

	app, deps, err := buildApp(collage.Env{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatalf("buildApp() = %v", err)
	}
	t.Cleanup(func() {
		if err := app.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown() = %v", err)
		}
	})
	return collagetest.New(t, app.Handler()), &testEnv{Clock: clock, Deps: deps, App: app}
}
