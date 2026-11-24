package main

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Elagoht/collage/pkg/collage"
	"github.com/Elagoht/collage/pkg/collagetest"
)

func TestAppBoots(t *testing.T) {
	t.Setenv("CSRF_KEY", strings.Repeat("k", 32))
	t.Setenv("DATABASE_PATH", filepath.Join(t.TempDir(), "passenger.db"))

	app, err := newApp(collage.Env{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatalf("newApp() = %v", err)
	}
	t.Cleanup(func() { _ = app.Shutdown(context.Background()) })

	res := collagetest.New(t, app.Handler()).Get("/")
	if res.Status != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", res.Status)
	}
}

func TestNewAppRequiresCSRFKey(t *testing.T) {
	t.Setenv("CSRF_KEY", "")
	if _, err := newApp(collage.Env{CacheDir: t.TempDir()}); err == nil {
		t.Fatal("newApp() = nil error, want CSRF_KEY error")
	}
}
