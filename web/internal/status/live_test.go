package status

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hungcabinet/docker-amneziawg/web/internal/config"
)

type fakeRunner struct {
	dumpOut string
	linkOut string
	dumpErr error
	linkErr error
	pings   map[string]cmdResult
	ips     map[string]cmdResult
}

type cmdResult struct {
	out string
	err error
}

func (f fakeRunner) dump(context.Context) (string, error) {
	return f.dumpOut, f.dumpErr
}

func (f fakeRunner) links(context.Context) (string, error) {
	return f.linkOut, f.linkErr
}

func (f fakeRunner) ping(_ context.Context, iface, target string) (string, error) {
	res, ok := f.pings[iface+" "+target]
	if !ok {
		return "", errors.New("unexpected ping")
	}
	return res.out, res.err
}

func (f fakeRunner) exitIP(_ context.Context, iface string) (string, error) {
	res, ok := f.ips[iface]
	if !ok {
		return "", errors.New("unexpected exit ip")
	}
	return res.out, res.err
}

func TestLive_FixtureScene(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		ConfigDir:       fixtureConfigDir(t),
		DirectInterface: "eth0",
		PingTargets:     []string{"1.1.1.1", "8.8.8.8"},
	}
	live := NewLive(cfg)
	live.now = func() time.Time {
		return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	}
	live.run = fixtureRunner()
	ctx := context.Background()

	server, err := live.Server(ctx)
	if err != nil {
		t.Fatalf("Server() error = %v", err)
	}
	if !server.Up || server.Interface != "wg0" || server.ListenPort != 51820 {
		t.Fatalf("Server() = %+v", server)
	}

	peers, err := live.Peers(ctx)
	if err != nil {
		t.Fatalf("Peers() error = %v", err)
	}
	if len(peers) != 6 {
		t.Fatalf("len(Peers) = %d, want 6", len(peers))
	}
	wantNames := []string{"bunny", "bear", "deer", "fox", "owl", "KKKKKKKK"}
	for i, name := range wantNames {
		if peers[i].Name != name {
			t.Fatalf("peer[%d].Name = %q, want %q", i, peers[i].Name, name)
		}
		if peers[i].Online || !peers[i].LastHandshake.IsZero() {
			t.Fatalf("peer %s should be offline never-seen: %+v", name, peers[i])
		}
	}
	if !strings.Contains(peers[0].Config, "PrivateKey = REDACTED") {
		t.Fatalf("bunny config = %q", peers[0].Config)
	}
	if peers[1].Config != "" {
		t.Fatalf("bear config should be empty, got %q", peers[1].Config)
	}

	exits, err := live.Exits(ctx)
	if err != nil {
		t.Fatalf("Exits() error = %v", err)
	}
	if len(exits) != 3 {
		t.Fatalf("len(Exits) = %d, want 3: %+v", len(exits), exits)
	}
	if exits[0].Name != "wgup1" || exits[0].Kind != ExitKindTunnel {
		t.Fatalf("exit[0] = %+v", exits[0])
	}
	if exits[1].Name != "wgup2" || exits[1].Kind != ExitKindTunnel {
		t.Fatalf("exit[1] = %+v", exits[1])
	}
	if exits[2].Name != "eth0" || exits[2].Kind != ExitKindDirect {
		t.Fatalf("exit[2] = %+v", exits[2])
	}
	for _, exit := range exits {
		if exit.Name == "wg0" {
			t.Fatal("server interface must not be an exit")
		}
		if len(exit.Pings) != 2 || !exit.Pings[0].OK || exit.Pings[0].Latency != 18*time.Millisecond {
			t.Fatalf("exit %s pings = %+v", exit.Name, exit.Pings)
		}
	}

	ip, err := live.ExitIP(ctx, "wgup1")
	if err != nil || ip != "192.0.2.10" {
		t.Fatalf("ExitIP(wgup1) = %q, %v", ip, err)
	}
	ip, err = live.ExitIP(ctx, "eth0@if2")
	if err != nil || ip != "192.0.2.1" {
		t.Fatalf("ExitIP(eth0@if2) = %q, %v", ip, err)
	}
	if _, err := live.ExitIP(ctx, "wg0"); !errors.Is(err, ErrUnknownInterface) {
		t.Fatalf("ExitIP(wg0) error = %v", err)
	}
	if _, err := live.ExitIP(ctx, "missing"); !errors.Is(err, ErrUnknownInterface) {
		t.Fatalf("ExitIP(missing) error = %v", err)
	}
}

func TestLive_ServerDown(t *testing.T) {
	t.Parallel()

	live := NewLive(config.Config{DirectInterface: "eth0"})
	live.run = fakeRunner{
		dumpOut: "wgup1\tREDACTED\tGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG=\t40001\t5\n" +
			"wgup1\tHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHH=\tREDACTED\t192.0.2.10:51820\t0.0.0.0/0\t1\t1\t1\toff\n",
		linkOut: "eth0 UP\n",
		pings: map[string]cmdResult{
			"wgup1 1.1.1.1": {out: "64 bytes from 1.1.1.1: icmp_seq=1 ttl=55 time=18.0 ms"},
			"wgup1 8.8.8.8": {out: "64 bytes from 8.8.8.8: icmp_seq=1 ttl=55 time=18.0 ms"},
			"eth0 1.1.1.1":     {out: "64 bytes from 1.1.1.1: icmp_seq=1 ttl=55 time=18.0 ms"},
			"eth0 8.8.8.8":     {out: "64 bytes from 8.8.8.8: icmp_seq=1 ttl=55 time=18.0 ms"},
		},
	}
	ctx := context.Background()
	server, err := live.Server(ctx)
	if err != nil {
		t.Fatalf("Server() error = %v", err)
	}
	if server.Up {
		t.Fatalf("Server() = %+v, want down", server)
	}
	peers, err := live.Peers(ctx)
	if err != nil {
		t.Fatalf("Peers() error = %v", err)
	}
	if len(peers) != 0 {
		t.Fatalf("Peers() = %+v, want none", peers)
	}
}

func TestLive_OnlinePeer(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "peer_wolf", "publickey-peer_wolf"), "LLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLL=\n")
	dump := "wg0\tREDACTED\tAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\t51820\t8\n" +
		"wg0\tLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLL=\tREDACTED\t(none)\t10.13.13.8/32\t" +
		strconv.FormatInt(now.Add(-20*time.Second).Unix(), 10) + "\t100\t200\t25\n"
	live := NewLive(config.Config{ConfigDir: root, DirectInterface: "eth0"})
	live.now = func() time.Time { return now }
	live.run = fakeRunner{dumpOut: dump}
	peers, err := live.Peers(context.Background())
	if err != nil {
		t.Fatalf("Peers() error = %v", err)
	}
	if len(peers) != 1 {
		t.Fatalf("len(Peers) = %d", len(peers))
	}
	if peers[0].Name != "wolf" || !peers[0].Online || peers[0].RxBytes != 100 || peers[0].TxBytes != 200 {
		t.Fatalf("peer = %+v", peers[0])
	}
}

func TestLive_DumpError(t *testing.T) {
	t.Parallel()

	live := NewLive(config.Config{DirectInterface: "eth0"})
	live.run = fakeRunner{dumpErr: errors.New("awg: not found")}
	if _, err := live.Server(context.Background()); err == nil {
		t.Fatal("Server() error = nil")
	}
}

func fixtureRunner() fakeRunner {
	ok := cmdResult{out: "64 bytes from 1.1.1.1: icmp_seq=1 ttl=55 time=18.0 ms"}
	return fakeRunner{
		dumpOut: mustRead("testdata/dump.txt"),
		linkOut: mustRead("testdata/ip-link.txt"),
		pings: map[string]cmdResult{
			"wgup1 1.1.1.1": ok,
			"wgup1 8.8.8.8": ok,
			"wgup2 1.1.1.1": ok,
			"wgup2 8.8.8.8": ok,
			"eth0 1.1.1.1":     ok,
			"eth0 8.8.8.8":     ok,
		},
		ips: map[string]cmdResult{
			"wgup1": {out: "192.0.2.10\n"},
			"wgup2": {out: "198.51.100.20\n"},
			"eth0":     {out: "192.0.2.1\n"},
		},
	}
}

func fixtureConfigDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	peers := []string{
		"peer_bunny",
		"peer_bear",
		"peer_deer",
		"peer_fox",
		"peer_owl",
		"peer_wolf",
	}
	for _, dir := range peers {
		mustCopy(
			t,
			filepath.Join("testdata", "publickeys", dir),
			filepath.Join(root, dir, "publickey-"+dir),
		)
	}
	mustCopy(
		t,
		filepath.Join("testdata", "peer_bunny.conf.txt"),
		filepath.Join(root, "peer_bunny", "peer_bunny.conf"),
	)
	return root
}

func mustCopy(t *testing.T, src, dest string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, dest, string(b))
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(b)
}
