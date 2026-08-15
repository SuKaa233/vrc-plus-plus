package web

import (
	"embed"
	"io/fs"
)

//go:embed dist
var assets embed.FS

func Dist() fs.FS {
	sub, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
