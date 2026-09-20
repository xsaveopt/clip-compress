package encoder

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xsaveopt/clip-compress/internal/config"
)

func fakeProbe(t *testing.T, working ...string) *[]string {
	t.Helper()
	tried := []string{}
	orig := probe
	probe = func(_, videoEnc string) error {
		tried = append(tried, videoEnc)
		for _, w := range working {
			if w == videoEnc {
				return nil
			}
		}
		return errors.New("no " + videoEnc)
	}
	t.Cleanup(func() { probe = orig })
	return &tried
}

func TestResolveProfileAutoPrefersAV1(t *testing.T) {
	tried := fakeProbe(t, "av1_nvenc", "hevc_nvenc", "h264_nvenc")

	p, err := ResolveProfile("ffmpeg", config.CodecAuto)
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}
	if p.Key != config.CodecAV1 {
		t.Errorf("key = %q, want %q", p.Key, config.CodecAV1)
	}
	if len(*tried) != 1 || (*tried)[0] != "av1_nvenc" {
		t.Errorf("probed %v, want a single av1_nvenc probe", *tried)
	}
}

func TestResolveProfileEmptyWantIsAuto(t *testing.T) {
	fakeProbe(t, "av1_nvenc")

	p, err := ResolveProfile("ffmpeg", "")
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}
	if p.Key != config.CodecAV1 {
		t.Errorf("key = %q, want %q", p.Key, config.CodecAV1)
	}
}

func TestResolveProfileAutoFallsBackToHEVC(t *testing.T) {
	tried := fakeProbe(t, "hevc_nvenc", "h264_nvenc")

	p, err := ResolveProfile("ffmpeg", config.CodecAuto)
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}
	if p.Key != config.CodecHEVC {
		t.Errorf("key = %q, want %q", p.Key, config.CodecHEVC)
	}
	if p.Container != ".mp4" || p.AudioEnc != "aac" {
		t.Errorf("profile = %+v, want mp4/aac", p)
	}
	want := []string{"av1_nvenc", "hevc_nvenc"}
	if strings.Join(*tried, ",") != strings.Join(want, ",") {
		t.Errorf("probed %v, want %v", *tried, want)
	}
}

func TestResolveProfileAutoFallsBackToH264(t *testing.T) {
	tried := fakeProbe(t, "h264_nvenc")

	p, err := ResolveProfile("ffmpeg", config.CodecAuto)
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}
	if p.Key != config.CodecH264 {
		t.Errorf("key = %q, want %q", p.Key, config.CodecH264)
	}
	want := []string{"av1_nvenc", "hevc_nvenc", "h264_nvenc"}
	if strings.Join(*tried, ",") != strings.Join(want, ",") {
		t.Errorf("probed %v, want %v", *tried, want)
	}
}

func TestResolveProfileAutoAllFail(t *testing.T) {
	tried := fakeProbe(t)

	_, err := ResolveProfile("ffmpeg", config.CodecAuto)
	if err == nil {
		t.Fatal("ResolveProfile should fail when no encoder probes clean")
	}
	if !strings.Contains(err.Error(), "no working NVENC encoder found") {
		t.Errorf("error = %v, want the no-NVENC message", err)
	}
	if !strings.Contains(err.Error(), "h264_nvenc") {
		t.Errorf("error = %v, want it to wrap the last probe failure", err)
	}
	if len(*tried) != 3 {
		t.Errorf("probed %v, want all three tried", *tried)
	}
}

func TestResolveProfileExplicitCodec(t *testing.T) {
	tried := fakeProbe(t, "av1_nvenc", "hevc_nvenc", "h264_nvenc")

	p, err := ResolveProfile("ffmpeg", config.CodecHEVC)
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}
	if p.Key != config.CodecHEVC {
		t.Errorf("key = %q, want %q", p.Key, config.CodecHEVC)
	}
	if len(*tried) != 1 || (*tried)[0] != "hevc_nvenc" {
		t.Errorf("probed %v, want only hevc_nvenc", *tried)
	}
}

func TestResolveProfileExplicitCodecNeverFallsBack(t *testing.T) {
	fakeProbe(t, "h264_nvenc")

	_, err := ResolveProfile("ffmpeg", config.CodecAV1)
	if err == nil {
		t.Fatal("an explicit codec that fails to probe should not fall back")
	}
	if !strings.Contains(err.Error(), "av1_nvenc") {
		t.Errorf("error = %v, want it to name av1_nvenc", err)
	}
}

func TestResolveProfileUnknownCodec(t *testing.T) {
	tried := fakeProbe(t, "av1_nvenc")

	_, err := ResolveProfile("ffmpeg", "vp9")
	if err == nil {
		t.Fatal("ResolveProfile should reject an unknown codec")
	}
	if !strings.Contains(err.Error(), `unknown codec "vp9"`) {
		t.Errorf("error = %v, want an unknown codec message", err)
	}
	if len(*tried) != 0 {
		t.Errorf("probed %v, want no probe for an unknown codec", *tried)
	}
}

func TestProfilesTable(t *testing.T) {
	want := map[string]Profile{
		config.CodecAV1:  {config.CodecAV1, "av1_nvenc", ".webm", "libopus"},
		config.CodecHEVC: {config.CodecHEVC, "hevc_nvenc", ".mp4", "aac"},
		config.CodecH264: {config.CodecH264, "h264_nvenc", ".mp4", "aac"},
	}
	if len(Profiles) != len(want) {
		t.Fatalf("Profiles has %d entries, want %d", len(Profiles), len(want))
	}
	for key, w := range want {
		got, ok := Profiles[key]
		if !ok {
			t.Errorf("Profiles is missing %q", key)
			continue
		}
		if got != w {
			t.Errorf("Profiles[%q] = %+v, want %+v", key, got, w)
		}
		if got.Key != key {
			t.Errorf("Profiles[%q].Key = %q, want the map key", key, got.Key)
		}
	}
}

func TestAutoOrder(t *testing.T) {
	want := []string{config.CodecAV1, config.CodecHEVC, config.CodecH264}
	if strings.Join(autoOrder, ",") != strings.Join(want, ",") {
		t.Errorf("autoOrder = %v, want %v", autoOrder, want)
	}
}

func TestProbeEncoderMissingBinary(t *testing.T) {
	err := probeEncoder(filepath.Join(t.TempDir(), "definitely-not-ffmpeg"), "av1_nvenc")
	if err == nil {
		t.Fatal("probeEncoder should fail when ffmpeg is not there")
	}
	if !strings.Contains(err.Error(), "could not query ffmpeg encoders") {
		t.Errorf("error = %v, want the encoder query message", err)
	}
}
