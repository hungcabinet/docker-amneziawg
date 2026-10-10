package status

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hungcabinet/docker-amneziawg/web/internal/config"
)

func TestNew_SelectsImplementation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  string
		wantErr bool
		check   func(t *testing.T, p Provider)
	}{
		{
			name:   "mock",
			source: SourceMock,
			check: func(t *testing.T, p Provider) {
				if _, ok := p.(*Mock); !ok {
					t.Fatalf("got %T, want *Mock", p)
				}
			},
		},
		{
			name:   "live",
			source: SourceLive,
			check: func(t *testing.T, p Provider) {
				if _, ok := p.(*Live); !ok {
					t.Fatalf("got %T, want *Live", p)
				}
			},
		},
		{
			name:    "unknown",
			source:  "file",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := New(config.Config{Source: tt.source, DirectInterface: "eth0"})
			if tt.wantErr {
				if err == nil {
					t.Fatal("New() error = nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			tt.check(t, p)
		})
	}
}

func TestMock_Scene(t *testing.T) {
	t.Parallel()
	cfg := config.Config{
		DirectInterface: "ens3",
		PingTargets:     []string{"1.1.1.1", "8.8.8.8"},
		PingTargetsByIF: map[string][]string{
			"wg1": {"9.9.9.9"},
		},
	}
	m := NewMock(cfg)
	m.now = func() time.Time {
		return time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	}
	ctx := context.Background()

	server, err := m.Server(ctx)
	if err != nil {
		t.Fatalf("Server() error = %v", err)
	}
	if !server.Up || server.Interface != "wg0" || server.ListenPort != 51820 {
		t.Fatalf("Server() = %+v", server)
	}

	peers, err := m.Peers(ctx)
	if err != nil {
		t.Fatalf("Peers() error = %v", err)
	}
	if len(peers) != 3 {
		t.Fatalf("len(Peers) = %d, want 3", len(peers))
	}
	if !peers[0].Online || peers[0].Name != "peer_phone" || peers[0].LastHandshake.IsZero() {
		t.Fatalf("online peer = %+v", peers[0])
	}
	if peers[1].Online || peers[1].LastHandshake.IsZero() {
		t.Fatalf("stale peer = %+v", peers[1])
	}
	if peers[2].Online || !peers[2].LastHandshake.IsZero() {
		t.Fatalf("never-seen peer = %+v", peers[2])
	}
	for _, peer := range peers {
		if peer.Config == "" || !strings.Contains(peer.Config, "MOCK_PRIVATE_KEY_") {
			t.Fatalf("peer %s config is not mock data", peer.Name)
		}
	}

	exits, err := m.Exits(ctx)
	if err != nil {
		t.Fatalf("Exits() error = %v", err)
	}
	if len(exits) != 3 {
		t.Fatalf("len(Exits) = %d, want 3", len(exits))
	}
	if exits[0].Name != "wg1" || exits[0].Kind != ExitKindTunnel {
		t.Fatalf("wg1 = %+v", exits[0])
	}
	if got := pingTargets(exits[0]); !stringSliceEqual(got, []string{"9.9.9.9"}) {
		t.Fatalf("wg1 ping targets = %v", got)
	}
	if exits[1].Name != "wg2" || exits[1].Pings[1].OK {
		t.Fatalf("wg2 should have a failed ping: %+v", exits[1])
	}
	if exits[2].Name != "ens3" || exits[2].Kind != ExitKindDirect {
		t.Fatalf("direct = %+v", exits[2])
	}

	ip, err := m.ExitIP(ctx, "wg1")
	if err != nil || ip != "203.0.113.10" {
		t.Fatalf("ExitIP(wg1) = %q, %v", ip, err)
	}
	ip, err = m.ExitIP(ctx, "ens3")
	if err != nil || ip != "192.0.2.1" {
		t.Fatalf("ExitIP(ens3) = %q, %v", ip, err)
	}
	if _, err := m.ExitIP(ctx, "missing"); err == nil {
		t.Fatal("ExitIP(missing) = nil")
	}
}

func TestLive_NotConnected(t *testing.T) {
	t.Parallel()
	p := NewLive()
	ctx := context.Background()
	if _, err := p.Server(ctx); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("Server() error = %v", err)
	}
	if _, err := p.Peers(ctx); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("Peers() error = %v", err)
	}
	if _, err := p.Exits(ctx); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("Exits() error = %v", err)
	}
	if _, err := p.ExitIP(ctx, "wg1"); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("ExitIP() error = %v", err)
	}
}

func pingTargets(e Exit) []string {
	out := make([]string, 0, len(e.Pings))
	for _, p := range e.Pings {
		out = append(out, p.Target)
	}
	return out
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
