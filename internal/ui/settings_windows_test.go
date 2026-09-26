package ui

import "testing"

func TestToggleValue(t *testing.T) {
	if got := toggleValue(true); got != "ON" {
		t.Errorf("toggleValue(true) = %q, want ON", got)
	}
	if got := toggleValue(false); got != "OFF" {
		t.Errorf("toggleValue(false) = %q, want OFF", got)
	}
}

func TestParseInt(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"2500", 2500},
		{"0", 0},
		{"", 1900},
		{"abc", 1900},
		{"12k", 1900},
		{" 12", 1900},
		{"99999999999999999999", 1900},
	}
	for _, c := range cases {
		if got := parseInt(c.in, 1900); got != c.want {
			t.Errorf("parseInt(%q, 1900) = %d, want %d", c.in, got, c.want)
		}
	}
}
