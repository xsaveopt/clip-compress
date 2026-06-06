package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/sratabix/clip-compress/internal/config"
)

type TrayActions struct {
	ShowSettings func()
	OpenOutput   func()
	TogglePause  func()
	Quit         func()
}

type Tray struct {
	app        fyne.App
	icon       fyne.Resource
	cfg        *config.Config
	actions    TrayActions
	statusItem *fyne.MenuItem
	pauseItem  *fyne.MenuItem
	menu       *fyne.Menu
}

func NewTray(app fyne.App, cfg *config.Config, icon fyne.Resource, actions TrayActions) *Tray {
	t := &Tray{app: app, icon: icon, cfg: cfg, actions: actions}
	t.statusItem = fyne.NewMenuItem("Status: starting…", nil)
	t.statusItem.Disabled = true
	t.pauseItem = fyne.NewMenuItem(t.pauseLabel(), func() {
		actions.TogglePause()
		t.refresh()
	})
	t.menu = t.build()

	if desk, ok := app.(desktop.App); ok {
		desk.SetSystemTrayIcon(icon)
		desk.SetSystemTrayMenu(t.menu)
	}
	return t
}

func (t *Tray) build() *fyne.Menu {
	return fyne.NewMenu("ClipCompress",
		t.statusItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Settings…", t.actions.ShowSettings),
		fyne.NewMenuItem("Open output folder", t.actions.OpenOutput),
		t.pauseItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", t.actions.Quit),
	)
}

func (t *Tray) pauseLabel() string {
	if t.cfg.Paused() {
		return "Resume"
	}
	return "Pause"
}

func (t *Tray) SetStatus(s string) {
	fyne.Do(func() {
		t.statusItem.Label = "Status: " + s
		t.refresh()
	})
}

func (t *Tray) refresh() {
	t.pauseItem.Label = t.pauseLabel()
	if desk, ok := t.app.(desktop.App); ok {
		desk.SetSystemTrayMenu(t.menu)
	}
}
