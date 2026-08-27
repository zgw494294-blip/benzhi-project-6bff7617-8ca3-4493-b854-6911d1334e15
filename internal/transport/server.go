package transport

import (
	"encoding/json"
	"fieldlingua/internal/application"
	"net/http"
	"time"
)

type Server struct {
	App *application.Service
	mux *http.ServeMux
}

func New(app *application.Service) *Server {
	s := &Server{App: app, mux: http.NewServeMux()}
	s.routes()
	return s
}
func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.health)
	s.mux.HandleFunc("/workbench", s.workbench)
	s.mux.HandleFunc("/api/projects", s.projects)
	s.mux.HandleFunc("/api/projects/", s.projectDetail)
	s.mux.HandleFunc("/api/segments", s.segments)
	s.mux.HandleFunc("/api/revisions", s.revisions)
	s.mux.HandleFunc("/api/reviews", s.reviews)
	s.mux.HandleFunc("/api/issues", s.issues)
	s.mux.HandleFunc("/api/releases", s.releases)
	s.mux.HandleFunc("/api/credentials/verify", s.verify)
	s.mux.HandleFunc("/api/credentials/", s.credential)
	s.mux.HandleFunc("/static/style.css", style)
	s.mux.HandleFunc("/static/app.js", script)
}
func (s *Server) Handler() http.Handler {
	return http.TimeoutHandler(s.mux, 10*time.Second, "请求超时")
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
func write(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
