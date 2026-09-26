package encoder

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
)

const (
	fakeModeEnv     = "CLIP_COMPRESS_FAKE_FFMPEG"
	fakeEncodersEnv = "CLIP_COMPRESS_FAKE_ENCODERS"
	fakeTailMarker  = "tail-marker"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv(fakeModeEnv); mode != "" {
		os.Exit(runFakeFFmpeg(mode, os.Args[1:]))
	}
	os.Exit(m.Run())
}

func runFakeFFmpeg(mode string, args []string) int {
	switch mode {
	case "encode":
		if err := os.WriteFile(args[len(args)-1], []byte(strings.Join(args, "\n")), 0o644); err != nil {
			fmt.Fprint(os.Stderr, err)
			return 1
		}
		return 0
	case "fail":
		fmt.Fprint(os.Stderr, strings.Repeat("x", 1000)+fakeTailMarker)
		return 1
	case "probe-ok", "probe-broken":
		if slices.Contains(args, "-encoders") {
			fmt.Print(os.Getenv(fakeEncodersEnv))
			return 0
		}
		if mode == "probe-broken" {
			fmt.Fprint(os.Stderr, "  no capable device  \n")
			return 1
		}
		return 0
	}
	return 2
}

func fakeFFmpeg(t *testing.T, mode, encoders string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable: %v", err)
	}
	t.Setenv(fakeModeEnv, mode)
	t.Setenv(fakeEncodersEnv, encoders)
	return exe
}
