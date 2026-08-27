package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fieldlingua/internal/domain"
	"fmt"
	"strings"
	"time"
)

type Credential struct {
	CredentialID       string    `json:"credentialID"`
	ProjectID          string    `json:"projectID"`
	RevisionIDs        []string  `json:"revisionIDs"`
	ContentDigest      string    `json:"contentDigest"`
	ProjectVersion     int       `json:"projectVersion"`
	IssuedBy           string    `json:"issuedBy"`
	IssuedAt           time.Time `json:"issuedAt"`
	Signature          string    `json:"signature"`
	VerificationStatus string    `json:"verificationStatus"`
}

func Issue(p *domain.CorpusProject, id, by string) (Credential, error) {
	if strings.TrimSpace(id) == "" {
		return Credential{}, domain.InvalidField("credentialID", "凭据 ID 不能为空")
	}
	if strings.TrimSpace(by) == "" {
		return Credential{}, domain.InvalidField("issuedBy", "签发人不能为空")
	}
	d, e := Digest(p)
	if e != nil {
		return Credential{}, e
	}
	c := Credential{CredentialID: id, ProjectID: p.ProjectID, ContentDigest: d, ProjectVersion: p.Version, IssuedBy: by, IssuedAt: time.Now().UTC(), VerificationStatus: "valid"}
	for _, r := range p.HeadRevisions() {
		c.RevisionIDs = append(c.RevisionIDs, r.RevisionID)
	}
	c.Signature = sign(c)
	return c, nil
}
func sign(c Credential) string {
	b, _ := json.Marshal(struct {
		ID, Project, Digest, By string
		Version                 int
	}{c.CredentialID, c.ProjectID, c.ContentDigest, c.IssuedBy, c.ProjectVersion})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func Verify(c Credential, p *domain.CorpusProject) error {
	if e := ValidateFormat(c); e != nil {
		return e
	}
	d, e := Digest(p)
	if e != nil {
		return e
	}
	if d != c.ContentDigest || p.Version != c.ProjectVersion {
		return fmt.Errorf("凭据内容摘要或项目版本不匹配")
	}
	revisions := p.HeadRevisions()
	if len(revisions) != len(c.RevisionIDs) {
		return fmt.Errorf("凭据修订列表与项目不匹配")
	}
	for i, revision := range revisions {
		if revision.RevisionID != c.RevisionIDs[i] {
			return fmt.Errorf("凭据修订列表与项目不匹配")
		}
	}
	if sign(c) != c.Signature {
		return fmt.Errorf("凭据签名不匹配")
	}
	return nil
}
