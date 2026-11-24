package main

import (
	"os"

	"passenger-go/internal/config"

	"github.com/Elagoht/collage/pkg/collage"
)

func newApp(env collage.Env) (*collage.App, error) {
	cfg, err := config.FromEnv(os.Getenv)
	if err != nil {
		return nil, err
	}

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

	if err := register(app); err != nil {
		return nil, err
	}

	return app, app.Mount("/static/", env.Static(files, "static"))
}
