package main

import (
	"context"
	"os"
	"time"

	"passenger-go/internal/config"
	"passenger-go/internal/ratelimit"
	"passenger-go/internal/session"
	"passenger-go/internal/store"
	"passenger-go/internal/vault"
	"passenger-go/internal/web"

	"github.com/Elagoht/collage/pkg/collage"
)

// Test hooks, set only by tests: a cheap KDF and a controllable clock.
var (
	kdfOverride *vault.KDFParams
	nowOverride func() time.Time
)

func newApp(env collage.Env) (*collage.App, error) {
	app, _, err := buildApp(env)
	return app, err
}

// buildApp is newApp that also returns the services the app was built with.
func buildApp(env collage.Env) (*collage.App, *web.Deps, error) {
	cfg, err := config.FromEnv(os.Getenv)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now
	if nowOverride != nil {
		now = nowOverride
	}
	kdf := vault.DefaultKDF
	if kdfOverride != nil {
		kdf = *kdfOverride
	}

	st, err := store.Open(cfg.DatabasePath)
	if err != nil {
		return nil, nil, err
	}
	st.SetClock(now)
	deps := &web.Deps{
		Store:    st,
		Sessions: session.NewManager(cfg.SessionIdle, cfg.SessionMax, now),
		Limiter:  ratelimit.New(now),
		KDF:      kdf,
		Now:      now,
	}

	app, err := wire(env, cfg, deps)
	if err != nil {
		st.Close()
		return nil, nil, err
	}
	return app, deps, nil
}

func wire(env collage.Env, cfg config.Config, deps *web.Deps) (*collage.App, error) {
	app, err := collage.New(&collage.Config{
		DevMode: env.Dev,
		Server: collage.ServerConfig{
			Host:           env.Host,
			Port:           env.Port,
			TrustedProxies: cfg.TrustedProxies,
		},
		Template: collage.TemplateConfig{
			FS:        files,
			Root:      "templates",
			Extension: ".html",
		},
		Cache:    collage.CacheConfig{Enabled: false},
		Security: collage.SecurityConfig{CSRFKey: cfg.CSRFKey},
	})
	if err != nil {
		return nil, err
	}

	if err := app.RegisterPlugin(&lifecycle{deps: deps}); err != nil {
		return nil, err
	}
	if err := app.Use(web.SecurityHeaders); err != nil {
		return nil, err
	}
	if err := app.Use(deps.Sessions.Middleware); err != nil {
		return nil, err
	}
	if err := register(app, deps); err != nil {
		return nil, err
	}
	return app, app.Mount("/static/", env.Static(files, "static"))
}

// lifecycle runs the session sweeper while the app is up and closes the store
// when it shuts down.
type lifecycle struct {
	deps   *web.Deps
	cancel context.CancelFunc
}

func (l *lifecycle) Name() string    { return "passenger-lifecycle" }
func (l *lifecycle) Version() string { return "1" }

func (l *lifecycle) Init(_ context.Context, _ collage.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	l.deps.Sessions.Start(ctx)
	return nil
}

func (l *lifecycle) Shutdown(_ context.Context) error {
	if l.cancel != nil {
		l.cancel()
	}
	return l.deps.Store.Close()
}
