package status

import (
	"strconv"
	"strings"
	"time"
)

const peerDumpFields = 9

type awgInterface struct {
	Name       string
	ListenPort int
	Peers      []awgPeer
}

type awgPeer struct {
	PublicKey string
	Endpoint  string
	Handshake int64
	Rx        int64
	Tx        int64
}

func parseDump(text string) []awgInterface {
	ifaces := make([]awgInterface, 0)
	index := map[string]int{}

	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		idx, ok := index[name]
		if !ok {
			idx = len(ifaces)
			index[name] = idx
			ifaces = append(ifaces, awgInterface{Name: name})
		}
		if len(fields) == peerDumpFields {
			ifaces[idx].Peers = append(ifaces[idx].Peers, parsePeerLine(fields))
			continue
		}
		if len(fields) > 3 {
			if port, err := strconv.Atoi(fields[3]); err == nil {
				ifaces[idx].ListenPort = port
			}
		}
	}
	return ifaces
}

func parsePeerLine(fields []string) awgPeer {
	return awgPeer{
		PublicKey: fields[1],
		Endpoint:  fields[3],
		Handshake: parseInt64(fields[5]),
		Rx:        parseInt64(fields[6]),
		Tx:        parseInt64(fields[7]),
	}
}

func parseInt64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func (iface awgInterface) isServer() bool {
	if len(iface.Peers) == 0 {
		return false
	}
	for _, peer := range iface.Peers {
		if hasEndpoint(peer.Endpoint) {
			return false
		}
	}
	return true
}

func (iface awgInterface) isTunnel() bool {
	for _, peer := range iface.Peers {
		if hasEndpoint(peer.Endpoint) {
			return true
		}
	}
	return false
}

func hasEndpoint(endpoint string) bool {
	return endpoint != "" && endpoint != "(none)"
}

func findServer(ifaces []awgInterface) (awgInterface, bool) {
	for _, iface := range ifaces {
		if iface.isServer() {
			return iface, true
		}
	}
	return awgInterface{}, false
}

func handshakeState(unix int64, now time.Time) (time.Time, bool) {
	if unix <= 0 {
		return time.Time{}, false
	}
	last := time.Unix(unix, 0)
	return last, now.Sub(last) <= OnlineWindow
}

func cleanIface(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.IndexByte(name, '@'); i >= 0 {
		return name[:i]
	}
	return name
}

func parseLinkNames(text string) []string {
	names := make([]string, 0)
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := cleanIface(fields[0])
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}
