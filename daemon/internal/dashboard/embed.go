package dashboard

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var assets embed.FS

// FS returns the dashboard static files as an fs.FS rooted at dist/.
func FS() (fs.FS, error) {
	return fs.Sub(assets, "dist")
}
