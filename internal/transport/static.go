package transport

import (
	"embed"
	"net/http"
)

//go:embed web/index.html web/style.css web/app.js
var assets embed.FS

func style(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	b, _ := assets.ReadFile("web/style.css")
	w.Write(b)
}
func script(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	b, _ := assets.ReadFile("web/app.js")
	w.Write(b)
}
func (s *Server) workbench(w http.ResponseWriter, r *http.Request) {
	b, _ := assets.ReadFile("web/index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(b)
}
