package gui

import (
	"image/color"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type touchTheme struct{}

var _ fyne.Theme = (*touchTheme)(nil)

func (m touchTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (m touchTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m touchTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m touchTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)*2
}
