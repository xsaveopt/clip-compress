package encoder

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/xsaveopt/clip-compress/internal/config"
)

type Profile struct {
	Key       string
	VideoEnc  string
	Container string
	AudioEnc  string
}

var Profiles = map[string]Profile{
	config.CodecAV1:  {config.CodecAV1, "av1_nvenc", ".webm", "libopus"},
	config.CodecHEVC: {config.CodecHEVC, "hevc_nvenc", ".mp4", "aac"},
	config.CodecH264: {config.CodecH264, "h264_nvenc", ".mp4", "aac"},
}

var autoOrder = []string{config.CodecAV1, config.CodecHEVC, config.CodecH264}

func ResolveProfile(ffmpegPath, want string) (Profile, error) {
	if want == config.CodecAuto || want == "" {
		var lastErr error
		for _, k := range autoOrder {
			p := Profiles[k]
			if err := probeEncoder(ffmpegPath, p.VideoEnc); err == nil {
				return p, nil
			} else {
				lastErr = err
			}
		}
		return Profile{}, fmt.Errorf("no working NVENC encoder found — is this an NVIDIA GPU with a current driver? %w", lastErr)
	}

	p, ok := Profiles[want]
	if !ok {
		return Profile{}, fmt.Errorf("unknown codec %q", want)
	}
	if err := probeEncoder(ffmpegPath, p.VideoEnc); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func probeEncoder(ffmpegPath, videoEnc string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	list := exec.CommandContext(ctx, ffmpegPath, "-hide_banner", "-encoders")
	hideWindow(list)
	encoders, err := list.Output()
	if err != nil {
		return fmt.Errorf("could not query ffmpeg encoders: %w", err)
	}
	if !strings.Contains(string(encoders), videoEnc) {
		return fmt.Errorf("this ffmpeg build has no %s encoder", videoEnc)
	}

	test := exec.CommandContext(ctx, ffmpegPath,
		"-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=duration=0.2:size=1280x720:rate=30",
		"-pix_fmt", "yuv420p", "-c:v", videoEnc, "-f", "null", "-",
	)
	hideWindow(test)
	if out, err := test.CombinedOutput(); err != nil {
		return fmt.Errorf("%s test encode failed (unsupported GPU/driver): %w\n%s", videoEnc, err, strings.TrimSpace(string(out)))
	}
	return nil
}
