package docs

import (
	"embed"
	"io/fs"
)

//go:embed *.md
var assets embed.FS

func Files() fs.FS { return assets }
