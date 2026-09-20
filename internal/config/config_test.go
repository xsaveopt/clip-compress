package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const testAppName = "ClipCompressTest"

func isolateHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("AppData", filepath.Join(dir, "AppData"))
	t.Setenv("USERPROFILE", dir)
	return dir
}

func configPath(t *testing.T) string {
	t.Helper()
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	return filepath.Join(base, testAppName, "config.json")
}

func writeConfig(t *testing.T, raw string) {
	t.Helper()
	path := configPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	home := isolateHome(t)

	c, err := Load(testAppName)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := c.SourceDir(), filepath.Join(home, "Videos"); got != want {
		t.Errorf("SourceDir = %q, want %q", got, want)
	}
	if got, want := c.OutputDir(), filepath.Join(home, "Videos", "ClipCompress"); got != want {
		t.Errorf("OutputDir = %q, want %q", got, want)
	}
	if got := c.VideoBitrateK(); got != 1900 {
		t.Errorf("VideoBitrateK = %d, want 1900", got)
	}
	if c.DeleteOriginal() || c.Notify() || c.Paused() {
		t.Error("boolean defaults should all be false")
	}
	if c.StartAtLoginSet() {
		t.Error("StartAtLogin should be unset by default")
	}
}

func TestLoadReadsExistingFile(t *testing.T) {
	isolateHome(t)
	writeConfig(t, `{
		"sourceDir": "S",
		"outputDir": "O",
		"videoBitrateK": 4200,
		"deleteOriginal": true,
		"notify": true,
		"startAtLogin": false,
		"paused": true
	}`)

	c, err := Load(testAppName)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.SourceDir() != "S" || c.OutputDir() != "O" {
		t.Errorf("dirs = %q/%q, want S/O", c.SourceDir(), c.OutputDir())
	}
	if c.VideoBitrateK() != 4200 {
		t.Errorf("VideoBitrateK = %d, want 4200", c.VideoBitrateK())
	}
	if !c.DeleteOriginal() || !c.Notify() || !c.Paused() {
		t.Error("deleteOriginal, notify and paused should all be true")
	}
	if !c.StartAtLoginSet() {
		t.Error("StartAtLoginSet = false, want true for an explicit false")
	}
	if c.StartAtLogin() {
		t.Error("StartAtLogin = true, want false")
	}
}

func TestLoadPartialFileKeepsDefaults(t *testing.T) {
	home := isolateHome(t)
	writeConfig(t, `{"videoBitrateK": 800}`)

	c, err := Load(testAppName)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.VideoBitrateK() != 800 {
		t.Errorf("VideoBitrateK = %d, want 800", c.VideoBitrateK())
	}
	if got, want := c.SourceDir(), filepath.Join(home, "Videos"); got != want {
		t.Errorf("SourceDir = %q, want default %q", got, want)
	}
}

func TestLoadInvalidJSONReturnsErrorAndDefaults(t *testing.T) {
	home := isolateHome(t)
	writeConfig(t, `{not json`)

	c, err := Load(testAppName)
	if err == nil {
		t.Fatal("Load should report a parse error")
	}
	if c == nil {
		t.Fatal("Load should still return a usable config")
	}
	if got, want := c.SourceDir(), filepath.Join(home, "Videos"); got != want {
		t.Errorf("SourceDir = %q, want default %q", got, want)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	isolateHome(t)

	c, err := Load(testAppName)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	c.SetSourceDir("src")
	c.SetOutputDir("out")
	c.SetVideoBitrateK(2500)
	c.SetDeleteOriginal(true)
	c.SetNotify(true)
	c.SetPaused(true)
	c.SetStartAtLogin(true)

	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	again, err := Load(testAppName)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if again.SourceDir() != "src" || again.OutputDir() != "out" {
		t.Errorf("dirs = %q/%q, want src/out", again.SourceDir(), again.OutputDir())
	}
	if again.VideoBitrateK() != 2500 {
		t.Errorf("VideoBitrateK = %d, want 2500", again.VideoBitrateK())
	}
	if !again.DeleteOriginal() || !again.Notify() || !again.Paused() || !again.StartAtLogin() {
		t.Error("saved booleans did not survive a reload")
	}
}

func TestSaveCreatesDirectoryAndLeavesNoTempFile(t *testing.T) {
	isolateHome(t)

	c, err := Load(testAppName)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	path := configPath(t)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file missing after Save: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Error("Save left a .tmp file behind")
	}
}

func TestSaveOmitsUnsetStartAtLogin(t *testing.T) {
	isolateHome(t)

	c, err := Load(testAppName)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	raw, err := os.ReadFile(configPath(t))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := m["startAtLogin"]; ok {
		t.Error("startAtLogin should be omitted while unset")
	}
}

func TestSettersAreIndependent(t *testing.T) {
	c := &Config{}

	c.SetSourceDir("a")
	c.SetOutputDir("b")
	c.SetVideoBitrateK(1)
	c.SetDeleteOriginal(true)
	c.SetNotify(true)
	c.SetPaused(true)
	c.SetStartAtLogin(true)

	if c.SourceDir() != "a" {
		t.Errorf("SourceDir = %q, want a", c.SourceDir())
	}
	if c.OutputDir() != "b" {
		t.Errorf("OutputDir = %q, want b", c.OutputDir())
	}
	if c.VideoBitrateK() != 1 {
		t.Errorf("VideoBitrateK = %d, want 1", c.VideoBitrateK())
	}
	if !c.DeleteOriginal() || !c.Notify() || !c.Paused() || !c.StartAtLogin() {
		t.Error("boolean setters did not stick")
	}

	c.SetStartAtLogin(false)
	if c.StartAtLogin() {
		t.Error("SetStartAtLogin(false) did not take")
	}
	if !c.StartAtLoginSet() {
		t.Error("StartAtLoginSet should stay true after an explicit false")
	}
}

func TestCodecIsAlwaysAuto(t *testing.T) {
	c := &Config{}
	if got := c.Codec(); got != CodecAuto {
		t.Errorf("Codec = %q, want %q", got, CodecAuto)
	}
}

func TestIsWatched(t *testing.T) {
	c := &Config{}
	cases := []struct {
		path string
		want bool
	}{
		{"clip.mp4", true},
		{"clip.mkv", true},
		{"clip.mov", true},
		{"clip.avi", true},
		{"clip.webm", true},
		{"shot.png", true},
		{"shot.jpg", true},
		{"shot.jpeg", true},
		{"shot.bmp", true},
		{"CLIP.MP4", true},
		{"Shot.PNG", true},
		{filepath.Join("a", "b", "clip.mp4"), true},
		{"clip.txt", false},
		{"clip.gif", false},
		{"clip", false},
		{"", false},
		{"archive.mp4.zip", false},
	}
	for _, tc := range cases {
		if got := c.IsWatched(tc.path); got != tc.want {
			t.Errorf("IsWatched(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestIsImage(t *testing.T) {
	c := &Config{}
	cases := []struct {
		path string
		want bool
	}{
		{"shot.png", true},
		{"shot.JPG", true},
		{"shot.jpeg", true},
		{"shot.bmp", true},
		{"clip.mp4", false},
		{"clip.webm", false},
		{"shot.gif", false},
		{"shot", false},
	}
	for _, tc := range cases {
		if got := c.IsImage(tc.path); got != tc.want {
			t.Errorf("IsImage(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
