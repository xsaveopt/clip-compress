package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"image/png"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gen2brain/iup-go/iup"

	"github.com/xsaveopt/clip-compress/internal/config"
	"github.com/xsaveopt/clip-compress/internal/encoder"
	"github.com/xsaveopt/clip-compress/internal/ffmpeg"
	"github.com/xsaveopt/clip-compress/internal/singleton"
	"github.com/xsaveopt/clip-compress/internal/startup"
	"github.com/xsaveopt/clip-compress/internal/ui"
	"github.com/xsaveopt/clip-compress/internal/watcher"
)

var version = "dev"

//go:embed assets/icon.png
var iconPNG []byte

type identity struct {
	dataName string
	runName  string
	mutex    string
}

func buildIdentity() identity {
	if strings.HasPrefix(version, "v") {
		return identity{
			dataName: "ClipCompress",
			runName:  "ClipCompress",
			mutex:    `Global\ClipCompress`,
		}
	}
	return identity{
		dataName: "ClipCompress (Dev)",
		runName:  "ClipCompress (Dev)",
		mutex:    `Global\ClipCompress-Dev`,
	}
}

func main() {
	runtime.LockOSThread()

	log.SetPrefix("clip-compress: ")
	log.Printf("starting ClipCompress %s", version)

	id := buildIdentity()

	first, err := singleton.Acquire(id.mutex)
	if err != nil {
		log.Printf("singleton check: %v", err)
	}
	if !first {
		log.Print("another instance is already running; exiting")
		return
	}

	icon, err := png.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		log.Fatalf("decode icon: %v", err)
	}

	cfg, err := config.Load(id.dataName)
	if err != nil {
		log.Printf("load config: %v", err)
	}
	if !cfg.StartAtLoginSet() {
		cfg.SetStartAtLogin(startup.Enabled(id.runName))
	}
	if err := startup.Sync(id.runName, cfg.StartAtLogin()); err != nil {
		log.Printf("start-at-login sync: %v", err)
	}

	mgr, err := ffmpeg.NewManager(id.dataName)
	if err != nil {
		log.Fatalf("ffmpeg manager: %v", err)
	}

	iup.Open()
	defer iup.Close()
	iup.SetGlobal("LOCKLOOP", "YES")

	enc := &encoder.Encoder{FFmpegPath: mgr.FFmpegPath()}

	ctx, cancel := context.WithCancel(context.Background())

	var tray *ui.Tray
	w := watcher.New(cfg, func(ctx context.Context, path string) {
		if cfg.IsImage(path) {
			copyImage(cfg, tray, path)
			return
		}
		encodeOne(ctx, cfg, enc, tray, path)
	}, func(msg string) { log.Print(msg) })

	settings := ui.NewSettings(cfg, func() {
		saveConfig(cfg)
		if err := startup.Sync(id.runName, cfg.StartAtLogin()); err != nil {
			log.Printf("start-at-login sync: %v", err)
		}
		w.Rewatch()
		go applyCodec(cfg, enc, tray)
	})

	tray = ui.NewTray(settings.Dialog(), cfg, icon, ui.TrayActions{
		ShowSettings: settings.Show,
		OpenOutput:   func() { openFolder(cfg.OutputDir()) },
		TogglePause: func() {
			cfg.SetPaused(!cfg.Paused())
			saveConfig(cfg)
			go applyCodec(cfg, enc, tray)
		},
		Quit: func() { cancel(); iup.ExitLoop() },
	})

	go startBackground(ctx, cfg, mgr, enc, tray, w)

	iup.MainLoop()
	cancel()
}

func saveConfig(cfg *config.Config) {
	if err := cfg.Save(); err != nil {
		log.Printf("save config: %v", err)
	}
}

func startBackground(ctx context.Context, cfg *config.Config, mgr *ffmpeg.Manager, enc *encoder.Encoder, tray *ui.Tray, w *watcher.Watcher) {
	if !mgr.Installed() {
		tray.SetStatus("downloading ffmpeg…")
		err := mgr.Ensure(func(done, total int64) {
			if total > 0 {
				tray.SetStatus(fmt.Sprintf("downloading ffmpeg… %d%%", done*100/total))
			}
		})
		if err != nil {
			tray.SetStatus("ffmpeg download failed")
			notify(cfg, tray, "Could not download ffmpeg: "+err.Error())
			log.Printf("ffmpeg ensure: %v", err)
			return
		}
	}

	applyCodec(cfg, enc, tray)

	if err := w.Start(ctx); err != nil {
		log.Printf("watcher: %v", err)
	}
}

func applyCodec(cfg *config.Config, enc *encoder.Encoder, tray *ui.Tray) {
	profile, err := encoder.ResolveProfile(enc.FFmpegPath, cfg.Codec())
	if err != nil {
		tray.SetStatus("no GPU encoder")
		notify(cfg, tray, err.Error())
		log.Printf("codec resolve: %v", err)
		return
	}
	enc.Profile = profile
	if cfg.Paused() {
		tray.SetStatus("paused")
	} else {
		tray.SetStatus("watching (" + profile.Key + ")")
	}
}

func encodeOne(ctx context.Context, cfg *config.Config, enc *encoder.Encoder, tray *ui.Tray, path string) {
	name := filepath.Base(path)
	tray.SetStatus("encoding " + name)
	out, err := enc.Encode(ctx, path, encoder.OptionsFromConfig(cfg))
	if err != nil {
		tray.SetStatus("encode failed")
		notify(cfg, tray, "Failed to encode "+name)
		log.Printf("encode %s: %v", name, err)
		return
	}
	if cfg.DeleteOriginal() {
		if rerr := os.Remove(path); rerr != nil {
			log.Printf("delete original %s: %v", name, rerr)
		}
	}
	notify(cfg, tray, "Compressed "+filepath.Base(out))
	tray.SetStatus("watching")
}

func copyImage(cfg *config.Config, tray *ui.Tray, path string) {
	name := filepath.Base(path)
	dst := filepath.Join(cfg.OutputDir(), name)
	if _, err := os.Stat(dst); err == nil {
		return
	}
	tray.SetStatus("copying " + name)
	if err := copyFile(path, dst); err != nil {
		tray.SetStatus("copy failed")
		notify(cfg, tray, "Failed to copy "+name)
		log.Printf("copy %s: %v", name, err)
		return
	}
	if cfg.DeleteOriginal() {
		if rerr := os.Remove(path); rerr != nil {
			log.Printf("delete original %s: %v", name, rerr)
		}
	}
	notify(cfg, tray, "Copied "+name)
	tray.SetStatus("watching")
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func notify(cfg *config.Config, tray *ui.Tray, body string) {
	if cfg.Notify() {
		tray.Notify(body)
	}
}

func openFolder(path string) {
	if path == "" {
		return
	}
	_ = os.MkdirAll(path, 0o755)
	_ = exec.Command("explorer", path).Start()
}
