package web

import (
	"embed"
	"io/fs"
	"net/http"
)

// Assets contains all embedded web UI files (HTML, CSS, JS).
//
//go:embed index.html style.css app.js
var Assets embed.FS

// GetFileSystem returns an http.FileSystem serving the embedded web UI files.
func GetFileSystem() http.FileSystem {
	sub, err := fs.Sub(Assets, ".")
	if err != nil {
		return http.FS(Assets)
	}
	return http.FS(sub)
}
