package ui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/xsaveopt/clip-compress/internal/config"
)

func NewSettingsWindow(app fyne.App, cfg *config.Config, onSave func()) fyne.Window {
	w := app.NewWindow("ClipCompress — Settings")

	source := widget.NewEntry()
	source.SetText(cfg.SourceDir())
	output := widget.NewEntry()
	output.SetText(cfg.OutputDir())

	bitrate := widget.NewEntry()
	bitrate.SetText(strconv.Itoa(cfg.VideoBitrateK()))

	deleteOriginal := widget.NewCheck("", nil)
	deleteOriginal.SetChecked(cfg.DeleteOriginal())
	notify := widget.NewCheck("", nil)
	notify.SetChecked(cfg.Notify())
	startAtLogin := widget.NewCheck("", nil)
	startAtLogin.SetChecked(cfg.StartAtLogin())

	form := widget.NewForm(
		widget.NewFormItem("Watch folder", withBrowse(w, source)),
		widget.NewFormItem("Output folder", withBrowse(w, output)),
		widget.NewFormItem("Bitrate (kbps)", bitrate),
		widget.NewFormItem("Delete original", deleteOriginal),
		widget.NewFormItem("Notifications", notify),
		widget.NewFormItem("Start at login", startAtLogin),
	)

	save := widget.NewButton("Save", func() {
		cfg.SetSourceDir(source.Text)
		cfg.SetOutputDir(output.Text)
		cfg.SetVideoBitrateK(parseInt(bitrate.Text, cfg.VideoBitrateK()))
		cfg.SetDeleteOriginal(deleteOriginal.Checked)
		cfg.SetNotify(notify.Checked)
		cfg.SetStartAtLogin(startAtLogin.Checked)
		if onSave != nil {
			onSave()
		}
		w.Hide()
	})
	save.Importance = widget.HighImportance

	content := container.NewBorder(nil, save, nil, nil, form)
	w.SetContent(content)
	w.Resize(fyne.NewSize(480, 320))
	w.SetCloseIntercept(w.Hide)
	return w
}

func withBrowse(w fyne.Window, entry *widget.Entry) fyne.CanvasObject {
	browse := widget.NewButton("Browse…", func() {
		dialog.ShowFolderOpen(func(list fyne.ListableURI, err error) {
			if err != nil || list == nil {
				return
			}
			entry.SetText(list.Path())
		}, w)
	})
	return container.NewBorder(nil, nil, nil, browse, entry)
}

func parseInt(s string, fallback int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return fallback
}
