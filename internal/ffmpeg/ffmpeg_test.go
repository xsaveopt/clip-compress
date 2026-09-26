package ffmpeg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathSitsNextToTheExecutable(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable: %v", err)
	}

	got, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if want := filepath.Join(filepath.Dir(exe), "ffmpeg.exe"); got != want {
		t.Errorf("Path = %q, want %q", got, want)
	}
}

func TestInstalledTrueForAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ffmpeg.exe")
	if err := os.WriteFile(path, []byte("bin"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if !Installed(path) {
		t.Error("Installed = false, want true for an existing file")
	}
}

func TestInstalledFalseForAMissingFile(t *testing.T) {
	if Installed(filepath.Join(t.TempDir(), "ffmpeg.exe")) {
		t.Error("Installed = true, want false for a missing file")
	}
}

func TestInstalledFalseForADirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ffmpeg.exe")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	if Installed(path) {
		t.Error("Installed = true, want false for a directory")
	}
}

func TestInstalledFalseForAnEmptyPath(t *testing.T) {
	if Installed("") {
		t.Error("Installed = true, want false for an empty path")
	}
}
