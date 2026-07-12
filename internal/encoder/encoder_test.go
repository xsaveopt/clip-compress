package encoder

import (
	"slices"
	"testing"

	"github.com/xsaveopt/clip-compress/internal/config"
)

func argValue(args []string, flag string) string {
	i := slices.Index(args, flag)
	if i < 0 || i+1 >= len(args) {
		return ""
	}
	return args[i+1]
}

func av1Encoder() *Encoder  { return &Encoder{Profile: Profiles[config.CodecAV1]} }
func hevcEncoder() *Encoder { return &Encoder{Profile: Profiles[config.CodecHEVC]} }

func TestBuildArgsBitrate(t *testing.T) {
	e := av1Encoder()
	args := e.buildArgs("in.mp4", "out.webm", Options{VideoBitrateK: 1900})

	if v := argValue(args, "-b:v"); v != "1900k" {
		t.Fatalf("video bitrate = %q, want 1900k", v)
	}
	if v := argValue(args, "-rc"); v != "cbr" {
		t.Fatalf("rc = %q, want cbr", v)
	}
	if v := argValue(args, "-multipass"); v != "fullres" {
		t.Fatalf("multipass = %q, want fullres", v)
	}
	if v := argValue(args, "-preset"); v != "p7" {
		t.Fatalf("preset = %q, want p7", v)
	}
}

func TestBuildArgsBitrateFloor(t *testing.T) {
	e := av1Encoder()
	args := e.buildArgs("in.mp4", "out.webm", Options{VideoBitrateK: 0})
	if v := argValue(args, "-b:v"); v != "200k" {
		t.Fatalf("video bitrate = %q, want floor 200k", v)
	}
}

func TestCopiesMetadata(t *testing.T) {
	e := av1Encoder()
	args := e.buildArgs("in.mp4", "out.webm", Options{VideoBitrateK: 1900})
	if v := argValue(args, "-map_metadata"); v != "0" {
		t.Fatalf("map_metadata = %q, want 0", v)
	}
}

func TestAV1UsesOpus(t *testing.T) {
	e := av1Encoder()
	args := e.buildArgs("in.mp4", "out.webm", Options{VideoBitrateK: 1900})
	if v := argValue(args, "-c:a"); v != "libopus" {
		t.Fatalf("audio codec = %q, want libopus", v)
	}
	if v := argValue(args, "-ac"); v != "2" {
		t.Fatalf("channels = %q, want 2", v)
	}
	if slices.Contains(args, "-movflags") {
		t.Fatal("webm output should not use -movflags")
	}
}

func TestHEVCUsesAACInMP4(t *testing.T) {
	e := hevcEncoder()
	args := e.buildArgs("in.mp4", "out.mp4", Options{VideoBitrateK: 1900})
	if v := argValue(args, "-c:a"); v != "aac" {
		t.Fatalf("audio codec = %q, want aac", v)
	}
	if v := argValue(args, "-c:v"); v != "hevc_nvenc" {
		t.Fatalf("video codec = %q, want hevc_nvenc", v)
	}
	if !slices.Contains(args, "-movflags") {
		t.Fatal("mp4 output should use -movflags +faststart")
	}
}
