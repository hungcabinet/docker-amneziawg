package status

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hungcabinet/docker-amneziawg/web/internal/config"
)

// ErrUnknownInterface is returned when ExitIP is asked for a name that is not an exit.
var ErrUnknownInterface = errors.New("unknown interface")

// Live reads AmneziaWG dump, peer directories, ping and curl from the host.
type Live struct {
	cfg config.Config
	run runner
	now func() time.Time
}

// NewLive builds the live provider from panel config.
func NewLive(cfg config.Config) *Live {
	if cfg.ConfigDir == "" {
		cfg.ConfigDir = "/config"
	}
	if cfg.DirectInterface == "" {
		cfg.DirectInterface = "eth0"
	}
	if len(cfg.PingTargets) == 0 {
		cfg.PingTargets = []string{"1.1.1.1", "8.8.8.8"}
	}
	return &Live{
		cfg: cfg,
		run: hostRunner{},
		now: time.Now,
	}
}

// Server reports the interface whose peers all have endpoint (none).
func (l *Live) Server(ctx context.Context) (ServerStatus, error) {
	ifaces, err := l.interfaces(ctx)
	if err != nil {
		return ServerStatus{}, err
	}
	srv, ok := findServer(ifaces)
	if !ok {
		return ServerStatus{}, nil
	}
	return ServerStatus{
		Up:         true,
		Interface:  srv.Name,
		ListenPort: srv.ListenPort,
	}, nil
}

// Peers returns server peers named from publickey files under ConfigDir.
func (l *Live) Peers(ctx context.Context) ([]Peer, error) {
	ifaces, err := l.interfaces(ctx)
	if err != nil {
		return nil, err
	}
	srv, ok := findServer(ifaces)
	if !ok {
		return []Peer{}, nil
	}
	files, err := loadPeerFiles(l.cfg.ConfigDir)
	if err != nil {
		return nil, fmt.Errorf("peer configs: %w", err)
	}
	now := l.now()
	peers := make([]Peer, 0, len(srv.Peers))
	for _, raw := range srv.Peers {
		name := shortPublicKey(raw.PublicKey)
		var configText string
		if file, ok := files[raw.PublicKey]; ok {
			name = file.Name
			configText = file.Config
		}
		last, online := handshakeState(raw.Handshake, now)
		peers = append(peers, Peer{
			Name:          name,
			Online:        online,
			LastHandshake: last,
			RxBytes:       raw.Rx,
			TxBytes:       raw.Tx,
			Config:        configText,
		})
	}
	return peers, nil
}

// Exits lists client tunnels plus the direct uplink, then pings each target.
func (l *Live) Exits(ctx context.Context) ([]Exit, error) {
	ifaces, err := l.interfaces(ctx)
	if err != nil {
		return nil, err
	}
	exits := make([]Exit, 0)
	seen := map[string]bool{}
	for _, iface := range ifaces {
		if !iface.isTunnel() {
			continue
		}
		exits = append(exits, Exit{Name: iface.Name, Kind: ExitKindTunnel})
		seen[iface.Name] = true
	}
	direct := cleanIface(l.cfg.DirectInterface)
	if links, err := l.linkSet(ctx); err == nil && links[direct] && !seen[direct] {
		exits = append(exits, Exit{Name: direct, Kind: ExitKindDirect})
	}
	l.fillPings(ctx, exits)
	return exits, nil
}

// ExitIP queries the public IPv4 of a known exit interface on demand.
func (l *Live) ExitIP(ctx context.Context, iface string) (string, error) {
	iface = cleanIface(iface)
	known, err := l.knownExits(ctx)
	if err != nil {
		return "", err
	}
	if !validIface(iface) || !known[iface] {
		return "", fmt.Errorf("%w: %s", ErrUnknownInterface, iface)
	}
	out, err := l.run.exitIP(ctx, iface)
	return parseExitIP(out, err)
}

func (l *Live) interfaces(ctx context.Context) ([]awgInterface, error) {
	out, err := l.run.dump(ctx)
	if err != nil {
		return nil, fmt.Errorf("awg dump: %w", err)
	}
	return parseDump(out), nil
}

func (l *Live) linkSet(ctx context.Context) (map[string]bool, error) {
	out, err := l.run.links(ctx)
	if err != nil {
		return nil, fmt.Errorf("ip link: %w", err)
	}
	set := make(map[string]bool)
	for _, name := range parseLinkNames(out) {
		set[name] = true
	}
	return set, nil
}

func (l *Live) knownExits(ctx context.Context) (map[string]bool, error) {
	ifaces, err := l.interfaces(ctx)
	if err != nil {
		return nil, err
	}
	known := make(map[string]bool)
	for _, iface := range ifaces {
		if iface.isTunnel() {
			known[iface.Name] = true
		}
	}
	direct := cleanIface(l.cfg.DirectInterface)
	if links, err := l.linkSet(ctx); err == nil && links[direct] {
		known[direct] = true
	}
	return known, nil
}

func (l *Live) fillPings(ctx context.Context, exits []Exit) {
	type job struct {
		exitIdx int
		pingIdx int
		iface   string
		target  string
	}
	jobs := make([]job, 0)
	for i := range exits {
		targets := l.cfg.PingTargetsFor(exits[i].Name)
		exits[i].Pings = make([]PingResult, len(targets))
		for j, target := range targets {
			exits[i].Pings[j] = PingResult{Target: target}
			jobs = append(jobs, job{exitIdx: i, pingIdx: j, iface: exits[i].Name, target: target})
		}
	}
	var wg sync.WaitGroup
	for _, item := range jobs {
		wg.Go(func() {
			res := l.pingOne(ctx, item.iface, item.target)
			res.Target = item.target
			exits[item.exitIdx].Pings[item.pingIdx] = res
		})
	}
	wg.Wait()
}

func (l *Live) pingOne(ctx context.Context, iface, target string) PingResult {
	if !validIface(iface) || !validTarget(target) {
		return PingResult{Target: target, Error: "invalid probe"}
	}
	out, err := l.run.ping(ctx, iface, target)
	res := parsePing(out, err)
	res.Target = target
	return res
}
