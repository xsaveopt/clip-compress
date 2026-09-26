package ui

import (
	"testing"

	"github.com/xsaveopt/clip-compress/internal/config"
)

func TestTrayTip(t *testing.T) {
	for status, want := range map[string]string{
		"starting…":         "ClipCompress — starting…",
		"watching (av1)":    "ClipCompress — watching (av1)",
		"encoding clip.mp4": "ClipCompress — encoding clip.mp4",
	} {
		tr := &Tray{status: status}
		if got := tr.tip(); got != want {
			t.Errorf("tip() with status %q = %q, want %q", status, got, want)
		}
	}
}

func TestTrayPauseLabelFollowsConfig(t *testing.T) {
	cfg := &config.Config{}
	tr := &Tray{cfg: cfg}

	if got := tr.pauseLabel(); got != "Pause" {
		t.Errorf("pauseLabel() = %q, want Pause while running", got)
	}
	cfg.SetPaused(true)
	if got := tr.pauseLabel(); got != "Resume" {
		t.Errorf("pauseLabel() = %q, want Resume while paused", got)
	}
}
