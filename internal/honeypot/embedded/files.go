package embedded

import (
	"embed"
)

//go:embed *.txt *.md
var Files embed.FS
