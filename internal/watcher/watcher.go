package watcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/xsaveopt/clip-compress/internal/config"
)

type Handler func(ctx context.Context, path string)

type Watcher struct {
	cfg     *config.Config
	handler Handler
	log     func(string)

	fsw  *fsnotify.Watcher
	jobs chan string

	mu       sync.Mutex
	inflight map[string]bool
	done     map[string]bool
}

func New(cfg *config.Config, handler Handler, log func(string)) *Watcher {
	if log == nil {
		log = func(string) {}
	}
	return &Watcher{
		cfg:      cfg,
		handler:  handler,
		log:      log,
		jobs:     make(chan string, 16),
		inflight: map[string]bool{},
		done:     map[string]bool{},
	}
}

func (w *Watcher) Start(ctx context.Context) error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.fsw = fsw
	defer func() { _ = fsw.Close() }()

	w.addTree(w.cfg.SourceDir())
	go w.worker(ctx)
	w.scanExisting(w.cfg.SourceDir())

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-fsw.Events:
			if !ok {
				return nil
			}
			w.handleEvent(event)
		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			w.log("watch error: " + err.Error())
		}
	}
}

func (w *Watcher) Rewatch() {
	if w.fsw == nil {
		return
	}
	for _, p := range w.fsw.WatchList() {
		_ = w.fsw.Remove(p)
	}
	w.addTree(w.cfg.SourceDir())
	w.scanExisting(w.cfg.SourceDir())
}

func (w *Watcher) scanExisting(root string) {
	if root == "" {
		return
	}
	out := w.cfg.OutputDir()
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, _ error) error {
		if d == nil {
			return nil
		}
		if d.IsDir() {
			if isWithin(path, out) {
				return filepath.SkipDir
			}
			return nil
		}
		if w.isCandidate(path) {
			w.trackStable(path)
		}
		return nil
	})
}

func (w *Watcher) addTree(root string) {
	if root == "" {
		return
	}
	out := w.cfg.OutputDir()
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, _ error) error {
		if d == nil || !d.IsDir() {
			return nil
		}
		if isWithin(path, out) {
			return filepath.SkipDir
		}
		w.addWatch(path)
		return nil
	})
}

func (w *Watcher) addWatch(path string) {
	if err := w.fsw.Add(path); err != nil {
		w.log("could not watch " + path + ": " + err.Error())
	}
}

func (w *Watcher) handleEvent(e fsnotify.Event) {
	if e.Op&(fsnotify.Create|fsnotify.Write) == 0 {
		return
	}
	info, err := os.Stat(e.Name)
	if err != nil {
		return
	}
	if info.IsDir() {
		if e.Op&fsnotify.Create != 0 && !isWithin(e.Name, w.cfg.OutputDir()) {
			w.addTree(e.Name)
		}
		return
	}
	if w.isCandidate(e.Name) {
		w.trackStable(e.Name)
	}
}

func (w *Watcher) isCandidate(path string) bool {
	if isWithin(path, w.cfg.OutputDir()) {
		return false
	}
	stem := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	for _, key := range []string{config.CodecAV1, config.CodecHEVC, config.CodecH264} {
		if strings.HasSuffix(stem, " ("+key+")") {
			return false
		}
	}
	return w.cfg.IsWatched(path)
}

func (w *Watcher) trackStable(path string) {
	w.mu.Lock()
	if w.inflight[path] || w.done[path] {
		w.mu.Unlock()
		return
	}
	w.inflight[path] = true
	w.mu.Unlock()

	go func() {
		defer func() {
			w.mu.Lock()
			delete(w.inflight, path)
			w.mu.Unlock()
		}()

		var last int64 = -1
		stableFor := time.Duration(0)
		const interval = time.Second
		const needStable = 2 * time.Second
		const timeout = 5 * time.Minute
		deadline := time.Now().Add(timeout)

		for time.Now().Before(deadline) {
			time.Sleep(interval)
			info, err := os.Stat(path)
			if err != nil {
				return
			}
			if info.Size() == last && info.Size() > 0 {
				stableFor += interval
				if stableFor >= needStable {
					break
				}
			} else {
				stableFor = 0
				last = info.Size()
			}
		}

		w.mu.Lock()
		if w.done[path] {
			w.mu.Unlock()
			return
		}
		w.done[path] = true
		w.mu.Unlock()

		select {
		case w.jobs <- path:
		default:
			w.jobs <- path
		}
	}()
}

func (w *Watcher) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case path := <-w.jobs:
			if w.cfg.Paused() {
				w.log("paused; skipping " + filepath.Base(path))
				continue
			}
			w.handler(ctx, path)
		}
	}
}

func isWithin(path, dir string) bool {
	if dir == "" {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != "..")
}
