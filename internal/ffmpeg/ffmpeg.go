package ffmpeg

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	ffmpegVersion = "7.1"
	ffmpegBaseURL = "https://github.com/GyanD/codexffmpeg/releases/download"
)

func zipURL() string {
	return fmt.Sprintf("%s/%s/ffmpeg-%s-full_build.zip", ffmpegBaseURL, ffmpegVersion, ffmpegVersion)
}

type Manager struct {
	dir string
}

func NewManager(dataName string) (*Manager, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return &Manager{dir: filepath.Join(base, dataName, "ffmpeg")}, nil
}

func exeName(stem string) string {
	if runtime.GOOS == "windows" {
		return stem + ".exe"
	}
	return stem
}

func (m *Manager) FFmpegPath() string {
	if runtime.GOOS != "windows" {
		return "ffmpeg"
	}
	return filepath.Join(m.dir, exeName("ffmpeg"))
}

func (m *Manager) FFprobePath() string {
	if runtime.GOOS != "windows" {
		return "ffprobe"
	}
	return filepath.Join(m.dir, exeName("ffprobe"))
}

func (m *Manager) Installed() bool {
	if runtime.GOOS != "windows" {
		_, err1 := exec.LookPath("ffmpeg")
		_, err2 := exec.LookPath("ffprobe")
		return err1 == nil && err2 == nil
	}
	return fileExists(m.FFmpegPath()) && fileExists(m.FFprobePath())
}

type ProgressFunc func(done, total int64)

func (m *Manager) Ensure(progress ProgressFunc) error {
	if m.Installed() {
		return nil
	}
	if runtime.GOOS != "windows" {
		return fmt.Errorf("ffmpeg/ffprobe not found on PATH (install them for local dev)")
	}
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp("", "ffmpeg-*.zip")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := download(zipURL(), tmp, progress); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("downloading ffmpeg: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := extractBinaries(tmpPath, m.dir); err != nil {
		return fmt.Errorf("extracting ffmpeg: %w", err)
	}
	if !m.Installed() {
		return fmt.Errorf("ffmpeg archive did not contain the expected binaries")
	}
	return nil
}

func download(url string, dst io.Writer, progress ProgressFunc) error {
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s for %s", resp.Status, url)
	}

	total := resp.ContentLength
	var done int64
	buf := make([]byte, 64*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			done += int64(n)
			if progress != nil {
				progress(done, total)
			}
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			return rerr
		}
	}
}

func extractBinaries(zipPath, dir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func() { _ = zr.Close() }()

	wanted := map[string]bool{"ffmpeg.exe": true, "ffprobe.exe": true}
	for _, f := range zr.File {
		base := filepath.Base(filepath.FromSlash(f.Name))
		if !wanted[strings.ToLower(base)] {
			continue
		}
		if err := writeZipFile(f, filepath.Join(dir, base)); err != nil {
			return err
		}
	}
	return nil
}

func writeZipFile(f *zip.File, dst string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, rc)
	return err
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
