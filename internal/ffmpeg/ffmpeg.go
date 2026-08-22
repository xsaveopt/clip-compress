package ffmpeg

import (
	"os"
	"path/filepath"
)

const exeName = "ffmpeg.exe"

func Path() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), exeName), nil
}

func Installed(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
