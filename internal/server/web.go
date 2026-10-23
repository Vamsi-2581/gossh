package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web/index.html
var webFiles embed.FS

func terminalPageHandler(w http.ResponseWriter, r *http.Request) {
	content, err := fs.ReadFile(webFiles, "web/index.html")
	if err != nil {
		http.Error(w, "Failed to load terminal page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}
