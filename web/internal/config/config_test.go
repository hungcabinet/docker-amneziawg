package config

import (
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("PANEL_PASSWORD", "")
	t.Setenv("PANEL_USER", "")
	t.Setenv("PANEL_SOURCE", "")
	t.Setenv("PANEL_LISTEN", "")
	t.Setenv("PANEL_DB_PATH", "")
	t.Setenv("PANEL_CONFIG_DIR", "")
	t.Setenv("PANEL_DIRECT_INTERFACE", "")
	t.Setenv("PANEL_PING_TARGETS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Source != SourceMock {
		t.Errorf("Source = %q, want %q", cfg.Source, SourceMock)
	}
	if cfg.User != defaultUser {
		t.Errorf("User = %q, want %q", cfg.User, defaultUser)
	}
	if cfg.Listen != defaultListen {
		t.Errorf("Listen = %q, want %q", cfg.Listen, defaultListen)
	}
	if cfg.ConfigDir != defaultConfigDir {
		t.Errorf("ConfigDir = %q, want %q", cfg.ConfigDir, defaultConfigDir)
	}
	if cfg.DirectInterface != defaultDirectInterface {
		t.Errorf("DirectInterface = %q, want %q", cfg.DirectInterface, defaultDirectInterface)
	}
	if got, want := cfg.PingTargets, []string{"1.1.1.1", "8.8.8.8"}; !stringSliceEqual(got, want) {
		t.Errorf("PingTargets = %v, want %v", got, want)
	}
}

func TestLoad_SourceAndNamedPingTargets(t *testing.T) {
	t.Setenv("PANEL_SOURCE", "LIVE")
	t.Setenv("PANEL_CONFIG_DIR", "/tmp/awg-config")
	t.Setenv("PANEL_DIRECT_INTERFACE", "ens3")
	t.Setenv("PANEL_PING_TARGETS", "1.1.1.1, 8.8.8.8")
	t.Setenv("PANEL_PING_TARGETS_WG1", "9.9.9.9")
	t.Setenv("PANEL_PING_TARGETS_ENS3", "1.0.0.1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Source != SourceLive {
		t.Errorf("Source = %q, want %q", cfg.Source, SourceLive)
	}
	if cfg.ConfigDir != "/tmp/awg-config" {
		t.Errorf("ConfigDir = %q, want /tmp/awg-config", cfg.ConfigDir)
	}
	if got := cfg.PingTargetsFor("wg1"); !stringSliceEqual(got, []string{"9.9.9.9"}) {
		t.Errorf("PingTargetsFor(wg1) = %v", got)
	}
	if got := cfg.PingTargetsFor("ens3"); !stringSliceEqual(got, []string{"1.0.0.1"}) {
		t.Errorf("PingTargetsFor(ens3) = %v", got)
	}
	if got := cfg.PingTargetsFor("wg2"); !stringSliceEqual(got, []string{"1.1.1.1", "8.8.8.8"}) {
		t.Errorf("PingTargetsFor(wg2) = %v", got)
	}
}

func TestLoad_UnknownSource(t *testing.T) {
	t.Setenv("PANEL_SOURCE", "file")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want unknown source")
	}
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
