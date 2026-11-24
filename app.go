package main

import (
	"context"
	"log"
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
	code, err := web.NewSetupCode()
	if err != nil {
		st.Close()
		return nil, nil, err
	}
	deps := &web.Deps{
		Store:     st,
		Bans:      newBans(),
		Sessions:  session.NewManager(cfg.SessionIdle, cfg.SessionMax, now),
		Limiter:   ratelimit.New(now),
		KDF:       kdf,
		Now:       now,
		SetupCode: code,
		Pending:   web.NewPendingSetup(now),
	}
	initialized, err := st.Initialized(context.Background())
	if err != nil {
		st.Close()
		return nil, nil, err
	}
	if !initialized {
		// Only someone who can read this log can claim a fresh install.
		log.Printf("setup code: %s", code)
	}

	app, err := wire(env, cfg, deps)
	if err != nil {
		st.Close()
		return nil, nil, err
	}
	return app, deps, nil
}

func wire(env collage.Env, cfg config.Config, deps *web.Deps) (*collage.App, error) {
	pluginConfig, err := env.PluginConfig(files)
	if err != nil {
		return nil, err
	}
	app, err := collage.New(&collage.Config{
		DevMode: env.Dev,
		Server: collage.ServerConfig{
			Host:           env.Host,
			Port:           listenPort(env, os.Getenv),
			TrustedProxies: cfg.TrustedProxies,
		},
		Template: collage.TemplateConfig{
			FS:        files,
			Root:      "templates",
			Extension: ".html",
		},
		Cache:        collage.CacheConfig{Enabled: false},
		Security:     collage.SecurityConfig{CSRFKey: cfg.CSRFKey},
		PluginConfig: pluginConfig,
		Plugins:      plugins(deps.Store, deps.Bans),
	})
	if err != nil {
		return nil, err
	}

	if err := app.RegisterPlugin(&lifecycle{deps: deps}); err != nil {
		return nil, err
	}
	if err := register(app, deps, newSession(cfg)); err != nil {
		return nil, err
	}
	return app, app.Mount("/static/", env.Static(files, "static"))
}

// defaultPort is where the server listens when neither PORT nor -port says.
const defaultPort = 8080

// listenPort is env.Port, except that collage's own default (6060, used when
// neither PORT nor -port is given) becomes defaultPort.
func listenPort(env collage.Env, getenv func(string) string) int {
	if getenv("PORT") == "" && env.Port == 6060 {
		return defaultPort
	}
	return env.Port
}

// lifecycle runs the session sweeper while the app is up and, when it shuts
// down, ends every session (wiping their keys) and closes the store.
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
	l.deps.Sessions.DeleteAll()
	return l.deps.Store.Close()
}
