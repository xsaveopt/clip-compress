package ui

import (
	"image"

	"github.com/gen2brain/iup-go/iup"

	"github.com/xsaveopt/clip-compress/internal/config"
)

const (
	appTitle      = "ClipCompress"
	trayImageName = "clipcompress-tray"
)

const (
	msgStatus = iota
	msgNotify
)

type TrayActions struct {
	ShowSettings func()
	OpenOutput   func()
	TogglePause  func()
	Quit         func()
}

type Tray struct {
	dlg     iup.Ihandle
	cfg     *config.Config
	actions TrayActions
	status  string
}

func NewTray(dlg iup.Ihandle, cfg *config.Config, icon image.Image, actions TrayActions) *Tray {
	t := &Tray{dlg: dlg, cfg: cfg, actions: actions, status: "starting…"}

	img := iup.ImageFromImage(icon)
	iup.SetHandle(trayImageName, img)
	iup.SetAttributeHandle(dlg, "ICON", img)

	iup.Map(dlg)
	dlg.SetAttributes(map[string]string{
		"TRAY":                "YES",
		"TRAYIMAGE":           trayImageName,
		"TRAYTIP":             t.tip(),
		"TRAYTIPBALLOONTITLE": appTitle,
	})

	dlg.SetCallback("TRAYCLICK_CB", iup.TrayClickFunc(func(_ iup.Ihandle, button, pressed, dclick int) int {
		switch {
		case button == 1 && dclick == 1:
			t.actions.ShowSettings()
		case button == 3 && pressed == 1:
			t.popupMenu()
		}
		return iup.DEFAULT
	}))

	dlg.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(_ iup.Ihandle, s string, code int, _ any) int {
		switch code {
		case msgStatus:
			t.status = s
			t.dlg.SetAttribute("TRAYTIP", t.tip())
		case msgNotify:
			t.balloon(s)
		}
		return iup.DEFAULT
	}))

	return t
}

func (t *Tray) SetStatus(s string) {
	iup.PostMessage(t.dlg, s, msgStatus, nil)
}

func (t *Tray) Notify(body string) {
	iup.PostMessage(t.dlg, body, msgNotify, nil)
}

func (t *Tray) tip() string {
	return appTitle + " — " + t.status
}

func (t *Tray) balloon(body string) {
	t.dlg.SetAttribute("TRAYTIPBALLOON", "YES")
	t.dlg.SetAttribute("TRAYTIP", body)
	t.dlg.SetAttribute("TRAYTIPBALLOON", "NO")
}

func (t *Tray) popupMenu() {
	status := iup.MenuItem("Status: " + t.status).SetAttributes(map[string]string{"ACTIVE": "NO"})

	settings := iup.MenuItem("Settings…")
	settings.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		t.actions.ShowSettings()
		return iup.DEFAULT
	}))

	output := iup.MenuItem("Open output folder")
	output.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		t.actions.OpenOutput()
		return iup.DEFAULT
	}))

	pause := iup.MenuItem(t.pauseLabel())
	pause.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		t.actions.TogglePause()
		return iup.DEFAULT
	}))

	quit := iup.MenuItem("Quit")
	quit.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		t.actions.Quit()
		return iup.DEFAULT
	}))

	menu := iup.Menu(status, iup.MenuSeparator(), settings, output, pause, iup.MenuSeparator(), quit)
	defer menu.Destroy()

	iup.Popup(menu, iup.MOUSEPOS, iup.MOUSEPOS)
}

func (t *Tray) pauseLabel() string {
	if t.cfg.Paused() {
		return "Resume"
	}
	return "Pause"
}
