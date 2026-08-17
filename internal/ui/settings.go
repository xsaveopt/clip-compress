package ui

import (
	"strconv"

	"github.com/gen2brain/iup-go/iup"

	"github.com/xsaveopt/clip-compress/internal/config"
)

type Settings struct {
	dlg            iup.Ihandle
	cfg            *config.Config
	source         iup.Ihandle
	output         iup.Ihandle
	bitrate        iup.Ihandle
	deleteOriginal iup.Ihandle
	notify         iup.Ihandle
	startAtLogin   iup.Ihandle
}

func NewSettings(cfg *config.Config, onSave func()) *Settings {
	s := &Settings{
		cfg:            cfg,
		source:         entry(),
		output:         entry(),
		bitrate:        entry().SetAttributes(map[string]string{"MASK": "/d+", "SIZE": "60x"}),
		deleteOriginal: iup.Toggle(""),
		notify:         iup.Toggle(""),
		startAtLogin:   iup.Toggle(""),
	}

	grid := iup.GridBox(
		iup.Label("Watch folder"), iup.Hbox(s.source, browseButton(s, s.source, "Select watch folder")),
		iup.Label("Output folder"), iup.Hbox(s.output, browseButton(s, s.output, "Select output folder")),
		iup.Label("Bitrate (kbps)"), iup.Hbox(s.bitrate, iup.Fill()),
		iup.Label("Delete original"), iup.Hbox(s.deleteOriginal, iup.Fill()),
		iup.Label("Notifications"), iup.Hbox(s.notify, iup.Fill()),
		iup.Label("Start at login"), iup.Hbox(s.startAtLogin, iup.Fill()),
	).SetAttributes(map[string]string{
		"ORIENTATION":  "HORIZONTAL",
		"NUMDIV":       "2",
		"SIZECOL":      "1",
		"ALIGNMENTLIN": "ACENTER",
		"GAPLIN":       "6",
		"GAPCOL":       "10",
	})

	save := iup.Button("Save").SetAttributes(map[string]string{"PADDING": "16x3"})
	save.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		s.apply()
		if onSave != nil {
			onSave()
		}
		iup.Hide(s.dlg)
		return iup.DEFAULT
	}))

	s.dlg = iup.Dialog(
		iup.Vbox(grid, iup.Hbox(iup.Fill(), save)).SetAttributes(map[string]string{
			"MARGIN": "12x12",
			"GAP":    "12",
		}),
	).SetAttributes(map[string]string{
		"TITLE": "ClipCompress — Settings",
		"SIZE":  "280x",
	})

	iup.SetAttributeHandle(s.dlg, "DEFAULTENTER", save)

	s.dlg.SetCallback("CLOSE_CB", iup.CloseFunc(func(iup.Ihandle) int {
		iup.Hide(s.dlg)
		return iup.IGNORE
	}))

	return s
}

func (s *Settings) Dialog() iup.Ihandle {
	return s.dlg
}

func (s *Settings) Show() {
	s.reload()
	iup.ShowXY(s.dlg, iup.CENTER, iup.CENTER)
	s.dlg.SetAttribute("TOPMOST", "YES")
	s.dlg.SetAttribute("TOPMOST", "NO")
	iup.SetFocus(s.source)
}

func (s *Settings) reload() {
	s.source.SetAttribute("VALUE", s.cfg.SourceDir())
	s.output.SetAttribute("VALUE", s.cfg.OutputDir())
	s.bitrate.SetAttribute("VALUE", strconv.Itoa(s.cfg.VideoBitrateK()))
	s.deleteOriginal.SetAttribute("VALUE", toggleValue(s.cfg.DeleteOriginal()))
	s.notify.SetAttribute("VALUE", toggleValue(s.cfg.Notify()))
	s.startAtLogin.SetAttribute("VALUE", toggleValue(s.cfg.StartAtLogin()))
}

func (s *Settings) apply() {
	s.cfg.SetSourceDir(s.source.GetAttribute("VALUE"))
	s.cfg.SetOutputDir(s.output.GetAttribute("VALUE"))
	s.cfg.SetVideoBitrateK(parseInt(s.bitrate.GetAttribute("VALUE"), s.cfg.VideoBitrateK()))
	s.cfg.SetDeleteOriginal(s.deleteOriginal.GetInt("VALUE") == 1)
	s.cfg.SetNotify(s.notify.GetInt("VALUE") == 1)
	s.cfg.SetStartAtLogin(s.startAtLogin.GetInt("VALUE") == 1)
}

func entry() iup.Ihandle {
	return iup.Text().SetAttributes(map[string]string{"EXPAND": "HORIZONTAL"})
}

func browseButton(s *Settings, target iup.Ihandle, title string) iup.Ihandle {
	b := iup.Button("…").SetAttributes(map[string]string{"PADDING": "8x3"})
	b.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		fd := iup.FileDlg().SetAttributes(map[string]string{
			"DIALOGTYPE": "DIR",
			"TITLE":      title,
			"DIRECTORY":  target.GetAttribute("VALUE"),
		})
		defer fd.Destroy()

		iup.SetAttributeHandle(fd, "PARENTDIALOG", s.dlg)
		iup.Popup(fd, iup.CENTERPARENT, iup.CENTERPARENT)
		if fd.GetInt("STATUS") == -1 {
			return iup.DEFAULT
		}
		target.SetAttribute("VALUE", fd.GetAttribute("VALUE"))
		return iup.DEFAULT
	}))
	return b
}

func toggleValue(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}

func parseInt(s string, fallback int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return fallback
}
