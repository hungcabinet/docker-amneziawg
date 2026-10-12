package status

import (
	"errors"
	"testing"
	"time"
)

func TestParsePing_SuccessFromSnapshot(t *testing.T) {
	t.Parallel()

	const out = `PING 1.1.1.1 (1.1.1.1) from 10.13.13.7 wgup1: 56(84) bytes of data.
64 bytes from 1.1.1.1: icmp_seq=1 ttl=55 time=18.0 ms

--- 1.1.1.1 ping statistics ---
1 packets transmitted, 1 received, 0% packet loss, time 0ms
rtt min/avg/max/mdev = 17.991/17.991/17.991/0.000 ms
`
	got := parsePing(out, nil)
	if !got.OK {
		t.Fatalf("parsePing() = %+v", got)
	}
	if got.Latency != 18*time.Millisecond {
		t.Fatalf("Latency = %v, want 18ms", got.Latency)
	}
}

func TestParsePing_FailureFromSnapshot(t *testing.T) {
	t.Parallel()

	const out = `PING 1.1.1.1 (1.1.1.1) from 10.13.13.1 wg0: 56(84) bytes of data.
From 10.13.13.1 icmp_seq=1 Destination Host Unreachable
ping: sendmsg: Required key not available

--- 1.1.1.1 ping statistics ---
1 packets transmitted, 0 received, +1 errors, 100% packet loss, time 0ms
`
	got := parsePing(out, errors.New("exit status 1"))
	if got.OK || got.Error == "" {
		t.Fatalf("parsePing() = %+v", got)
	}
	if got.Error != "From 10.13.13.1 icmp_seq=1 Destination Host Unreachable" {
		t.Fatalf("Error = %q", got.Error)
	}
}

func TestParseExitIP(t *testing.T) {
	t.Parallel()

	ip, err := parseExitIP("192.0.2.10\n", nil, familyV4)
	if err != nil || ip != "192.0.2.10" {
		t.Fatalf("parseExitIP(v4) = %q, %v", ip, err)
	}
	ip, err = parseExitIP("2001:DB8:0::10\n", nil, familyV6)
	if err != nil || ip != "2001:db8::10" {
		t.Fatalf("parseExitIP(v6) = %q, %v", ip, err)
	}
	if _, err := parseExitIP("", errors.New("exit status 7"), familyV4); err == nil {
		t.Fatal("expected curl error")
	}
	if _, err := parseExitIP("", nil, familyV4); err == nil {
		t.Fatal("expected empty response error")
	}
	if _, err := parseExitIP("<html>", nil, familyV4); err == nil {
		t.Fatal("expected non-address error")
	}
	if _, err := parseExitIP("192.0.2.10", nil, familyV6); err == nil {
		t.Fatal("expected family mismatch for v4 answer to a v6 lookup")
	}
	if _, err := parseExitIP("2001:db8::10", nil, familyV4); err == nil {
		t.Fatal("expected family mismatch for v6 answer to a v4 lookup")
	}
}
