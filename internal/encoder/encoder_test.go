package encoder

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func TestOptionsFromConfig(t *testing.T) {
	cfg := &config.Config{}
	out := filepath.Join(t.TempDir(), "out")
	cfg.SetOutputDir(out)
	cfg.SetVideoBitrateK(3100)

	got := OptionsFromConfig(cfg)
	if got.OutputDir != out || got.VideoBitrateK != 3100 {
		t.Errorf("OptionsFromConfig = %+v, want OutputDir %q and VideoBitrateK 3100", got, out)
	}
}

func TestEncodeWithoutAProfileFails(t *testing.T) {
	e := &Encoder{FFmpegPath: "ffmpeg"}
	out := filepath.Join(t.TempDir(), "out")

	_, err := e.Encode(context.Background(), "clip.mp4", Options{OutputDir: out})
	if err == nil || !strings.Contains(err.Error(), "no NVENC encoder resolved") {
		t.Fatalf("Encode error = %v, want the unresolved encoder message", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("output dir stat err = %v, want it never created", err)
	}
}

func TestEncodeFailsWhenTheOutputDirCannotBeCreated(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "out")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	e := av1Encoder()
	e.FFmpegPath = filepath.Join(dir, "missing-ffmpeg")

	if _, err := e.Encode(context.Background(), "clip.mp4", Options{OutputDir: filepath.Join(blocker, "sub")}); err == nil {
		t.Fatal("Encode should fail when the output dir cannot be created")
	}
}

func TestEncodeReusesAnExistingOutput(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	existing := filepath.Join(out, "clip (av1).webm")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(existing, []byte("done"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	e := av1Encoder()
	e.FFmpegPath = filepath.Join(dir, "missing-ffmpeg")

	got, err := e.Encode(context.Background(), filepath.Join(dir, "in", "clip.mp4"), Options{OutputDir: out})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if got != existing {
		t.Errorf("Encode = %q, want %q", got, existing)
	}
}

func TestEncodeMissingFFmpegFails(t *testing.T) {
	dir := t.TempDir()
	e := hevcEncoder()
	e.FFmpegPath = filepath.Join(dir, "missing-ffmpeg")

	_, err := e.Encode(context.Background(), "clip.mkv", Options{OutputDir: filepath.Join(dir, "out")})
	if err == nil || !strings.Contains(err.Error(), "ffmpeg failed") {
		t.Fatalf("Encode error = %v, want an ffmpeg failure", err)
	}
}

func TestEncodeRunsFFmpegAndReturnsTheOutput(t *testing.T) {
	dir := t.TempDir()
	e := hevcEncoder()
	e.FFmpegPath = fakeFFmpeg(t, "encode", "")
	input := filepath.Join(dir, "in", "Game Clip.mkv")
	out := filepath.Join(dir, "out")

	got, err := e.Encode(context.Background(), input, Options{OutputDir: out, VideoBitrateK: 2500})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if want := filepath.Join(out, "Game Clip (hevc).mp4"); got != want {
		t.Errorf("Encode = %q, want %q", got, want)
	}
	raw, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	args := strings.Split(string(raw), "\n")
	if v := argValue(args, "-i"); v != input {
		t.Errorf("ffmpeg -i = %q, want %q", v, input)
	}
	if v := argValue(args, "-b:v"); v != "2500k" {
		t.Errorf("ffmpeg -b:v = %q, want 2500k", v)
	}
}

func TestEncodeFailureCarriesTheStderrTail(t *testing.T) {
	e := av1Encoder()
	e.FFmpegPath = fakeFFmpeg(t, "fail", "")

	_, err := e.Encode(context.Background(), "clip.mp4", Options{OutputDir: filepath.Join(t.TempDir(), "out")})
	if err == nil {
		t.Fatal("Encode should fail when ffmpeg exits non-zero")
	}
	msg := err.Error()
	if !strings.Contains(msg, "ffmpeg failed") || !strings.Contains(msg, "…") || !strings.Contains(msg, fakeTailMarker) {
		t.Errorf("error = %q, want the ffmpeg failure with a truncated stderr tail", msg)
	}
	if strings.Contains(msg, strings.Repeat("x", 700)) {
		t.Error("error carries the whole stderr, want it cut to the tail")
	}
}

func TestEncodeHonoursACancelledContext(t *testing.T) {
	e := av1Encoder()
	e.FFmpegPath = fakeFFmpeg(t, "encode", "")
	out := filepath.Join(t.TempDir(), "out")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := e.Encode(ctx, "clip.mp4", Options{OutputDir: out}); err == nil {
		t.Fatal("Encode should fail on a cancelled context")
	}
	if _, err := os.Stat(filepath.Join(out, "clip (av1).webm")); !os.IsNotExist(err) {
		t.Errorf("output stat err = %v, want no output", err)
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.mp4")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if !fileExists(file) {
		t.Error("fileExists(file) = false, want true")
	}
	if fileExists(dir) {
		t.Error("fileExists(dir) = true, want false")
	}
	if fileExists(filepath.Join(dir, "missing")) {
		t.Error("fileExists(missing) = true, want false")
	}
}

func TestTail(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"", 5, ""},
		{"abc", 5, "abc"},
		{"abcde", 5, "abcde"},
		{"abcdefgh", 3, "…fgh"},
	}
	for _, c := range cases {
		if got := tail(c.in, c.n); got != c.want {
			t.Errorf("tail(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}
