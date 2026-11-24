package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

const goodKey = "0123456789abcdef0123456789abcdef"

func TestFromEnvRequiresCSRFKey(t *testing.T) {
	_, err := FromEnv(env(nil))
	if err == nil || !strings.Contains(err.Error(), "CSRF_KEY") {
		t.Fatalf("err = %v, want error mentioning CSRF_KEY", err)
	}
}

func TestFromEnvRejectsShortCSRFKey(t *testing.T) {
	_, err := FromEnv(env(map[string]string{"CSRF_KEY": goodKey[:31]}))
	if err == nil {
		t.Fatal("want error for 31-byte key")
	}
}

func TestFromEnvDefaults(t *testing.T) {
	cfg, err := FromEnv(env(map[string]string{"CSRF_KEY": goodKey}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabasePath != "./data/passenger.db" {
		t.Errorf("DatabasePath = %q", cfg.DatabasePath)
	}
	if cfg.SessionIdle != 15*time.Minute || cfg.SessionMax != 12*time.Hour {
		t.Errorf("session = %v / %v", cfg.SessionIdle, cfg.SessionMax)
	}
	if cfg.TrustedProxies != nil {
		t.Errorf("TrustedProxies = %v, want nil", cfg.TrustedProxies)
	}
	if string(cfg.CSRFKey) != goodKey {
		t.Errorf("CSRFKey = %q", cfg.CSRFKey)
	}
}

func TestFromEnvParsesDurationsAndProxies(t *testing.T) {
	cfg, err := FromEnv(env(map[string]string{
		"CSRF_KEY":        goodKey,
		"DATABASE_PATH":   "/tmp/x.db",
		"SESSION_IDLE":    "5m",
		"SESSION_MAX":     "1h",
		"TRUSTED_PROXIES": "10.0.0.1, 192.168.0.0/16",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabasePath != "/tmp/x.db" || cfg.SessionIdle != 5*time.Minute || cfg.SessionMax != time.Hour {
		t.Errorf("cfg = %+v", cfg)
	}
	if want := []string{"10.0.0.1", "192.168.0.0/16"}; !reflect.DeepEqual(cfg.TrustedProxies, want) {
		t.Errorf("TrustedProxies = %v, want %v", cfg.TrustedProxies, want)
	}
}

func TestFromEnvRejectsBadDuration(t *testing.T) {
	for _, v := range []string{"abc", "0s"} {
		_, err := FromEnv(env(map[string]string{"CSRF_KEY": goodKey, "SESSION_IDLE": v}))
		if err == nil {
			t.Errorf("SESSION_IDLE=%q: want error", v)
		}
	}
}
