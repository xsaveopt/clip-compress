package main

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gen2brain/iup-go/iup"

	"github.com/xsaveopt/clip-compress/internal/config"
	"github.com/xsaveopt/clip-compress/internal/encoder"
	"github.com/xsaveopt/clip-compress/internal/ui"
	"github.com/xsaveopt/clip-compress/internal/watcher"
)

const (
	fakeFFmpegEnv   = "CLIP_COMPRESS_MAIN_FAKE_FFMPEG"
	fakeEncodersEnv = "CLIP_COMPRESS_MAIN_FAKE_ENCODERS"
	statusCode      = 0
	notifyCode      = 1
)

func TestMain(m *testing.M) {
	if os.Getenv(fakeFFmpegEnv) != "" {
		if slices.Contains(os.Args[1:], "-encoders") {
			fmt.Print(os.Getenv(fakeEncodersEnv))
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func fakeFFmpeg(t *testing.T, encoders string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable: %v", err)
	}
	t.Setenv(fakeFFmpegEnv, "1")
	t.Setenv(fakeEncodersEnv, encoders)
	return exe
}

var (
	uiOnce  sync.Once
	uiCalls = make(chan func())
)

func onUI(t *testing.T, fn func()) {
	t.Helper()
	uiOnce.Do(func() {
		ready := make(chan struct{})
		go func() {
			runtime.LockOSThread()
			iup.Open()
			close(ready)
			for f := range uiCalls {
				f()
			}
		}()
		<-ready
	})
	done := make(chan struct{})
	uiCalls <- func() {
		defer close(done)
		fn()
	}
	<-done
}

type trayMsg struct {
	code int
	text string
}

func status(s string) trayMsg  { return trayMsg{statusCode, s} }
func balloon(s string) trayMsg { return trayMsg{notifyCode, s} }

type trayRecorder struct {
	mu   sync.Mutex
	msgs []trayMsg
}

func (r *trayRecorder) snapshot() []trayMsg {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.msgs)
}

func recordingTray(t *testing.T, cfg *config.Config) (*ui.Tray, *trayRecorder) {
	t.Helper()
	rec := &trayRecorder{}
	var tray *ui.Tray
	var dlg iup.Ihandle
	onUI(t, func() {
		dlg = iup.Dialog(iup.Vbox())
		tray = ui.NewTray(dlg, cfg, image.NewRGBA(image.Rect(0, 0, 16, 16)), ui.TrayActions{})
		dlg.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(_ iup.Ihandle, s string, code int, _ any) int {
			rec.mu.Lock()
			rec.msgs = append(rec.msgs, trayMsg{code, s})
			rec.mu.Unlock()
			return iup.DEFAULT
		}))
	})
	t.Cleanup(func() {
		onUI(t, func() {
			dlg.SetAttribute("TRAY", "NO")
			iup.Destroy(dlg)
		})
	})
	return tray, rec
}

func expectTray(t *testing.T, rec *trayRecorder, want ...trayMsg) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for len(rec.snapshot()) < len(want) && time.Now().Before(deadline) {
		onUI(t, func() { iup.LoopStep() })
		time.Sleep(time.Millisecond)
	}
	for range 20 {
		onUI(t, func() { iup.LoopStep() })
	}
	if got := rec.snapshot(); !slices.Equal(got, want) {
		t.Errorf("tray messages = %+v, want %+v", got, want)
	}
}

func mediaConfig(t *testing.T, deleteOriginal, notifyOn bool) (*config.Config, string, string) {
	t.Helper()
	cfg := loadConfig(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in")
	out := filepath.Join(dir, "out")
	cfg.SetSourceDir(in)
	cfg.SetOutputDir(out)
	cfg.SetDeleteOriginal(deleteOriginal)
	cfg.SetNotify(notifyOn)
	return cfg, in, out
}

func av1Encoder() *encoder.Encoder {
	return &encoder.Encoder{Profile: encoder.Profiles[config.CodecAV1]}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestEncodeOneDeletesTheOriginalAfterSuccess(t *testing.T) {
	cfg, in, out := mediaConfig(t, true, true)
	tray, rec := recordingTray(t, cfg)
	src := filepath.Join(in, "clip.mp4")
	writeFile(t, src, "raw")
	writeFile(t, filepath.Join(out, "clip (av1).webm"), "encoded")

	encodeOne(context.Background(), cfg, av1Encoder(), tray, src)

	if exists(src) {
		t.Error("original should be deleted after a successful encode")
	}
	expectTray(t, rec,
		status("encoding clip.mp4"),
		balloon("Compressed clip (av1).webm"),
		status("watching"),
	)
}

func TestEncodeOneKeepsTheOriginalWhenDeleteIsOff(t *testing.T) {
	cfg, in, out := mediaConfig(t, false, true)
	tray, rec := recordingTray(t, cfg)
	src := filepath.Join(in, "clip.mp4")
	writeFile(t, src, "raw")
	writeFile(t, filepath.Join(out, "clip (av1).webm"), "encoded")

	encodeOne(context.Background(), cfg, av1Encoder(), tray, src)

	if got := readFile(t, src); got != "raw" {
		t.Errorf("original = %q, want it kept", got)
	}
	expectTray(t, rec,
		status("encoding clip.mp4"),
		balloon("Compressed clip (av1).webm"),
		status("watching"),
	)
}

func TestEncodeOneWithNotificationsOffSendsNoBalloon(t *testing.T) {
	cfg, in, out := mediaConfig(t, false, false)
	tray, rec := recordingTray(t, cfg)
	src := filepath.Join(in, "clip.mp4")
	writeFile(t, src, "raw")
	writeFile(t, filepath.Join(out, "clip (av1).webm"), "encoded")

	encodeOne(context.Background(), cfg, av1Encoder(), tray, src)

	expectTray(t, rec, status("encoding clip.mp4"), status("watching"))
}

func TestEncodeOneFailureKeepsTheOriginalAndReports(t *testing.T) {
	cfg, in, _ := mediaConfig(t, true, true)
	tray, rec := recordingTray(t, cfg)
	src := filepath.Join(in, "clip.mp4")
	writeFile(t, src, "raw")
	buf := captureLog(t)

	encodeOne(context.Background(), cfg, &encoder.Encoder{}, tray, src)

	if got := readFile(t, src); got != "raw" {
		t.Errorf("original = %q, want it kept after a failed encode", got)
	}
	if !strings.Contains(buf.String(), "encode clip.mp4:") {
		t.Errorf("log = %q, want an encode error", buf.String())
	}
	expectTray(t, rec,
		status("encoding clip.mp4"),
		status("encode failed"),
		balloon("Failed to encode clip.mp4"),
	)
}

func TestCopyImageDeletesTheOriginalAfterSuccess(t *testing.T) {
	cfg, in, out := mediaConfig(t, true, true)
	tray, rec := recordingTray(t, cfg)
	src := filepath.Join(in, "shot.png")
	writeFile(t, src, "pixels")

	copyImage(cfg, tray, src)

	if got := readFile(t, filepath.Join(out, "shot.png")); got != "pixels" {
		t.Errorf("copy = %q, want pixels", got)
	}
	if exists(src) {
		t.Error("original should be deleted after a successful copy")
	}
	expectTray(t, rec,
		status("copying shot.png"),
		balloon("Copied shot.png"),
		status("watching"),
	)
}

func TestCopyImageKeepsTheOriginalWhenDeleteIsOff(t *testing.T) {
	cfg, in, out := mediaConfig(t, false, false)
	tray, rec := recordingTray(t, cfg)
	src := filepath.Join(in, "shot.png")
	writeFile(t, src, "pixels")

	copyImage(cfg, tray, src)

	if got := readFile(t, filepath.Join(out, "shot.png")); got != "pixels" {
		t.Errorf("copy = %q, want pixels", got)
	}
	if got := readFile(t, src); got != "pixels" {
		t.Errorf("original = %q, want it kept", got)
	}
	expectTray(t, rec, status("copying shot.png"), status("watching"))
}

func TestCopyImageFailureKeepsTheOriginalAndReports(t *testing.T) {
	cfg, in, out := mediaConfig(t, true, true)
	tray, rec := recordingTray(t, cfg)
	src := filepath.Join(in, "shot.png")
	writeFile(t, src, "pixels")
	writeFile(t, out, "not a directory")
	buf := captureLog(t)

	copyImage(cfg, tray, src)

	if got := readFile(t, src); got != "pixels" {
		t.Errorf("original = %q, want it kept after a failed copy", got)
	}
	if !strings.Contains(buf.String(), "copy shot.png:") {
		t.Errorf("log = %q, want a copy error", buf.String())
	}
	expectTray(t, rec,
		status("copying shot.png"),
		status("copy failed"),
		balloon("Failed to copy shot.png"),
	)
}

func TestStartBackgroundReportsMissingFFmpeg(t *testing.T) {
	cfg, _, _ := mediaConfig(t, false, true)
	tray, rec := recordingTray(t, cfg)
	missing := filepath.Join(t.TempDir(), "ffmpeg.exe")
	enc := &encoder.Encoder{FFmpegPath: missing}
	w := watcher.New(cfg, func(context.Context, string) {}, nil)
	buf := captureLog(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	done := make(chan struct{})
	go func() {
		startBackground(ctx, cfg, enc, tray, w)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("startBackground kept running without ffmpeg")
	}

	if !strings.Contains(buf.String(), "ffmpeg not found at "+missing) {
		t.Errorf("log = %q, want the missing ffmpeg path", buf.String())
	}
	expectTray(t, rec,
		status("ffmpeg missing"),
		balloon("ffmpeg.exe is missing from the ClipCompress folder — reinstall the app"),
	)
}

func TestApplyCodecReportsNoGPUEncoder(t *testing.T) {
	cfg, _, _ := mediaConfig(t, false, true)
	tray, rec := recordingTray(t, cfg)
	enc := &encoder.Encoder{FFmpegPath: fakeFFmpeg(t, "libx264")}
	buf := captureLog(t)

	applyCodec(cfg, enc, tray)

	if enc.Profile != (encoder.Profile{}) {
		t.Errorf("Profile = %+v, want it left empty", enc.Profile)
	}
	if !strings.Contains(buf.String(), "codec resolve:") {
		t.Errorf("log = %q, want a codec resolve error", buf.String())
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(rec.snapshot()) < 2 && time.Now().Before(deadline) {
		onUI(t, func() { iup.LoopStep() })
		time.Sleep(time.Millisecond)
	}
	got := rec.snapshot()
	if len(got) != 2 || got[0] != status("no GPU encoder") || got[1].code != notifyCode ||
		!strings.Contains(got[1].text, "no working NVENC encoder found") {
		t.Errorf("tray messages = %+v, want no GPU encoder status and a NVENC balloon", got)
	}
}

func TestApplyCodecSetsTheProfileAndStatus(t *testing.T) {
	cfg, _, _ := mediaConfig(t, false, true)
	tray, rec := recordingTray(t, cfg)
	enc := &encoder.Encoder{FFmpegPath: fakeFFmpeg(t, "av1_nvenc hevc_nvenc h264_nvenc")}

	applyCodec(cfg, enc, tray)

	if enc.Profile != encoder.Profiles[config.CodecAV1] {
		t.Errorf("Profile = %+v, want av1", enc.Profile)
	}
	expectTray(t, rec, status("watching (av1)"))
}

func TestApplyCodecWhilePausedSaysPaused(t *testing.T) {
	cfg, _, _ := mediaConfig(t, false, true)
	cfg.SetPaused(true)
	tray, rec := recordingTray(t, cfg)
	enc := &encoder.Encoder{FFmpegPath: fakeFFmpeg(t, "hevc_nvenc")}

	applyCodec(cfg, enc, tray)

	if enc.Profile != encoder.Profiles[config.CodecHEVC] {
		t.Errorf("Profile = %+v, want hevc", enc.Profile)
	}
	expectTray(t, rec, status("paused"))
}
