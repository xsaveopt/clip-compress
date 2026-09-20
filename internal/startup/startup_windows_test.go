package startup

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func stubRegistry(t *testing.T) (*[]string, *[]string, *error) {
	t.Helper()
	enabled := []string{}
	disabled := []string{}
	var ret error

	origEnable, origDisable := enable, disable
	enable = func(name string) error {
		enabled = append(enabled, name)
		return ret
	}
	disable = func(name string) error {
		disabled = append(disabled, name)
		return ret
	}
	t.Cleanup(func() { enable, disable = origEnable, origDisable })

	return &enabled, &disabled, &ret
}

func TestSyncEnablesWhenWanted(t *testing.T) {
	enabled, disabled, _ := stubRegistry(t)

	if err := Sync("ClipCompress", true); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(*enabled) != 1 || (*enabled)[0] != "ClipCompress" {
		t.Errorf("enabled = %v, want one ClipCompress", *enabled)
	}
	if len(*disabled) != 0 {
		t.Errorf("disabled = %v, want none", *disabled)
	}
}

func TestSyncDisablesWhenNotWanted(t *testing.T) {
	enabled, disabled, _ := stubRegistry(t)

	if err := Sync("ClipCompress", false); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(*disabled) != 1 || (*disabled)[0] != "ClipCompress" {
		t.Errorf("disabled = %v, want one ClipCompress", *disabled)
	}
	if len(*enabled) != 0 {
		t.Errorf("enabled = %v, want none", *enabled)
	}
}

func TestSyncPropagatesTheError(t *testing.T) {
	_, _, ret := stubRegistry(t)
	want := errors.New("registry is shut")
	*ret = want

	for _, tc := range []bool{true, false} {
		if err := Sync("ClipCompress", tc); !errors.Is(err, want) {
			t.Errorf("Sync(_, %v) = %v, want %v", tc, err, want)
		}
	}
}

func TestEnabledIsFalseForAnUnknownName(t *testing.T) {
	name := fmt.Sprintf("ClipCompressTest-%d-absent", os.Getpid())
	if Enabled(name) {
		t.Errorf("Enabled(%q) = true, want false", name)
	}
}

func TestDisableIsQuietForAnUnknownName(t *testing.T) {
	name := fmt.Sprintf("ClipCompressTest-%d-absent", os.Getpid())
	if err := Disable(name); err != nil {
		t.Errorf("Disable(%q) = %v, want nil", name, err)
	}
}

func TestRunKeyPath(t *testing.T) {
	const want = `Software\Microsoft\Windows\CurrentVersion\Run`
	if runKeyPath != want {
		t.Errorf("runKeyPath = %q, want %q", runKeyPath, want)
	}
}
