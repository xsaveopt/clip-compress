package watcher

import (
	"path/filepath"
	"testing"
)

func TestIsWithin(t *testing.T) {
	out := filepath.Join("home", "Videos", "ClipCompress")
	cases := []struct {
		path string
		want bool
	}{
		{filepath.Join(out, "clip (av1).mp4"), true},
		{filepath.Join(out, "sub", "x.mp4"), true},
		{out, true},
		{filepath.Join("home", "Videos", "Game", "clip.mp4"), false},
		{filepath.Join("home", "Videos"), false},
	}
	for _, c := range cases {
		if got := isWithin(c.path, out); got != c.want {
			t.Errorf("isWithin(%q, %q) = %v, want %v", c.path, out, got, c.want)
		}
	}
}
