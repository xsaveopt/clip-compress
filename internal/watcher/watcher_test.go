package watcher

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/xsaveopt/clip-compress/internal/config"
)

type fakeClock struct {
	mu     sync.Mutex
	t      time.Time
	onWake func()
}

func newClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) sleep(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	wake := c.onWake
	c.mu.Unlock()
	if wake != nil {
		wake()
	}
}

func testConfig(t *testing.T) (*config.Config, string, string) {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join(root, "Videos")
	out := filepath.Join(src, "ClipCompress")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	cfg := &config.Config{}
	cfg.SetSourceDir(src)
	cfg.SetOutputDir(out)
	return cfg, src, out
}

func newTestWatcher(t *testing.T, cfg *config.Config) (*Watcher, *fakeClock) {
	t.Helper()
	clock := newClock()
	w := New(cfg, func(context.Context, string) {}, nil)
	w.interval = time.Second
	w.needStable = 2 * time.Second
	w.timeout = 30 * time.Second
	w.sleep = clock.sleep
	w.now = clock.now
	return w, clock
}

func withFSW(t *testing.T, w *Watcher) {
	t.Helper()
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("fsnotify.NewWatcher: %v", err)
	}
	t.Cleanup(func() { _ = fsw.Close() })
	w.fsw = fsw
}

func write(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

func drainJobs(t *testing.T, w *Watcher, want int) []string {
	t.Helper()
	got := []string{}
	deadline := time.After(5 * time.Second)
	for len(got) < want {
		select {
		case p := <-w.jobs:
			got = append(got, p)
		case <-deadline:
			t.Fatalf("timed out with %d of %d jobs: %v", len(got), want, got)
		}
	}
	return got
}

func expectNoJob(t *testing.T, w *Watcher) {
	t.Helper()
	select {
	case p := <-w.jobs:
		t.Fatalf("unexpected job %q", p)
	case <-time.After(150 * time.Millisecond):
	}
}

func waitUntil(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal(msg)
}

func relative(t *testing.T, root string, paths []string) []string {
	t.Helper()
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			t.Fatalf("Rel: %v", err)
		}
		out = append(out, filepath.ToSlash(rel))
	}
	slices.Sort(out)
	return out
}

func TestIsWithin(t *testing.T) {
	out := filepath.Join("home", "Videos", "ClipCompress")
	cases := []struct {
		path string
		want bool
	}{
		{filepath.Join(out, "clip (av1).mp4"), true},
		{filepath.Join(out, "sub", "x.mp4"), true},
		{out, true},
		{filepath.Join("home", "Videos", "Game", "clip.mp4"), false},
		{filepath.Join("home", "Videos"), false},
	}
	for _, c := range cases {
		if got := isWithin(c.path, out); got != c.want {
			t.Errorf("isWithin(%q, %q) = %v, want %v", c.path, out, got, c.want)
		}
	}
}

func TestIsWithinEmptyDir(t *testing.T) {
	if isWithin(filepath.Join("a", "b"), "") {
		t.Error("nothing is within an empty dir")
	}
}

func TestIsCandidate(t *testing.T) {
	cfg, src, out := testConfig(t)
	w, _ := newTestWatcher(t, cfg)

	cases := []struct {
		path string
		want bool
	}{
		{filepath.Join(src, "clip.mp4"), true},
		{filepath.Join(src, "sub", "clip.mkv"), true},
		{filepath.Join(src, "shot.png"), true},
		{filepath.Join(src, "notes.txt"), false},
		{filepath.Join(out, "clip (av1).webm"), false},
		{filepath.Join(src, "clip (av1).webm"), false},
		{filepath.Join(src, "clip (hevc).mp4"), false},
		{filepath.Join(src, "clip (h264).mp4"), false},
		{filepath.Join(src, "clip (AV1).webm"), false},
		{filepath.Join(src, "clip (av1) final.mp4"), true},
		{filepath.Join(src, "clip(av1).mp4"), true},
	}
	for _, c := range cases {
		if got := w.isCandidate(c.path); got != c.want {
			t.Errorf("isCandidate(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestAddTreeWatchesEveryDirExceptOutput(t *testing.T) {
	cfg, src, out := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	withFSW(t, w)

	for _, d := range []string{
		filepath.Join(src, "Game"),
		filepath.Join(src, "Game", "Clips"),
		filepath.Join(out, "nested"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}

	w.addTree(src)

	got := relative(t, src, w.fsw.WatchList())
	want := []string{".", "Game", "Game/Clips"}
	if !slices.Equal(got, want) {
		t.Errorf("WatchList = %v, want %v", got, want)
	}
}

func TestAddTreeIgnoresEmptyRoot(t *testing.T) {
	cfg, _, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	withFSW(t, w)

	w.addTree("")

	if got := w.fsw.WatchList(); len(got) != 0 {
		t.Errorf("WatchList = %v, want empty", got)
	}
}

func TestAddTreeMissingRootLogsNothingFatal(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	withFSW(t, w)

	w.addTree(filepath.Join(src, "gone"))

	if got := w.fsw.WatchList(); len(got) != 0 {
		t.Errorf("WatchList = %v, want empty", got)
	}
}

func TestScanExistingQueuesCandidatesOnly(t *testing.T) {
	cfg, src, out := testConfig(t)
	w, _ := newTestWatcher(t, cfg)

	write(t, filepath.Join(src, "a.mp4"), 10)
	write(t, filepath.Join(src, "Game", "b.mkv"), 10)
	write(t, filepath.Join(src, "shot.png"), 10)
	write(t, filepath.Join(src, "notes.txt"), 10)
	write(t, filepath.Join(src, "done (av1).webm"), 10)
	write(t, filepath.Join(out, "a (av1).webm"), 10)
	write(t, filepath.Join(out, "nested", "c (hevc).mp4"), 10)

	w.scanExisting(src)

	got := relative(t, src, drainJobs(t, w, 3))
	want := []string{"Game/b.mkv", "a.mp4", "shot.png"}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("queued %v, want %v", got, want)
	}
	expectNoJob(t, w)
}

func TestScanExistingIgnoresEmptyRoot(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	write(t, filepath.Join(src, "a.mp4"), 10)

	w.scanExisting("")

	expectNoJob(t, w)
}

func TestScanExistingQueuesEachFileOnce(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	write(t, filepath.Join(src, "a.mp4"), 10)

	w.scanExisting(src)
	drainJobs(t, w, 1)

	w.scanExisting(src)
	expectNoJob(t, w)
}

func TestHandleEventQueuesCreatedFile(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	withFSW(t, w)

	path := filepath.Join(src, "a.mp4")
	write(t, path, 10)
	w.handleEvent(fsnotify.Event{Name: path, Op: fsnotify.Create})

	if got := drainJobs(t, w, 1); got[0] != path {
		t.Errorf("queued %q, want %q", got[0], path)
	}
}

func TestHandleEventIgnoresUninterestingOps(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	withFSW(t, w)

	path := filepath.Join(src, "a.mp4")
	write(t, path, 10)
	for _, op := range []fsnotify.Op{fsnotify.Chmod, fsnotify.Remove, fsnotify.Rename} {
		w.handleEvent(fsnotify.Event{Name: path, Op: op})
	}

	expectNoJob(t, w)
}

func TestHandleEventIgnoresMissingFile(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	withFSW(t, w)

	w.handleEvent(fsnotify.Event{Name: filepath.Join(src, "gone.mp4"), Op: fsnotify.Create})

	expectNoJob(t, w)
}

func TestHandleEventIgnoresOutputDirFile(t *testing.T) {
	cfg, _, out := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	withFSW(t, w)

	path := filepath.Join(out, "a (av1).webm")
	write(t, path, 10)
	w.handleEvent(fsnotify.Event{Name: path, Op: fsnotify.Create})

	expectNoJob(t, w)
}

func TestHandleEventIgnoresNonMediaFile(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	withFSW(t, w)

	path := filepath.Join(src, "notes.txt")
	write(t, path, 10)
	w.handleEvent(fsnotify.Event{Name: path, Op: fsnotify.Write})

	expectNoJob(t, w)
}

func TestHandleEventWatchesNewDirectory(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	withFSW(t, w)

	dir := filepath.Join(src, "Game")
	if err := os.MkdirAll(filepath.Join(dir, "Clips"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	w.handleEvent(fsnotify.Event{Name: dir, Op: fsnotify.Create})

	got := relative(t, src, w.fsw.WatchList())
	want := []string{"Game", "Game/Clips"}
	if !slices.Equal(got, want) {
		t.Errorf("WatchList = %v, want %v", got, want)
	}
}

func TestHandleEventDoesNotWatchOutputDirectory(t *testing.T) {
	cfg, _, out := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	withFSW(t, w)

	w.handleEvent(fsnotify.Event{Name: out, Op: fsnotify.Create})

	if got := w.fsw.WatchList(); len(got) != 0 {
		t.Errorf("WatchList = %v, want empty", got)
	}
}

func TestHandleEventWriteOnDirectoryDoesNotWatch(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	withFSW(t, w)

	dir := filepath.Join(src, "Game")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	w.handleEvent(fsnotify.Event{Name: dir, Op: fsnotify.Write})

	if got := w.fsw.WatchList(); len(got) != 0 {
		t.Errorf("WatchList = %v, want empty", got)
	}
}

func TestTrackStableWaitsForTheSizeToSettle(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, clock := newTestWatcher(t, cfg)

	path := filepath.Join(src, "a.mp4")
	write(t, path, 1)

	var grew int
	clock.onWake = func() {
		if grew < 3 {
			grew++
			write(t, path, 1+grew*10)
		}
	}

	w.trackStable(path)
	drainJobs(t, w, 1)

	if grew != 3 {
		t.Fatalf("file grew %d times, want 3", grew)
	}
	elapsed := clock.now().Sub(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if elapsed < w.needStable {
		t.Errorf("settled after %v, want at least %v of stable time", elapsed, w.needStable)
	}
	if elapsed >= w.timeout {
		t.Errorf("settled after %v, want well before the %v timeout", elapsed, w.timeout)
	}
}

func TestTrackStableQueuesOncePerPath(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)

	path := filepath.Join(src, "a.mp4")
	write(t, path, 10)

	w.trackStable(path)
	drainJobs(t, w, 1)
	w.trackStable(path)
	w.trackStable(path)

	expectNoJob(t, w)
}

func TestTrackStableDropsAVanishedFile(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, clock := newTestWatcher(t, cfg)

	path := filepath.Join(src, "a.mp4")
	write(t, path, 10)
	clock.onWake = func() { _ = os.Remove(path) }

	w.trackStable(path)

	expectNoJob(t, w)
	waitUntil(t, func() bool {
		w.mu.Lock()
		defer w.mu.Unlock()
		return !w.inflight[path]
	}, "path left in flight after the file vanished")

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.done[path] {
		t.Error("a vanished file should not be marked done")
	}
}

func TestTrackStableGivesUpOnAnEmptyFileAfterTheTimeout(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, clock := newTestWatcher(t, cfg)

	path := filepath.Join(src, "a.mp4")
	write(t, path, 0)

	w.trackStable(path)
	drainJobs(t, w, 1)

	elapsed := clock.now().Sub(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if elapsed < w.timeout {
		t.Errorf("gave up after %v, want the full %v timeout", elapsed, w.timeout)
	}
}

func TestNewAppliesDefaults(t *testing.T) {
	cfg, _, _ := testConfig(t)
	w := New(cfg, nil, nil)

	if w.interval != time.Second {
		t.Errorf("interval = %v, want 1s", w.interval)
	}
	if w.needStable != 2*time.Second {
		t.Errorf("needStable = %v, want 2s", w.needStable)
	}
	if w.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want 5m", w.timeout)
	}
	if w.log == nil || w.sleep == nil || w.now == nil {
		t.Error("New left a hook nil")
	}
	if cap(w.jobs) != 16 {
		t.Errorf("jobs cap = %d, want 16", cap(w.jobs))
	}
	w.log("no panic please")
}

func TestRewatchWithoutAStartedWatcherIsANoOp(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	write(t, filepath.Join(src, "a.mp4"), 10)

	w.Rewatch()

	expectNoJob(t, w)
}

func TestRewatchRebuildsTheWatchListAfterASourceChange(t *testing.T) {
	cfg, src, _ := testConfig(t)
	w, _ := newTestWatcher(t, cfg)
	withFSW(t, w)

	other := filepath.Join(filepath.Dir(src), "Other")
	if err := os.MkdirAll(filepath.Join(other, "Clips"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	w.addTree(src)

	cfg.SetSourceDir(other)
	cfg.SetOutputDir(filepath.Join(other, "ClipCompress"))
	w.Rewatch()

	got := relative(t, other, w.fsw.WatchList())
	want := []string{".", "Clips"}
	if !slices.Equal(got, want) {
		t.Errorf("WatchList = %v, want %v", got, want)
	}
}
