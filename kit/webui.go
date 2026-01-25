package kit

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed webui/*
var webUIFiles embed.FS

// WebUIHandler returns an HTTP handler that serves the embedded web UI files
func WebUIHandler() http.Handler {
	// Strip "webui/" prefix from embedded files
	subFS, err := fs.Sub(webUIFiles, "webui")
	if err != nil {
		LogError(err)
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(subFS))
}
