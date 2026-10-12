package status

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hungcabinet/docker-amneziawg/web/internal/config"
)

// Source names accepted by New.
const (
	SourceMock = config.SourceMock
	SourceLive = config.SourceLive
)

// Exit kinds shown in the UI.
const (
	ExitKindTunnel ExitKind = "tunnel"
	ExitKindDirect ExitKind = "direct"
)

// OnlineWindow is how recent a handshake must be to count as online.
const OnlineWindow = 3 * time.Minute

// Provider is the AmneziaWG data source. Mock and live implementations share it.
type Provider interface {
	Server(ctx context.Context) (ServerStatus, error)
	Peers(ctx context.Context) ([]Peer, error)
	Exits(ctx context.Context) ([]Exit, error)
	ExitIP(ctx context.Context, iface string) (ExitIPs, error)
}

// ExitIPs is the public address seen through an exit, per address family.
// A field is empty when that family has no working route through the exit.
type ExitIPs struct {
	V4 string
	V6 string
}

// ServerStatus describes the local VPN server interface.
type ServerStatus struct {
	Up         bool
	Interface  string
	ListenPort int
}

// Peer is a server peer as shown in the panel.
type Peer struct {
	Name          string
	Online        bool
	LastHandshake time.Time
	RxBytes       int64
	TxBytes       int64
	// Endpoint is the peer's last seen address, empty if it never connected.
	Endpoint string
	Config   string
}

// ExitKind is tunnel or direct.
type ExitKind string

// PingResult is one probe through an exit interface.
type PingResult struct {
	Target  string
	OK      bool
	Latency time.Duration
	Error   string
}

// Exit is a client tunnel or the direct uplink.
type Exit struct {
	Name string
	Kind ExitKind
	// Endpoint is the remote server of a tunnel; empty for the direct uplink.
	Endpoint string
	Pings    []PingResult
}

// New returns the provider selected by cfg.Source.
func New(cfg config.Config) (Provider, error) {
	switch strings.ToLower(cfg.Source) {
	case SourceMock:
		return NewMock(cfg), nil
	case SourceLive:
		return NewLive(cfg), nil
	default:
		return nil, fmt.Errorf("unknown panel source %q", cfg.Source)
	}
}
