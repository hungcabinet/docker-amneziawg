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

	ip, err := parseExitIP("192.0.2.10\n", nil)
	if err != nil || ip != "192.0.2.10" {
		t.Fatalf("parseExitIP() = %q, %v", ip, err)
	}
	if _, err := parseExitIP("", errors.New("exit status 7")); err == nil {
		t.Fatal("expected curl error")
	}
	if _, err := parseExitIP("", nil); err == nil {
		t.Fatal("expected empty response error")
	}
}
