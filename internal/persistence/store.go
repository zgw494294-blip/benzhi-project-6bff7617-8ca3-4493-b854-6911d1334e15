package persistence

import (
	"encoding/json"
	"fieldlingua/internal/crypto"
	"fieldlingua/internal/domain"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type IdempotencyRecord struct {
	Operation  string                `json:"operation"`
	ProjectID  string                `json:"projectID"`
	Project    *domain.CorpusProject `json:"project,omitempty"`
	Credential *crypto.Credential    `json:"credential,omitempty"`
}

type Store struct {
	mu          sync.RWMutex
	dir         string
	projects    map[string]*domain.CorpusProject
	credentials map[string]crypto.Credential
	idem        map[string]IdempotencyRecord
}

func New(dir string) *Store {
	s := &Store{dir: dir, projects: map[string]*domain.CorpusProject{}, credentials: map[string]crypto.Credential{}, idem: map[string]IdempotencyRecord{}}
	s.load()
	return s
}

func cloneProject(p *domain.CorpusProject) *domain.CorpusProject {
	if p == nil {
		return nil
	}
	b, _ := json.Marshal(p)
	var cp domain.CorpusProject
	_ = json.Unmarshal(b, &cp)
	if cp.Segments == nil {
		cp.Segments = map[string]domain.RecordingSegment{}
	}
	if cp.Revisions == nil {
		cp.Revisions = map[string]domain.TranscriptRevision{}
	}
	return &cp
}

// cloneCredential 复制凭据的 RevisionIDs 切片，避免调用方修改返回结果时污染 Store 的共享状态。
func cloneCredential(c crypto.Credential) crypto.Credential {
	if c.RevisionIDs != nil {
		c.RevisionIDs = append([]string(nil), c.RevisionIDs...)
	}
	return c
}

func (s *Store) load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
}
func (s *Store) loadLocked() {
	b, err := os.ReadFile(filepath.Join(s.dir, "snapshot.json"))
	if err != nil {
		return
	}
	var snapshot struct {
		Projects    map[string]*domain.CorpusProject `json:"projects"`
		Credentials map[string]crypto.Credential     `json:"credentials"`
		Idempotency map[string]IdempotencyRecord     `json:"idempotency"`
	}
	if json.Unmarshal(b, &snapshot) != nil {
		return
	}
	if snapshot.Projects != nil {
		s.projects = snapshot.Projects
	}
	if snapshot.Credentials != nil {
		s.credentials = snapshot.Credentials
	}
	if snapshot.Idempotency != nil {
		s.idem = snapshot.Idempotency
	}
}
func (s *Store) saveLocked() error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}
	b, err := json.Marshal(struct {
		Projects    map[string]*domain.CorpusProject `json:"projects"`
		Credentials map[string]crypto.Credential     `json:"credentials"`
		Idempotency map[string]IdempotencyRecord     `json:"idempotency"`
	}{s.projects, s.credentials, s.idem})
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dir, "snapshot.tmp")
	if err = os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.dir, "snapshot.json"))
}

// CreateProject 将项目创建与幂等记录放在同一临界区和快照写入中。
func (s *Store) CreateProject(p *domain.CorpusProject, operation, key string) (*domain.CorpusProject, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key != "" {
		if rec, ok := s.idem[key]; ok {
			if rec.Operation != operation || rec.ProjectID != p.ProjectID {
				return nil, false, domain.ErrConflict
			}
			return cloneProject(rec.Project), true, nil
		}
	}
	if _, ok := s.projects[p.ProjectID]; ok {
		return nil, false, domain.ErrConflict
	}
	cp := cloneProject(p)
	s.projects[p.ProjectID] = cp
	if key != "" {
		s.idem[key] = IdempotencyRecord{Operation: operation, ProjectID: p.ProjectID, Project: cloneProject(cp)}
	}
	if err := s.saveLocked(); err != nil {
		return nil, false, err
	}
	return cloneProject(cp), false, nil
}

// ApplyProjectCommand 对版本校验、领域变更、幂等记录和快照写入进行原子串行化。
func (s *Store) ApplyProjectCommand(projectID string, expectedVersion int, operation, key string, mutate func(*domain.CorpusProject) error) (*domain.CorpusProject, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key != "" {
		if rec, ok := s.idem[key]; ok {
			if rec.Operation != operation || rec.ProjectID != projectID {
				return nil, false, domain.ErrConflict
			}
			return cloneProject(rec.Project), true, nil
		}
	}
	current, ok := s.projects[projectID]
	if !ok {
		return nil, false, domain.ErrNotFound
	}
	if expectedVersion > 0 && current.Version != expectedVersion {
		return nil, false, &domain.VersionConflictError{ExpectedVersion: expectedVersion, CurrentVersion: current.Version, ProjectID: projectID}
	}
	next := cloneProject(current)
	if err := mutate(next); err != nil {
		return nil, false, err
	}
	s.projects[projectID] = next
	if key != "" {
		s.idem[key] = IdempotencyRecord{Operation: operation, ProjectID: projectID, Project: cloneProject(next)}
	}
	if err := s.saveLocked(); err != nil {
		return nil, false, err
	}
	return cloneProject(next), false, nil
}

func (s *Store) ReleaseProject(projectID string, expectedVersion int, operation, key, credentialID string, issue func(*domain.CorpusProject) (crypto.Credential, error)) (*domain.CorpusProject, crypto.Credential, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key != "" {
		if rec, ok := s.idem[key]; ok {
			if rec.Operation != operation || rec.ProjectID != projectID || rec.Credential == nil {
				return nil, crypto.Credential{}, false, domain.ErrConflict
			}
			return cloneProject(rec.Project), cloneCredential(*rec.Credential), true, nil
		}
	}
	current, ok := s.projects[projectID]
	if !ok {
		return nil, crypto.Credential{}, false, domain.ErrNotFound
	}
	if expectedVersion > 0 && current.Version != expectedVersion {
		return nil, crypto.Credential{}, false, &domain.VersionConflictError{ExpectedVersion: expectedVersion, CurrentVersion: current.Version, ProjectID: projectID}
	}
	if existing, exists := s.credentials[credentialID]; exists {
		if existing.ProjectID != projectID {
			return nil, crypto.Credential{}, false, domain.ErrConflict
		}
		if err := crypto.Verify(existing, current); err != nil {
			return nil, crypto.Credential{}, false, domain.ErrConflict
		}
		return cloneProject(current), cloneCredential(existing), true, nil
	}
	next := cloneProject(current)
	if err := next.Freeze(); err != nil {
		return nil, crypto.Credential{}, false, err
	}
	credential, err := issue(next)
	if err != nil {
		return nil, crypto.Credential{}, false, err
	}
	s.projects[projectID] = next
	s.credentials[credential.CredentialID] = cloneCredential(credential)
	if key != "" {
		storedCredential := cloneCredential(credential)
		s.idem[key] = IdempotencyRecord{Operation: operation, ProjectID: projectID, Project: cloneProject(next), Credential: &storedCredential}
	}
	if err = s.saveLocked(); err != nil {
		return nil, crypto.Credential{}, false, err
	}
	return cloneProject(next), cloneCredential(credential), false, nil
}

func (s *Store) PutProject(p *domain.CorpusProject) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[p.ProjectID] = cloneProject(p)
	_ = s.saveLocked()
}
func (s *Store) GetProject(id string) (*domain.CorpusProject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.projects[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cloneProject(p), nil
}
func (s *Store) ListProjects() []*domain.CorpusProject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.CorpusProject, 0, len(s.projects))
	for _, p := range s.projects {
		out = append(out, cloneProject(p))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ProjectID < out[j].ProjectID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}
func (s *Store) PutCredential(c crypto.Credential) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[c.CredentialID] = cloneCredential(c)
	_ = s.saveLocked()
}
func (s *Store) GetCredential(id string) (crypto.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.credentials[id]
	if !ok {
		return c, domain.ErrNotFound
	}
	return cloneCredential(c), nil
}
func (s *Store) CredentialsForProject(projectID string) []crypto.Credential {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []crypto.Credential{}
	for _, c := range s.credentials {
		if c.ProjectID == projectID {
			out = append(out, cloneCredential(c))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].IssuedAt.After(out[j].IssuedAt) })
	return out
}
func (s *Store) Idempotent(key string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.idem[key]
	if !ok {
		return nil, false
	}
	b, _ := json.Marshal(rec)
	return b, true
}
func (s *Store) SaveIdempotent(key string, v []byte) {
	var rec IdempotencyRecord
	if json.Unmarshal(v, &rec) != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idem[key] = rec
	_ = s.saveLocked()
}
