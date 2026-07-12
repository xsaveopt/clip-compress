package encoder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xsaveopt/clip-compress/internal/config"
)

type Encoder struct {
	FFmpegPath string
	Profile    Profile
}

type Options struct {
	OutputDir     string
	VideoBitrateK int
}

func OptionsFromConfig(c *config.Config) Options {
	return Options{
		OutputDir:     c.OutputDir(),
		VideoBitrateK: c.VideoBitrateK(),
	}
}

const minVideoBitrateK = 200

func (e *Encoder) Encode(ctx context.Context, input string, opts Options) (string, error) {
	if e.Profile.VideoEnc == "" {
		return "", fmt.Errorf("no NVENC encoder resolved")
	}
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return "", err
	}

	stem := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	output := filepath.Join(opts.OutputDir, stem+" ("+e.Profile.Key+")"+e.Profile.Container)
	if fileExists(output) {
		return output, nil
	}

	args := e.buildArgs(input, output, opts)
	cmd := exec.CommandContext(ctx, e.FFmpegPath, args...)
	hideWindow(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w\n%s", err, tail(stderr.String(), 600))
	}
	return output, nil
}

func (e *Encoder) buildArgs(input, output string, opts Options) []string {
	p := e.Profile
	k := opts.VideoBitrateK
	if k < minVideoBitrateK {
		k = minVideoBitrateK
	}

	args := []string{
		"-hide_banner", "-y", "-i", input,
		"-map_metadata", "0",
		"-c:v", p.VideoEnc, "-preset", "p7", "-tune", "hq", "-pix_fmt", "yuv420p",
		"-rc", "cbr", "-b:v", fmt.Sprintf("%dk", k), "-multipass", "fullres",
	}

	args = append(args, "-c:a", p.AudioEnc, "-b:a", fmt.Sprintf("%dk", config.AudioBitrateK), "-ac", "2")
	if p.Container == ".mp4" {
		args = append(args, "-movflags", "+faststart")
	}
	args = append(args, output)
	return args
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
