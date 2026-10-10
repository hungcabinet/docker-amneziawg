package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	SourceMock = "mock"
	SourceLive = "live"

	defaultListen          = ":8080"
	defaultUser            = "admin"
	defaultDBPath          = "data/panel.db"
	defaultDirectInterface = "eth0"
	defaultSource          = SourceMock
)

// Config is process configuration loaded from PANEL_* environment variables.
type Config struct {
	Listen          string
	Source          string
	User            string
	Password        string
	DBPath          string
	DirectInterface string
	PingTargets     []string
	PingTargetsByIF map[string][]string
}

// Load reads PANEL_* variables from the process environment.
func Load() (Config, error) {
	cfg := Config{
		Listen:          envOr("PANEL_LISTEN", defaultListen),
		Source:          strings.ToLower(strings.TrimSpace(envOr("PANEL_SOURCE", defaultSource))),
		User:            envOr("PANEL_USER", defaultUser),
		Password:        os.Getenv("PANEL_PASSWORD"),
		DBPath:          envOr("PANEL_DB_PATH", defaultDBPath),
		DirectInterface: envOr("PANEL_DIRECT_INTERFACE", defaultDirectInterface),
		PingTargets:     splitCSV(envOr("PANEL_PING_TARGETS", "1.1.1.1,8.8.8.8")),
		PingTargetsByIF: map[string][]string{},
	}

	if cfg.User == "" {
		return Config{}, fmt.Errorf("PANEL_USER must not be empty")
	}
	if cfg.DirectInterface == "" {
		return Config{}, fmt.Errorf("PANEL_DIRECT_INTERFACE must not be empty")
	}

	switch cfg.Source {
	case SourceMock, SourceLive:
	default:
		return Config{}, fmt.Errorf("unknown PANEL_SOURCE %q", cfg.Source)
	}

	for _, env := range os.Environ() {
		key, val, ok := strings.Cut(env, "=")
		if !ok {
			continue
		}
		const prefix = "PANEL_PING_TARGETS_"
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		iface := strings.ToLower(strings.TrimPrefix(key, prefix))
		if iface == "" {
			continue
		}
		cfg.PingTargetsByIF[iface] = splitCSV(val)
	}

	return cfg, nil
}

// PingTargetsFor returns the per-interface ping list, or the default list.
func (c Config) PingTargetsFor(iface string) []string {
	if targets, ok := c.PingTargetsByIF[strings.ToLower(iface)]; ok && len(targets) > 0 {
		return append([]string{}, targets...)
	}
	return append([]string{}, c.PingTargets...)
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
