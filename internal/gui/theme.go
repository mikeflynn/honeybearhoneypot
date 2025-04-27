package gui

import (
	"image/color"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type touchTheme struct{}

var _ fyne.Theme = (*touchTheme)(nil)

func (m touchTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{R: 0xd5, G: 0xa0, B: 0x5c, A: 0xff}
	case theme.ColorNameButton:
		return color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x00}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (m touchTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m touchTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m touchTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name) * 2
}
