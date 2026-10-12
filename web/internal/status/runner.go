package status

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type ipFamily string

const (
	familyV4 ipFamily = "4"
	familyV6 ipFamily = "6"
)

var exitIPURLs = map[ipFamily]string{
	familyV4: "https://api.ipify.org",
	familyV6: "https://api6.ipify.org",
}

type runner interface {
	dump(ctx context.Context) (string, error)
	links(ctx context.Context) (string, error)
	ping(ctx context.Context, iface, target string) (string, error)
	exitIP(ctx context.Context, iface string, family ipFamily) (string, error)
}

type hostRunner struct{}

func (hostRunner) dump(ctx context.Context) (string, error) {
	return runCombined(ctx, "awg", "show", "all", "dump")
}

func (hostRunner) links(ctx context.Context) (string, error) {
	return runCombined(ctx, "ip", "-br", "link")
}

func (hostRunner) ping(ctx context.Context, iface, target string) (string, error) {
	return runCombined(ctx, "ping", "-c", "1", "-W", "2", "-I", iface, target)
}

func (hostRunner) exitIP(ctx context.Context, iface string, family ipFamily) (string, error) {
	return runStdout(ctx, "curl", "-"+string(family), "-m", "5", "-sS", "--interface", iface, exitIPURLs[family])
}

func runCombined(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runStdout(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return stdout.String(), fmt.Errorf("%s: %w", msg, err)
		}
		return stdout.String(), err
	}
	return stdout.String(), nil
}
