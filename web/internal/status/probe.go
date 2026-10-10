package status

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var pingTimeRe = regexp.MustCompile(`time[=<]([0-9]+(?:\.[0-9]+)?)\s*ms`)

var (
	ifaceNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]*$`)
	targetRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)
)

func parsePing(output string, err error) PingResult {
	if m := pingTimeRe.FindStringSubmatch(output); len(m) == 2 {
		ms, perr := strconv.ParseFloat(m[1], 64)
		if perr == nil {
			return PingResult{
				OK:      true,
				Latency: time.Duration(ms * float64(time.Millisecond)),
			}
		}
	}
	return PingResult{
		OK:    false,
		Error: pingError(output, err),
	}
}

func pingError(output string, err error) string {
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || skipPingLine(line) {
			continue
		}
		return line
	}
	if err != nil {
		return err.Error()
	}
	return "ping failed"
}

func skipPingLine(line string) bool {
	switch {
	case strings.HasPrefix(line, "PING "):
		return true
	case strings.HasPrefix(line, "--- "):
		return true
	case strings.Contains(line, "packets transmitted"):
		return true
	case strings.HasPrefix(line, "rtt "):
		return true
	default:
		return false
	}
}

func parseExitIP(stdout string, err error) (string, error) {
	ip := strings.TrimSpace(stdout)
	if err != nil {
		return "", fmt.Errorf("exit ip: %w", err)
	}
	if ip == "" {
		return "", errors.New("exit ip: empty response")
	}
	if strings.ContainsFunc(ip, unicode.IsSpace) {
		return "", errors.New("exit ip: unexpected response")
	}
	return ip, nil
}

func validIface(name string) bool {
	return ifaceNameRe.MatchString(name)
}

func validTarget(target string) bool {
	return targetRe.MatchString(target)
}
