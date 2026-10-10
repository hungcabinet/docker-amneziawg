package status

import (
	"context"
	"fmt"
	"time"

	"github.com/hungcabinet/docker-amneziawg/web/internal/config"
)

const mockPeerConfig = `[Interface]
Address = %s
PrivateKey = MOCK_PRIVATE_KEY_%s
DNS = 10.13.13.1
Jc = 4
Jmin = 50
Jmax = 1000
S1 = 68
S2 = 43
S3 = 0
S4 = 0
H1 = 1009484613
H2 = 826498173
H3 = 1709786516
H4 = 766939893

[Peer]
PublicKey = MOCK_SERVER_PUBLIC_KEY
PresharedKey = MOCK_PRESHARED_KEY_%s
Endpoint = test.example.com:51820
AllowedIPs = 0.0.0.0/0, ::/0
`

// Mock is a fixed scene for UI development.
type Mock struct {
	direct string
	cfg    config.Config
	now    func() time.Time
}

// NewMock builds the canned status scene from panel config.
func NewMock(cfg config.Config) *Mock {
	return &Mock{
		direct: cfg.DirectInterface,
		cfg:    cfg,
		now:    time.Now,
	}
}

// Server reports a running wg0.
func (m *Mock) Server(context.Context) (ServerStatus, error) {
	return ServerStatus{
		Up:         true,
		Interface:  "wg0",
		ListenPort: 51820,
	}, nil
}

// Peers returns three peers in distinct handshake states.
func (m *Mock) Peers(context.Context) ([]Peer, error) {
	now := m.now()
	return []Peer{
		{
			Name:          "peer_phone",
			Online:        true,
			LastHandshake: now.Add(-30 * time.Second),
			RxBytes:       12 * 1024 * 1024,
			TxBytes:       3 * 1024 * 1024,
			Config:        mockConfig("peer_phone", "10.13.13.2/32"),
		},
		{
			Name:          "peer_laptop",
			Online:        false,
			LastHandshake: now.Add(-2 * time.Hour),
			RxBytes:       450 * 1024 * 1024,
			TxBytes:       80 * 1024 * 1024,
			Config:        mockConfig("peer_laptop", "10.13.13.3/32"),
		},
		{
			Name:          "peer1",
			Online:        false,
			LastHandshake: time.Time{},
			RxBytes:       0,
			TxBytes:       0,
			Config:        mockConfig("peer1", "10.13.13.4/32"),
		},
	}, nil
}

// Exits returns wg1, wg2 and the configured direct interface.
func (m *Mock) Exits(context.Context) ([]Exit, error) {
	return []Exit{
		{
			Name:  "wg1",
			Kind:  ExitKindTunnel,
			Pings: m.pings("wg1", true, 12*time.Millisecond),
		},
		{
			Name:  "wg2",
			Kind:  ExitKindTunnel,
			Pings: m.mixedPings("wg2"),
		},
		{
			Name:  m.direct,
			Kind:  ExitKindDirect,
			Pings: m.pings(m.direct, true, 8*time.Millisecond),
		},
	}, nil
}

// ExitIP returns a distinct TEST-NET address per known interface.
func (m *Mock) ExitIP(_ context.Context, iface string) (string, error) {
	switch iface {
	case "wg1":
		return "203.0.113.10", nil
	case "wg2":
		return "198.51.100.20", nil
	case m.direct:
		return "192.0.2.1", nil
	default:
		return "", fmt.Errorf("unknown interface %q", iface)
	}
}

func (m *Mock) pings(iface string, ok bool, latency time.Duration) []PingResult {
	targets := m.cfg.PingTargetsFor(iface)
	out := make([]PingResult, 0, len(targets))
	for i, target := range targets {
		out = append(out, PingResult{
			Target:  target,
			OK:      ok,
			Latency: latency + time.Duration(i)*time.Millisecond,
		})
	}
	return out
}

func (m *Mock) mixedPings(iface string) []PingResult {
	targets := m.cfg.PingTargetsFor(iface)
	out := make([]PingResult, 0, len(targets))
	for i, target := range targets {
		if i == 0 {
			out = append(out, PingResult{
				Target:  target,
				OK:      true,
				Latency: 40 * time.Millisecond,
			})
			continue
		}
		out = append(out, PingResult{
			Target: target,
			OK:     false,
			Error:  "timeout",
		})
	}
	return out
}

func mockConfig(name, address string) string {
	return fmt.Sprintf(mockPeerConfig, address, name, name)
}
