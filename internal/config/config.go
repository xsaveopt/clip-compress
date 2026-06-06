package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
)

const (
	keySourceDir      = "sourceDir"
	keyOutputDir      = "outputDir"
	keyVideoBitrateK  = "videoBitrateK"
	keyDeleteOriginal = "deleteOriginal"
	keyNotify         = "notify"
	keyStartAtLogin   = "startAtLogin"
	keyPaused         = "paused"
)

const (
	CodecAuto = "auto"
	CodecAV1  = "av1"
	CodecHEVC = "hevc"
	CodecH264 = "h264"
)

const AudioBitrateK = 128

var (
	videoExtensions = []string{".mp4", ".mkv", ".mov", ".avi", ".webm"}
	imageExtensions = []string{".png", ".jpg", ".jpeg", ".bmp"}
)

type Config struct {
	p fyne.Preferences
}

func New(p fyne.Preferences) *Config {
	return &Config{p: p}
}

func defaultSourceDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Videos")
}

func defaultOutputDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Videos", "ClipCompress")
}

func (c *Config) SourceDir() string     { return c.p.StringWithFallback(keySourceDir, defaultSourceDir()) }
func (c *Config) SetSourceDir(v string) { c.p.SetString(keySourceDir, v) }

func (c *Config) OutputDir() string     { return c.p.StringWithFallback(keyOutputDir, defaultOutputDir()) }
func (c *Config) SetOutputDir(v string) { c.p.SetString(keyOutputDir, v) }

func (c *Config) VideoBitrateK() int     { return c.p.IntWithFallback(keyVideoBitrateK, 1900) }
func (c *Config) SetVideoBitrateK(v int) { c.p.SetInt(keyVideoBitrateK, v) }

func (c *Config) DeleteOriginal() bool     { return c.p.BoolWithFallback(keyDeleteOriginal, false) }
func (c *Config) SetDeleteOriginal(v bool) { c.p.SetBool(keyDeleteOriginal, v) }

func (c *Config) Notify() bool     { return c.p.BoolWithFallback(keyNotify, false) }
func (c *Config) SetNotify(v bool) { c.p.SetBool(keyNotify, v) }

func (c *Config) StartAtLogin() bool     { return c.p.BoolWithFallback(keyStartAtLogin, true) }
func (c *Config) SetStartAtLogin(v bool) { c.p.SetBool(keyStartAtLogin, v) }

func (c *Config) Paused() bool     { return c.p.BoolWithFallback(keyPaused, false) }
func (c *Config) SetPaused(v bool) { c.p.SetBool(keyPaused, v) }

func (c *Config) Codec() string { return CodecAuto }

func (c *Config) IsWatched(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return slices.Contains(videoExtensions, ext) || slices.Contains(imageExtensions, ext)
}

func (c *Config) IsImage(path string) bool {
	return slices.Contains(imageExtensions, strings.ToLower(filepath.Ext(path)))
}
