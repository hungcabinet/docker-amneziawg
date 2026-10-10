package webui

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{2048, "2.0 KiB"},
		{12 * 1024 * 1024, "12.0 MiB"},
	}
	for _, tt := range tests {
		if got := formatBytes(tt.in); got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatHandshakeAt(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	if got := formatHandshakeAt(time.Time{}, now); got != "never" {
		t.Errorf("zero = %q", got)
	}
	if got := formatHandshakeAt(now.Add(-30*time.Second), now); got != "just now" {
		t.Errorf("recent = %q", got)
	}
	if got := formatHandshakeAt(now.Add(-12*time.Minute), now); got != "12 min ago" {
		t.Errorf("minutes = %q", got)
	}
}
