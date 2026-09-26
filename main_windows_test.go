package main

import (
	"bytes"
	"encoding/json"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xsaveopt/clip-compress/internal/config"
)

const testAppName = "ClipCompressMainTest"

func isolateHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("AppData", filepath.Join(dir, "AppData"))
	t.Setenv("USERPROFILE", dir)
}

func loadConfig(t *testing.T) *config.Config {
	t.Helper()
	isolateHome(t)
	cfg, err := config.Load(testAppName)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

func configDir(t *testing.T) string {
	t.Helper()
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	return filepath.Join(base, testAppName)
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	origOut, origFlags, origPrefix := log.Writer(), log.Flags(), log.Prefix()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(origOut)
		log.SetFlags(origFlags)
		log.SetPrefix(origPrefix)
	})
	return &buf
}

func setVersion(t *testing.T, v string) {
	t.Helper()
	orig := version
	version = v
	t.Cleanup(func() { version = orig })
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return string(raw)
}

func TestBuildIdentityRelease(t *testing.T) {
	setVersion(t, "v1.2.3")

	got := buildIdentity()
	want := identity{dataName: "ClipCompress", runName: "ClipCompress", mutex: `Global\ClipCompress`}
	if got != want {
		t.Errorf("buildIdentity = %+v, want %+v", got, want)
	}
}

func TestBuildIdentityDev(t *testing.T) {
	for _, v := range []string{"dev", "dev-abc1234", "1.2.3", ""} {
		t.Run(v, func(t *testing.T) {
			setVersion(t, v)

			got := buildIdentity()
			want := identity{dataName: "ClipCompress (Dev)", runName: "ClipCompress (Dev)", mutex: `Global\ClipCompress-Dev`}
			if got != want {
				t.Errorf("buildIdentity = %+v, want %+v", got, want)
			}
		})
	}
}

func TestBuildIdentityDevAndReleaseDoNotCollide(t *testing.T) {
	setVersion(t, "v1.0.0")
	release := buildIdentity()
	version = "dev"
	dev := buildIdentity()

	if release.dataName == dev.dataName || release.runName == dev.runName || release.mutex == dev.mutex {
		t.Errorf("release %+v and dev %+v share a name", release, dev)
	}
}

func TestSaveConfigWritesFile(t *testing.T) {
	cfg := loadConfig(t)
	cfg.SetVideoBitrateK(3100)
	cfg.SetPaused(true)
	buf := captureLog(t)

	saveConfig(cfg)

	var got map[string]any
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(configDir(t), "config.json"))), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got["videoBitrateK"] != float64(3100) {
		t.Errorf("videoBitrateK = %v, want 3100", got["videoBitrateK"])
	}
	if got["paused"] != true {
		t.Errorf("paused = %v, want true", got["paused"])
	}
	if buf.Len() != 0 {
		t.Errorf("unexpected log output %q", buf.String())
	}
}

func TestSaveConfigLogsFailure(t *testing.T) {
	cfg := loadConfig(t)
	writeFile(t, configDir(t), "not a directory")
	buf := captureLog(t)

	saveConfig(cfg)

	if !strings.Contains(buf.String(), "save config:") {
		t.Errorf("log = %q, want a save config error", buf.String())
	}
}

func TestCopyFileCopiesContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "shot.png")
	dst := filepath.Join(dir, "out", "shot.png")
	writeFile(t, src, "pixels")

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	if got := readFile(t, dst); got != "pixels" {
		t.Errorf("dst = %q, want pixels", got)
	}
	if got := readFile(t, src); got != "pixels" {
		t.Errorf("src = %q, want it left intact", got)
	}
}

func TestCopyFileCreatesNestedDirs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "shot.png")
	dst := filepath.Join(dir, "a", "b", "c", "shot.png")
	writeFile(t, src, "x")

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	if got := readFile(t, dst); got != "x" {
		t.Errorf("dst = %q, want x", got)
	}
}

func TestCopyFileOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "shot.png")
	dst := filepath.Join(dir, "out", "shot.png")
	writeFile(t, src, "new")
	writeFile(t, dst, "old content that is longer")

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	if got := readFile(t, dst); got != "new" {
		t.Errorf("dst = %q, want new", got)
	}
}

func TestCopyFileMissingSource(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out", "shot.png")

	if err := copyFile(filepath.Join(dir, "missing.png"), dst); err == nil {
		t.Fatal("copyFile should fail for a missing source")
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Errorf("dst stat err = %v, want not exist", err)
	}
}

func TestCopyFileDestinationParentIsAFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "shot.png")
	blocker := filepath.Join(dir, "out")
	writeFile(t, src, "x")
	writeFile(t, blocker, "file")

	if err := copyFile(src, filepath.Join(blocker, "shot.png")); err == nil {
		t.Fatal("copyFile should fail when the destination parent is a file")
	}
}

func TestCopyImageSkipsExistingDestination(t *testing.T) {
	cfg := loadConfig(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	cfg.SetOutputDir(out)
	cfg.SetDeleteOriginal(true)
	cfg.SetNotify(true)
	src := filepath.Join(dir, "in", "shot.png")
	dst := filepath.Join(out, "shot.png")
	writeFile(t, src, "new")
	writeFile(t, dst, "old")

	copyImage(cfg, nil, src)

	if got := readFile(t, dst); got != "old" {
		t.Errorf("dst = %q, want it left as old", got)
	}
	if got := readFile(t, src); got != "new" {
		t.Errorf("src = %q, want it kept", got)
	}
}

func TestNotifyDisabledSkipsTray(t *testing.T) {
	cfg := loadConfig(t)
	cfg.SetNotify(false)

	notify(cfg, nil, "hello")
}

func TestOpenFolderEmptyPathIsANoOp(t *testing.T) {
	isolateHome(t)
	wd := t.TempDir()
	t.Chdir(wd)

	openFolder("")

	entries, err := os.ReadDir(wd)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("working dir has %d entries, want none", len(entries))
	}
}

func TestEmbeddedIconDecodes(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	if b := img.Bounds(); b.Dx() == 0 || b.Dy() == 0 {
		t.Errorf("icon bounds = %v, want a non-empty image", b)
	}
}

func TestCopyFileDestinationIsADirectory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "shot.png")
	dst := filepath.Join(dir, "out", "shot.png")
	writeFile(t, src, "x")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := copyFile(src, dst); err == nil {
		t.Fatal("copyFile should fail when the destination is a directory")
	}
	if got := readFile(t, src); got != "x" {
		t.Errorf("src = %q, want it left intact", got)
	}
}
