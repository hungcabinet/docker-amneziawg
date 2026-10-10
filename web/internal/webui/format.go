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
		return fmt.Sprintf("%.1f ГиБ", float64(n)/float64(giB))
	case n >= miB:
		return fmt.Sprintf("%.1f МиБ", float64(n)/float64(miB))
	case n >= kiB:
		return fmt.Sprintf("%.1f КиБ", float64(n)/float64(kiB))
	default:
		return fmt.Sprintf("%d Б", n)
	}
}

func formatHandshake(t time.Time) string {
	return formatHandshakeAt(t, time.Now())
}

func formatHandshakeAt(t, now time.Time) string {
	if t.IsZero() {
		return "никогда"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "только что"
	case d < time.Hour:
		return fmt.Sprintf("%d мин назад", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d ч назад", int(d.Hours()))
	default:
		return t.Local().Format("02.01.2006 15:04")
	}
}

func formatLatency(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d мс", d.Milliseconds())
}

func formatKind(kind status.ExitKind) string {
	switch kind {
	case status.ExitKindDirect:
		return "прямое"
	default:
		return "туннель"
	}
}
