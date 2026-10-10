package webui

import (
	"fmt"
	"time"

	"github.com/hungcabinet/docker-amneziawg/web/internal/status"
)

func formatBytes(n int64) string {
	const (
		kiB = 1024
		miB = 1024 * kiB
		giB = 1024 * miB
	)
	switch {
	case n >= giB:
		return fmt.Sprintf("%.1f GiB", float64(n)/float64(giB))
	case n >= miB:
		return fmt.Sprintf("%.1f MiB", float64(n)/float64(miB))
	case n >= kiB:
		return fmt.Sprintf("%.1f KiB", float64(n)/float64(kiB))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func formatHandshake(t time.Time) string {
	return formatHandshakeAt(t, time.Now())
}

func formatHandshakeAt(t, now time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()))
	default:
		return t.Local().Format("02 Jan 2006 15:04")
	}
}

func formatLatency(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d ms", d.Milliseconds())
}

func formatKind(kind status.ExitKind) string {
	switch kind {
	case status.ExitKindDirect:
		return "direct"
	default:
		return "tunnel"
	}
}
