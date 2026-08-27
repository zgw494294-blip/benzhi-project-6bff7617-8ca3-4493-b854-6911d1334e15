package domain

import (
	"strings"
	"time"
)

func NewProject(id, title, variant, owner string, sites []string, ethics EthicsStatus) (*CorpusProject, error) {
	return NewProjectWithConsent(id, title, variant, owner, sites, ethics, nil)
}

func NewProjectWithConsent(id, title, variant, owner string, sites []string, ethics EthicsStatus, consentRefs []string) (*CorpusProject, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(title) == "" || strings.TrimSpace(variant) == "" || strings.TrimSpace(owner) == "" || len(sites) == 0 {
		return nil, ErrInvalid
	}
	if ethics != "pending" && ethics != "approved" && ethics != "restricted" {
		return nil, ErrInvalid
	}
	now := time.Now().UTC()
	return &CorpusProject{ProjectID: id, Title: title, LanguageVariant: variant, CollectionSites: append([]string(nil), sites...), EthicsStatus: ethics, OwnerID: owner, ConsentRefs: append([]string(nil), consentRefs...), Status: StatusDraft, Version: 1, CreatedAt: now, UpdatedAt: now, Segments: map[string]RecordingSegment{}, Revisions: map[string]TranscriptRevision{}}, nil
}
func (p *CorpusProject) touch() { p.Version++; p.UpdatedAt = time.Now().UTC() }
func (p *CorpusProject) AddSegment(s RecordingSegment) error {
	if p.Status == StatusFrozen {
		return InvalidField("projectID", "项目已冻结，不能再登记片段")
	}
	fields := map[string]string{}
	if strings.TrimSpace(s.ProjectID) == "" {
		fields["projectID"] = "项目 ID 不能为空"
	} else if s.ProjectID != p.ProjectID {
		fields["projectID"] = "片段项目归属与目标项目不一致"
	}
	if strings.TrimSpace(s.SegmentID) == "" {
		fields["segment.segmentID"] = "片段 ID 不能为空"
	}
	if strings.TrimSpace(s.SpeakerID) == "" {
		fields["segment.speakerID"] = "说话人不能为空"
	}
	if strings.TrimSpace(s.RecordingDigest) == "" {
		fields["segment.recordingDigest"] = "录音摘要不能为空"
	}
	if strings.TrimSpace(s.ConsentRef) == "" {
		fields["segment.consentRef"] = "同意凭据不能为空"
	}
	if s.StartMillis < 0 {
		fields["segment.startMillis"] = "开始时间不能小于 0"
	}
	if s.EndMillis <= s.StartMillis {
		fields["segment.endMillis"] = "结束时间必须大于开始时间"
	}
	if len(fields) > 0 {
		return &ValidationError{Message: "录音片段元数据不完整", Fields: fields}
	}
	if _, ok := p.Segments[s.SegmentID]; ok {
		return InvalidField("segment.segmentID", "片段 ID 已存在")
	}
	if p.EthicsStatus == EthicsRestricted && !contains(p.ConsentRefs, s.ConsentRef) {
		return InvalidField("segment.consentRef", "受限伦理项目仅允许使用项目登记的同意凭据")
	}
	for _, existing := range p.Segments {
		if existing.RecordingDigest == s.RecordingDigest && existing.SpeakerID == s.SpeakerID && s.StartMillis < existing.EndMillis && existing.StartMillis < s.EndMillis {
			return InvalidField("segment.startMillis", "同一录音与说话人的时间区间与已有片段重叠")
		}
	}
	now := time.Now().UTC()
	s.MetadataStatus = "valid"
	s.CreatedAt, s.UpdatedAt = now, now
	p.Segments[s.SegmentID] = s
	p.touch()
	return nil
}
func (p *CorpusProject) AddRevision(r TranscriptRevision) error {
	if p.Status == StatusFrozen {
		return InvalidField("projectID", "项目已冻结，不能再提交修订")
	}
	if r.ProjectID != p.ProjectID || r.RevisionID == "" || r.SegmentID == "" || r.TranscriberID == "" || strings.TrimSpace(r.ChangeNote) == "" {
		return ErrInvalid
	}
	if _, ok := p.Segments[r.SegmentID]; !ok {
		return ErrInvalid
	}
	if _, ok := p.Revisions[r.RevisionID]; ok {
		return InvalidField("revision.revisionID", "修订 ID 已存在")
	}
	count := 0
	for _, existing := range p.Revisions {
		if existing.SegmentID == r.SegmentID {
			count++
		}
	}
	if count == 0 && r.BaseRevisionID != "" {
		return InvalidField("revision.baseRevisionID", "首个修订的基线必须为空")
	}
	if count > 0 && r.BaseRevisionID == "" {
		return InvalidField("revision.baseRevisionID", "后续修订必须引用同一片段的已有基线")
	}
	if r.BaseRevisionID != "" {
		base, ok := p.Revisions[r.BaseRevisionID]
		if !ok {
			return InvalidField("revision.baseRevisionID", "基线修订不存在")
		}
		if base.ProjectID != p.ProjectID || base.SegmentID != r.SegmentID {
			return InvalidField("revision.baseRevisionID", "基线修订不属于同一项目和片段")
		}
	}
	r.SubmittedAt = time.Now().UTC()
	if r.CheckedAt.IsZero() {
		r.CheckedAt = r.SubmittedAt
	}
	r.RevisionStatus = "submitted"
	p.Revisions[r.RevisionID] = r
	p.touch()
	return nil
}
func (p *CorpusProject) SetReady() error {
	if !p.readyEligible() {
		return ErrInvalid
	}
	p.Status = StatusReady
	p.touch()
	return nil
}

func (p *CorpusProject) Review(revisionID, reviewerID string, decision ReviewDecision, note string) error {
	if p.Status == StatusFrozen {
		return InvalidField("projectID", "项目已冻结，不能再复核")
	}
	if strings.TrimSpace(reviewerID) == "" {
		return InvalidField("reviewerID", "复核专家不能为空")
	}
	r, ok := p.Revisions[revisionID]
	if !ok {
		return ErrNotFound
	}
	if decision != ReviewPass && decision != ReviewReturn && decision != ReviewNote {
		return InvalidField("decision", "复核决定必须为 pass、return 或 note")
	}
	if (decision == ReviewReturn || decision == ReviewNote) && strings.TrimSpace(note) == "" {
		return InvalidField("note", "return 和 note 决定必须填写意见")
	}
	if decision == ReviewPass && hasOpenIssues(r.CheckIssues) {
		return InvalidField("decision", "修订仍有未关闭阻断问题，不能通过")
	}
	now := time.Now().UTC()
	r.ReviewDecision, r.ReviewerID, r.ReviewNote, r.ReviewedAt, r.RevisionStatus = decision, reviewerID, strings.TrimSpace(note), &now, "reviewed"
	p.Revisions[revisionID] = r
	if decision == ReviewReturn {
		p.Status = StatusDraft
	} else if p.readyEligible() {
		p.Status = StatusReady
	}
	p.touch()
	return nil
}

func (p *CorpusProject) readyEligible() bool {
	heads := p.HeadRevisions()
	if len(heads) == 0 || len(p.Segments) == 0 {
		return false
	}
	coveredSegments := map[string]bool{}
	for _, r := range heads {
		if r.RevisionStatus != "reviewed" || (r.ReviewDecision != ReviewPass && r.ReviewDecision != ReviewNote) || hasOpenIssues(r.CheckIssues) {
			return false
		}
		coveredSegments[r.SegmentID] = true
	}
	return len(coveredSegments) == len(p.Segments)
}

func hasOpenIssues(issues []Issue) bool {
	for _, issue := range issues {
		if !issue.Closed {
			return true
		}
	}
	return false
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func (p *CorpusProject) Freeze() error {
	if p.Status != StatusReady {
		return ErrInvalid
	}
	if !p.readyEligible() {
		return InvalidField("projectID", "项目存在未复核修订或未关闭问题，不能冻结")
	}
	p.Status = StatusFrozen
	p.touch()
	return nil
}
func (p *CorpusProject) Revision(id string) (TranscriptRevision, bool) {
	r, ok := p.Revisions[id]
	return r, ok
}
