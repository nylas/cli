package studio

import (
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
)

//go:embed templates/*.gohtml
var templateFiles embed.FS

//go:embed static/css/*.css static/js/*.js
var staticFiles embed.FS

var pageTemplates = template.Must(template.ParseFS(templateFiles, "templates/*.gohtml"))

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTemplates.ExecuteTemplate(w, "studio.gohtml", nil); err != nil {
		slog.Error("studio: render page", "err", err)
	}
}

// handleStatic serves the embedded CSS/JS tree. webguard's CSP is
// script-src 'self' with no 'unsafe-inline', so all scripts load from here.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "static assets unavailable")
		return
	}
	http.StripPrefix("/static/", http.FileServer(http.FS(sub))).ServeHTTP(w, r)
}
