package assets

import (
	"embed"
)

//go:embed *.jpg *.png
var Images embed.FS
