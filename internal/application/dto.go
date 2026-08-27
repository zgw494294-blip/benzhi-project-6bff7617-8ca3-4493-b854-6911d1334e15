package application

import (
	"fieldlingua/internal/crypto"
	"fieldlingua/internal/domain"
	"fieldlingua/internal/quality"
	"time"
)

type CreateProjectInput struct {
	ProjectID       string              `json:"projectID"`
	Title           string              `json:"title"`
	LanguageVariant string              `json:"languageVariant"`
	OwnerID         string              `json:"ownerID"`
	CollectionSites []string            `json:"collectionSites"`
	EthicsStatus    domain.EthicsStatus `json:"ethicsStatus"`
	ConsentRefs     []string            `json:"consentRefs,omitempty"`
	ExpectedVersion int                 `json:"expectedVersion"`
	IdempotencyKey  string              `json:"idempotencyKey"`
}
type AddSegmentInput struct {
	ProjectID       string                  `json:"projectID"`
	Segment         domain.RecordingSegment `json:"segment"`
	ExpectedVersion int                     `json:"expectedVersion"`
	IdempotencyKey  string                  `json:"idempotencyKey"`
}
type SubmitRevisionInput struct {
	ProjectID       string                    `json:"projectID"`
	Revision        domain.TranscriptRevision `json:"revision"`
	Closures        []quality.ClosureEvidence `json:"closures,omitempty"`
	ExpectedVersion int                       `json:"expectedVersion"`
	IdempotencyKey  string                    `json:"idempotencyKey"`
}
type ReviewInput struct {
	ProjectID       string                `json:"projectID"`
	RevisionID      string                `json:"revisionID"`
	ReviewerID      string                `json:"reviewerID"`
	Decision        domain.ReviewDecision `json:"decision"`
	Note            string                `json:"note"`
	ExpectedVersion int                   `json:"expectedVersion"`
	IdempotencyKey  string                `json:"idempotencyKey"`
}
type ReleaseInput struct {
	ProjectID       string `json:"projectID"`
	IssuedBy        string `json:"issuedBy"`
	CredentialID    string `json:"credentialID"`
	ExpectedVersion int    `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
}
type ProjectFilter struct {
	Status       domain.ProjectStatus
	EthicsStatus domain.EthicsStatus
	OwnerID      string
}
type ProjectSummary struct {
	ProjectID      string               `json:"projectID"`
	Title          string               `json:"title"`
	Status         domain.ProjectStatus `json:"status"`
	EthicsStatus   domain.EthicsStatus  `json:"ethicsStatus"`
	OwnerID        string               `json:"ownerID"`
	Version        int                  `json:"version"`
	SegmentCount   int                  `json:"segmentCount"`
	RevisionCount  int                  `json:"revisionCount"`
	OpenIssueCount int                  `json:"openIssueCount"`
	UpdatedAt      time.Time            `json:"updatedAt"`
}
type ProjectDetail struct {
	Summary     ProjectSummary              `json:"summary"`
	Project     *domain.CorpusProject       `json:"project"`
	Segments    []domain.RecordingSegment   `json:"segments"`
	Revisions   []domain.TranscriptRevision `json:"revisions"`
	Credentials []crypto.Credential         `json:"credentials"`
}
type RevisionHistoryItem struct {
	Revision       domain.TranscriptRevision `json:"revision"`
	BaseRevisionID string                    `json:"baseRevisionID,omitempty"`
	IssueCount     int                       `json:"issueCount"`
	OpenIssueCount int                       `json:"openIssueCount"`
	ReviewDecision domain.ReviewDecision     `json:"reviewDecision,omitempty"`
}
type IssueReport struct {
	RevisionID string                 `json:"revisionID"`
	Issues     []domain.Issue         `json:"issues"`
	Summaries  []quality.IssueSummary `json:"summaries"`
	Blocking   bool                   `json:"blocking"`
	Status     string                 `json:"status"`
}
type ReviewQueueItem struct {
	ProjectID      string    `json:"projectID"`
	ProjectTitle   string    `json:"projectTitle"`
	ProjectVersion int       `json:"projectVersion"`
	RevisionID     string    `json:"revisionID"`
	SegmentID      string    `json:"segmentID"`
	SpeakerID      string    `json:"speakerID"`
	StartMillis    int64     `json:"startMillis"`
	EndMillis      int64     `json:"endMillis"`
	BaseRevisionID string    `json:"baseRevisionID,omitempty"`
	OpenIssueCount int       `json:"openIssueCount"`
	SubmittedAt    time.Time `json:"submittedAt"`
}
type VerificationResult struct {
	VerificationStatus string             `json:"verificationStatus"`
	Reason             string             `json:"reason,omitempty"`
	Credential         *crypto.Credential `json:"credential,omitempty"`
	IssuedBy           string             `json:"issuedBy,omitempty"`
	IssuedAt           time.Time          `json:"issuedAt,omitempty"`
	ProjectVersion     int                `json:"projectVersion,omitempty"`
}
