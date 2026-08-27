package transport

import (
	"encoding/json"
	"errors"
	"fieldlingua/internal/application"
	"fieldlingua/internal/crypto"
	"fieldlingua/internal/domain"
	"io"
	"net/http"
	"strings"
)

func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}
func method(w http.ResponseWriter, r *http.Request, wanted string) bool {
	if r.Method == wanted {
		return true
	}
	w.Header().Set("Allow", wanted)
	writeError(w, http.StatusMethodNotAllowed, errors.New("请求方法不受支持"))
	return false
}
func writeError(w http.ResponseWriter, status int, err error) {
	payload := map[string]any{"error": err.Error()}
	var validation *domain.ValidationError
	if errors.As(err, &validation) {
		payload["fields"] = validation.Fields
	}
	var conflict *domain.VersionConflictError
	if errors.As(err, &conflict) {
		payload["expectedVersion"] = conflict.ExpectedVersion
		payload["currentVersion"] = conflict.CurrentVersion
		payload["projectID"] = conflict.ProjectID
		payload["reloadRequired"] = true
	}
	if errors.Is(err, domain.ErrVersionConflict) || errors.Is(err, domain.ErrConflict) {
		status = http.StatusConflict
	}
	if errors.Is(err, domain.ErrNotFound) {
		status = http.StatusNotFound
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	write(w, payload)
}

func (s *Server) projects(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		filter := application.ProjectFilter{Status: domain.ProjectStatus(r.URL.Query().Get("status")), EthicsStatus: domain.EthicsStatus(r.URL.Query().Get("ethicsStatus")), OwnerID: r.URL.Query().Get("ownerID")}
		write(w, map[string]any{"projects": s.App.ListProjects(filter)})
		return
	}
	if !method(w, r, http.MethodPost) {
		return
	}
	var in application.CreateProjectInput
	if err := decode(r, &in); err != nil {
		writeError(w, 400, err)
		return
	}
	out, err := s.App.Create(in)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	write(w, out)
}
func (s *Server) projectDetail(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodGet) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/projects/"), "/")
	if id == "" {
		writeError(w, 404, domain.ErrNotFound)
		return
	}
	out, err := s.App.ProjectDetail(id)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	write(w, out)
}
func (s *Server) segments(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}
	var in application.AddSegmentInput
	if err := decode(r, &in); err != nil {
		writeError(w, 400, err)
		return
	}
	out, err := s.App.AddSegment(in)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	write(w, out)
}
func (s *Server) revisions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		out, err := s.App.RevisionHistory(r.URL.Query().Get("projectID"), r.URL.Query().Get("segmentID"))
		if err != nil {
			writeError(w, 400, err)
			return
		}
		write(w, map[string]any{"revisions": out})
		return
	}
	if !method(w, r, http.MethodPost) {
		return
	}
	var in application.SubmitRevisionInput
	if err := decode(r, &in); err != nil {
		writeError(w, 400, err)
		return
	}
	out, err := s.App.SubmitRevision(in)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	write(w, out)
}
func (s *Server) issues(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodGet) {
		return
	}
	out, err := s.App.Issues(r.URL.Query().Get("projectID"), r.URL.Query().Get("revisionID"), r.URL.Query().Get("code"))
	if err != nil {
		writeError(w, 400, err)
		return
	}
	write(w, out)
}
func (s *Server) reviews(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		out, err := s.App.ReviewQueue(r.URL.Query().Get("projectID"))
		if err != nil {
			writeError(w, 400, err)
			return
		}
		write(w, map[string]any{"reviews": out})
		return
	}
	if !method(w, r, http.MethodPost) {
		return
	}
	var in application.ReviewInput
	if err := decode(r, &in); err != nil {
		writeError(w, 400, err)
		return
	}
	out, err := s.App.Review(in)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	write(w, out)
}
func (s *Server) releases(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}
	var in application.ReleaseInput
	if err := decode(r, &in); err != nil {
		writeError(w, 400, err)
		return
	}
	out, err := s.App.ReleaseInput(in)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	write(w, out)
}
func (s *Server) verify(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodPost) {
		return
	}
	var in struct {
		CredentialID string             `json:"credentialID"`
		Credential   *crypto.Credential `json:"credential,omitempty"`
	}
	if err := decode(r, &in); err != nil {
		writeError(w, 400, err)
		return
	}
	out, err := s.App.VerifyCredential(strings.TrimSpace(in.CredentialID), in.Credential)
	if err != nil {
		status := 400
		if errors.Is(err, domain.ErrNotFound) {
			status = 404
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		write(w, out)
		return
	}
	write(w, out)
}
func (s *Server) credential(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, http.MethodGet) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/credentials/"), "/")
	credential, err := s.App.Store.GetCredential(id)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	write(w, map[string]any{"credential": credential, "verificationStatus": credential.VerificationStatus})
}
