package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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

type data struct {
	SourceDir      string `json:"sourceDir"`
	OutputDir      string `json:"outputDir"`
	VideoBitrateK  int    `json:"videoBitrateK"`
	DeleteOriginal bool   `json:"deleteOriginal"`
	Notify         bool   `json:"notify"`
	StartAtLogin   *bool  `json:"startAtLogin,omitempty"`
	Paused         bool   `json:"paused"`
}

type Config struct {
	mu   sync.RWMutex
	path string
	d    data
}

func Load(dataName string) (*Config, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(base, dataName, "config.json")

	c := &Config{
		path: path,
		d: data{
			SourceDir:     defaultSourceDir(),
			OutputDir:     defaultOutputDir(),
			VideoBitrateK: 1900,
		},
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(raw, &c.d); err != nil {
		return c, err
	}
	return c, nil
}

func (c *Config) Save() error {
	c.mu.RLock()
	raw, err := json.MarshalIndent(c.d, "", "  ")
	c.mu.RUnlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
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

func (c *Config) SourceDir() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.d.SourceDir
}

func (c *Config) SetSourceDir(v string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.d.SourceDir = v
}

func (c *Config) OutputDir() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.d.OutputDir
}

func (c *Config) SetOutputDir(v string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.d.OutputDir = v
}

func (c *Config) VideoBitrateK() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.d.VideoBitrateK
}

func (c *Config) SetVideoBitrateK(v int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.d.VideoBitrateK = v
}

func (c *Config) DeleteOriginal() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.d.DeleteOriginal
}

func (c *Config) SetDeleteOriginal(v bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.d.DeleteOriginal = v
}

func (c *Config) Notify() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.d.Notify
}

func (c *Config) SetNotify(v bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.d.Notify = v
}

func (c *Config) StartAtLogin() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.d.StartAtLogin != nil && *c.d.StartAtLogin
}

func (c *Config) SetStartAtLogin(v bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.d.StartAtLogin = &v
}

func (c *Config) StartAtLoginSet() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.d.StartAtLogin != nil
}

func (c *Config) Paused() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.d.Paused
}

func (c *Config) SetPaused(v bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.d.Paused = v
}

func (c *Config) Codec() string { return CodecAuto }

func (c *Config) IsWatched(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return slices.Contains(videoExtensions, ext) || slices.Contains(imageExtensions, ext)
}

func (c *Config) IsImage(path string) bool {
	return slices.Contains(imageExtensions, strings.ToLower(filepath.Ext(path)))
}
