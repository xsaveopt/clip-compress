package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type classicTheme struct{}

func ClassicTheme() fyne.Theme { return classicTheme{} }

var (
	clFace      = color.NRGBA{R: 0xC0, G: 0xC0, B: 0xC0, A: 0xFF}
	clInput     = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	clText      = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}
	clNavy      = color.NRGBA{R: 0x00, G: 0x00, B: 0x80, A: 0xFF}
	clSelection = color.NRGBA{R: 0x00, G: 0x00, B: 0x80, A: 0x99}
	clBorder    = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xFF}
	clHover     = color.NRGBA{R: 0xD4, G: 0xD0, B: 0xC8, A: 0xFF}
	clShadow    = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x33}
)

func (classicTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground, theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		return clFace
	case theme.ColorNameButton, theme.ColorNameDisabledButton:
		return clFace
	case theme.ColorNameInputBackground:
		return clInput
	case theme.ColorNameForeground:
		return clText
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink:
		return clNavy
	case theme.ColorNameSelection:
		return clSelection
	case theme.ColorNameInputBorder, theme.ColorNameSeparator, theme.ColorNameScrollBar:
		return clBorder
	case theme.ColorNamePlaceHolder, theme.ColorNameDisabled:
		return clBorder
	case theme.ColorNameHover, theme.ColorNamePressed:
		return clHover
	case theme.ColorNameShadow:
		return clShadow
	default:
		return theme.DefaultTheme().Color(name, theme.VariantLight)
	}
}

func (classicTheme) Font(s fyne.TextStyle) fyne.Resource     { return theme.DefaultTheme().Font(s) }
func (classicTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }

func (classicTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameInputRadius, theme.SizeNameSelectionRadius:
		return 0
	case theme.SizeNamePadding:
		return 3
	case theme.SizeNameInputBorder:
		return 1
	default:
		return theme.DefaultTheme().Size(name)
	}
}
