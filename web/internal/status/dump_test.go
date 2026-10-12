package status

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDump_Fixture(t *testing.T) {
	t.Parallel()

	ifaces := parseDump(readTestdata(t, "dump.txt"))
	if len(ifaces) != 3 {
		t.Fatalf("len(ifaces) = %d, want 3", len(ifaces))
	}

	srv, ok := findServer(ifaces)
	if !ok {
		t.Fatal("expected server interface")
	}
	if srv.Name != "wg0" || srv.ListenPort != 51820 {
		t.Fatalf("server = %+v", srv)
	}
	if len(srv.Peers) != 6 {
		t.Fatalf("len(server peers) = %d, want 6", len(srv.Peers))
	}
	if hasEndpoint(srv.Peers[0].Endpoint) {
		t.Fatalf("server peer endpoint = %q", srv.Peers[0].Endpoint)
	}

	if ifaces[1].Name != "wgup1" || !ifaces[1].isTunnel() || ifaces[1].ListenPort != 40001 {
		t.Fatalf("wgup1 = %+v", ifaces[1])
	}
	if ifaces[1].Peers[0].Endpoint != "192.0.2.10:51820" {
		t.Fatalf("wgup1 endpoint = %q", ifaces[1].Peers[0].Endpoint)
	}
	if ifaces[1].Peers[0].Rx != 1000 || ifaces[1].Peers[0].Tx != 2000 {
		t.Fatalf("wgup1 transfer = %+v", ifaces[1].Peers[0])
	}
	if ifaces[2].Name != "wgup2" || !ifaces[2].isTunnel() {
		t.Fatalf("wgup2 = %+v", ifaces[2])
	}
	if ifaces[0].isTunnel() || ifaces[1].isServer() {
		t.Fatal("classification mixed listen-port with role")
	}
}

func TestParseDump_ListenPortDoesNotMarkServer(t *testing.T) {
	t.Parallel()

	dump := "wgclient\tREDACTED\tAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\t51820\t1\n" +
		"wgclient\tHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHHH=\tREDACTED\t192.0.2.10:51820\t0.0.0.0/0\t1\t1\t1\toff\n"
	ifaces := parseDump(dump)
	if len(ifaces) != 1 || ifaces[0].isServer() || !ifaces[0].isTunnel() {
		t.Fatalf("got %+v", ifaces)
	}
	if _, ok := findServer(ifaces); ok {
		t.Fatal("client tunnel with listen port must not be the server")
	}
}

func TestAssignRoles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "wg_confs"), 0o755); err != nil {
		t.Fatal(err)
	}
	confs := map[string]string{
		"wg0.conf":   "[Interface]\nListenPort = 51820\n\n[Peer]\n# Endpoint = example.com:51820\nAllowedIPs = 10.13.13.2/32\n",
		"wgup1.conf": "[Interface]\n\n[Peer]\n  Endpoint=[2001:db8::10]:51820\n",
	}
	for name, text := range confs {
		if err := os.WriteFile(filepath.Join(root, "wg_confs", name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ifaces := []awgInterface{
		{Name: "wg0", Peers: []awgPeer{{Endpoint: "203.0.113.25:41820"}}},
		{Name: "wgup1", Peers: []awgPeer{{Endpoint: "(none)"}}},
		{Name: "wgnoconf", Peers: []awgPeer{{Endpoint: "192.0.2.10:51820"}}},
	}
	assignRoles(ifaces, root)
	if !ifaces[0].isServer() || ifaces[0].isTunnel() {
		t.Fatalf("wg0 with a connected peer must stay the server: %+v", ifaces[0])
	}
	if !ifaces[1].isTunnel() || ifaces[1].isServer() {
		t.Fatalf("wgup1 conf has an Endpoint: %+v", ifaces[1])
	}
	if ifaces[2].role != roleUnknown || !ifaces[2].isTunnel() {
		t.Fatalf("interface without conf must fall back to dump endpoints: %+v", ifaces[2])
	}
}

func TestHandshakeState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	if last, online := handshakeState(0, now); online || !last.IsZero() {
		t.Fatalf("zero handshake: last=%v online=%v", last, online)
	}
	recent := now.Add(-30 * time.Second).Unix()
	if last, online := handshakeState(recent, now); !online || last.IsZero() {
		t.Fatalf("recent handshake: last=%v online=%v", last, online)
	}
	stale := now.Add(-10 * time.Minute).Unix()
	if last, online := handshakeState(stale, now); online || last.IsZero() {
		t.Fatalf("stale handshake: last=%v online=%v", last, online)
	}
}

func TestCleanIfaceAndLinks(t *testing.T) {
	t.Parallel()

	if got := cleanIface("eth0@if2"); got != "eth0" {
		t.Fatalf("cleanIface = %q", got)
	}
	names := parseLinkNames(readTestdata(t, "ip-link.txt"))
	want := []string{"lo", "eth0", "wg0", "wgup1", "wgup2"}
	if !stringSliceEqual(names, want) {
		t.Fatalf("parseLinkNames = %v, want %v", names, want)
	}
}

func TestDisplayPeerName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		dir  string
		want string
	}{
		{dir: "peer_bunny", want: "bunny"},
		{dir: "peer1", want: "peer1"},
		{dir: "peer", want: "peer"},
		{dir: "peer_", want: "peer_"},
	}
	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			t.Parallel()
			if got := displayPeerName(tt.dir); got != tt.want {
				t.Fatalf("displayPeerName(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
