package domain

import "time"

type ProjectStatus string

const (
	StatusDraft  ProjectStatus = "draft"
	StatusReady  ProjectStatus = "ready"
	StatusFrozen ProjectStatus = "frozen"
)

type EthicsStatus string

const (
	EthicsPending    EthicsStatus = "pending"
	EthicsApproved   EthicsStatus = "approved"
	EthicsRestricted EthicsStatus = "restricted"
)

type RecordingSegment struct {
	SegmentID       string    `json:"segmentID"`
	ProjectID       string    `json:"projectID"`
	RecordingDigest string    `json:"recordingDigest"`
	SpeakerID       string    `json:"speakerID"`
	StartMillis     int64     `json:"startMillis"`
	EndMillis       int64     `json:"endMillis"`
	ContextNote     string    `json:"contextNote"`
	ConsentRef      string    `json:"consentRef"`
	MetadataStatus  string    `json:"metadataStatus"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
type TranscriptPart struct {
	StartMillis int64  `json:"startMillis"`
	EndMillis   int64  `json:"endMillis"`
	SpeakerID   string `json:"speakerID"`
	Text        string `json:"text"`
	Annotation  string `json:"annotation"`
}
type Issue struct {
	Code        string     `json:"code"`
	Message     string     `json:"message"`
	StartMillis int64      `json:"startMillis,omitempty"`
	EndMillis   int64      `json:"endMillis,omitempty"`
	Closed      bool       `json:"closed"`
	ClosureNote string     `json:"closureNote,omitempty"`
	ClosedAt    *time.Time `json:"closedAt,omitempty"`
}
type ReviewDecision string

const (
	ReviewPass   ReviewDecision = "pass"
	ReviewReturn ReviewDecision = "return"
	ReviewNote   ReviewDecision = "note"
)

type TranscriptRevision struct {
	RevisionID         string           `json:"revisionID"`
	ProjectID          string           `json:"projectID"`
	SegmentID          string           `json:"segmentID"`
	BaseRevisionID     string           `json:"baseRevisionID"`
	TranscriberID      string           `json:"transcriberID"`
	TranscriptSegments []TranscriptPart `json:"transcriptSegments"`
	ChangeNote         string           `json:"changeNote"`
	CheckIssues        []Issue          `json:"checkIssues"`
	ReviewDecision     ReviewDecision   `json:"reviewDecision"`
	RevisionStatus     string           `json:"revisionStatus"`
	SubmittedAt        time.Time        `json:"submittedAt"`
	CheckedAt          time.Time        `json:"checkedAt"`
	ReviewerID         string           `json:"reviewerID,omitempty"`
	ReviewNote         string           `json:"reviewNote,omitempty"`
	ReviewedAt         *time.Time       `json:"reviewedAt,omitempty"`
}
type CorpusProject struct {
	ProjectID       string                        `json:"projectID"`
	Title           string                        `json:"title"`
	LanguageVariant string                        `json:"languageVariant"`
	CollectionSites []string                      `json:"collectionSites"`
	EthicsStatus    EthicsStatus                  `json:"ethicsStatus"`
	OwnerID         string                        `json:"ownerID"`
	ConsentRefs     []string                      `json:"consentRefs,omitempty"`
	Status          ProjectStatus                 `json:"status"`
	Version         int                           `json:"version"`
	CreatedAt       time.Time                     `json:"createdAt"`
	UpdatedAt       time.Time                     `json:"updatedAt"`
	Segments        map[string]RecordingSegment   `json:"segments"`
	Revisions       map[string]TranscriptRevision `json:"revisions"`
}
