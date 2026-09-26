package ui

import (
	"image"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gen2brain/iup-go/iup"

	"github.com/xsaveopt/clip-compress/internal/config"
)

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

func pumpUntil(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var ok bool
		onUI(t, func() {
			iup.LoopStep()
			ok = cond()
		})
		if ok {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal(msg)
}

func newTestTray(t *testing.T, cfg *config.Config) *Tray {
	t.Helper()
	var tr *Tray
	onUI(t, func() {
		dlg := iup.Dialog(iup.Vbox())
		tr = NewTray(dlg, cfg, image.NewRGBA(image.Rect(0, 0, 16, 16)), TrayActions{})
	})
	t.Cleanup(func() {
		onUI(t, func() {
			tr.dlg.SetAttribute("TRAY", "NO")
			iup.Destroy(tr.dlg)
		})
	})
	return tr
}

func newTestSettings(t *testing.T, cfg *config.Config) *Settings {
	t.Helper()
	var s *Settings
	onUI(t, func() {
		s = NewSettings(cfg, nil)
		iup.Map(s.Dialog())
	})
	t.Cleanup(func() {
		onUI(t, func() { iup.Destroy(s.Dialog()) })
	})
	return s
}

func TestTraySetStatusUpdatesTheStatusAndTip(t *testing.T) {
	tr := newTestTray(t, &config.Config{})

	var start string
	onUI(t, func() { start = tr.status })
	if start != "starting…" {
		t.Errorf("initial status = %q, want starting…", start)
	}

	tr.SetStatus("encoding clip.mp4")
	pumpUntil(t, func() bool { return tr.status == "encoding clip.mp4" }, "status never reached encoding clip.mp4")

	onUI(t, func() {
		if got, want := tr.tip(), "ClipCompress — encoding clip.mp4"; got != want {
			t.Errorf("tip() = %q, want %q", got, want)
		}
	})
}

func TestTraySetStatusAppliesMessagesInOrder(t *testing.T) {
	tr := newTestTray(t, &config.Config{})

	tr.SetStatus("encoding a.mp4")
	tr.SetStatus("encoding b.mp4")
	tr.SetStatus("watching")
	pumpUntil(t, func() bool { return tr.status == "watching" }, "status never reached watching")

	for range 20 {
		onUI(t, func() { iup.LoopStep() })
	}
	onUI(t, func() {
		if tr.status != "watching" {
			t.Errorf("status = %q after draining, want watching", tr.status)
		}
	})
}

func TestTrayNotifyLeavesTheStatusAlone(t *testing.T) {
	tr := newTestTray(t, &config.Config{})

	tr.SetStatus("watching (av1)")
	tr.Notify("Compressed clip (av1).webm")
	tr.SetStatus("done")
	pumpUntil(t, func() bool { return tr.status == "done" }, "status never reached done")

	onUI(t, func() {
		if got, want := tr.tip(), "ClipCompress — done"; got != want {
			t.Errorf("tip() = %q, want %q", got, want)
		}
	})
}

func TestSettingsReloadShowsTheConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.SetSourceDir("Videos")
	cfg.SetOutputDir("Compressed")
	cfg.SetVideoBitrateK(2500)
	cfg.SetDeleteOriginal(true)
	cfg.SetNotify(false)
	cfg.SetStartAtLogin(true)
	s := newTestSettings(t, cfg)

	onUI(t, func() {
		s.reload()

		if got := s.source.GetAttribute("VALUE"); got != "Videos" {
			t.Errorf("source = %q, want Videos", got)
		}
		if got := s.output.GetAttribute("VALUE"); got != "Compressed" {
			t.Errorf("output = %q, want Compressed", got)
		}
		if got := s.bitrate.GetAttribute("VALUE"); got != "2500" {
			t.Errorf("bitrate = %q, want 2500", got)
		}
		if got := s.deleteOriginal.GetInt("VALUE"); got != 1 {
			t.Errorf("deleteOriginal = %d, want 1", got)
		}
		if got := s.notify.GetInt("VALUE"); got != 0 {
			t.Errorf("notify = %d, want 0", got)
		}
		if got := s.startAtLogin.GetInt("VALUE"); got != 1 {
			t.Errorf("startAtLogin = %d, want 1", got)
		}
	})
}

func TestSettingsApplyWritesTheFormIntoTheConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.SetSourceDir("Videos")
	cfg.SetOutputDir("Compressed")
	cfg.SetVideoBitrateK(1900)
	cfg.SetNotify(true)
	s := newTestSettings(t, cfg)

	onUI(t, func() {
		s.reload()
		s.source.SetAttribute("VALUE", "Clips")
		s.output.SetAttribute("VALUE", "Small")
		s.bitrate.SetAttribute("VALUE", "3100")
		s.deleteOriginal.SetAttribute("VALUE", "ON")
		s.notify.SetAttribute("VALUE", "OFF")
		s.startAtLogin.SetAttribute("VALUE", "ON")
		s.apply()
	})

	if got := cfg.SourceDir(); got != "Clips" {
		t.Errorf("SourceDir = %q, want Clips", got)
	}
	if got := cfg.OutputDir(); got != "Small" {
		t.Errorf("OutputDir = %q, want Small", got)
	}
	if got := cfg.VideoBitrateK(); got != 3100 {
		t.Errorf("VideoBitrateK = %d, want 3100", got)
	}
	if !cfg.DeleteOriginal() {
		t.Error("DeleteOriginal = false, want true")
	}
	if cfg.Notify() {
		t.Error("Notify = true, want false")
	}
	if !cfg.StartAtLoginSet() || !cfg.StartAtLogin() {
		t.Error("StartAtLogin = unset or false, want true")
	}
}

func TestSettingsReloadThenApplyKeepsTheConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.SetSourceDir("Videos")
	cfg.SetOutputDir("Compressed")
	cfg.SetVideoBitrateK(2200)
	cfg.SetDeleteOriginal(true)
	cfg.SetNotify(true)
	cfg.SetStartAtLogin(false)
	s := newTestSettings(t, cfg)

	onUI(t, func() {
		s.reload()
		s.apply()
	})

	if cfg.SourceDir() != "Videos" || cfg.OutputDir() != "Compressed" {
		t.Errorf("dirs = %q/%q, want Videos/Compressed", cfg.SourceDir(), cfg.OutputDir())
	}
	if cfg.VideoBitrateK() != 2200 {
		t.Errorf("VideoBitrateK = %d, want 2200", cfg.VideoBitrateK())
	}
	if !cfg.DeleteOriginal() || !cfg.Notify() {
		t.Error("toggles that were on came back off")
	}
	if !cfg.StartAtLoginSet() || cfg.StartAtLogin() {
		t.Error("StartAtLogin should stay set to false")
	}
}

func TestSettingsReloadDiscardsUnsavedEdits(t *testing.T) {
	cfg := &config.Config{}
	cfg.SetSourceDir("Videos")
	s := newTestSettings(t, cfg)

	onUI(t, func() {
		s.reload()
		s.source.SetAttribute("VALUE", "Scratch")
		s.reload()
		if got := s.source.GetAttribute("VALUE"); got != "Videos" {
			t.Errorf("source = %q after reload, want Videos", got)
		}
	})
	if got := cfg.SourceDir(); got != "Videos" {
		t.Errorf("SourceDir = %q, want the unsaved edit ignored", got)
	}
}
