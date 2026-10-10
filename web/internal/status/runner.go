package status

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

const exitIPURL = "https://api.ipify.org"

type runner interface {
	dump(ctx context.Context) (string, error)
	links(ctx context.Context) (string, error)
	ping(ctx context.Context, iface, target string) (string, error)
	exitIP(ctx context.Context, iface string) (string, error)
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

func (hostRunner) exitIP(ctx context.Context, iface string) (string, error) {
	return runStdout(ctx, "curl", "-4", "-m", "5", "-sS", "--interface", iface, exitIPURL)
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
