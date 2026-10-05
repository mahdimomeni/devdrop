package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var embeddedFS embed.FS

// GetFileSystem returns an http.FileSystem serving the embedded Vite SPA
func GetFileSystem() (http.FileSystem, error) {
	distSub, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(distSub), nil
}
