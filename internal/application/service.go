package application

import (
	"fieldlingua/internal/crypto"
	"fieldlingua/internal/domain"
	"fieldlingua/internal/persistence"
	"fieldlingua/internal/quality"
	"sort"
	"strings"
)

type Service struct {
	Store   *persistence.Store
	Checker quality.Checker
}

func New(store *persistence.Store) *Service { return &Service{Store: store} }

func (a *Service) Create(in CreateProjectInput) (Result, error) {
	if in.ExpectedVersion != 0 {
		return Result{}, domain.InvalidField("expectedVersion", "新项目的 expectedVersion 必须为 0")
	}
	p, err := domain.NewProjectWithConsent(in.ProjectID, in.Title, in.LanguageVariant, in.OwnerID, in.CollectionSites, in.EthicsStatus, in.ConsentRefs)
	if err != nil {
		return Result{}, err
	}
	p, reused, err := a.Store.CreateProject(p, "create-project", strings.TrimSpace(in.IdempotencyKey))
	if err != nil {
		return Result{}, err
	}
	return Result{Project: p, Message: "项目已创建", IdempotentReplay: reused}, nil
}
func (a *Service) AddSegment(in AddSegmentInput) (Result, error) {
	if in.Segment.ProjectID == "" {
		in.Segment.ProjectID = in.ProjectID
	}
	p, reused, err := a.Store.ApplyProjectCommand(in.ProjectID, in.ExpectedVersion, "add-segment:"+in.Segment.SegmentID, strings.TrimSpace(in.IdempotencyKey), func(project *domain.CorpusProject) error { return project.AddSegment(in.Segment) })
	if err != nil {
		return Result{}, err
	}
	return Result{Project: p, Message: "录音片段已登记", IdempotentReplay: reused}, nil
}
func (a *Service) SubmitRevision(in SubmitRevisionInput) (Result, error) {
	if in.Revision.ProjectID == "" {
		in.Revision.ProjectID = in.ProjectID
	}
	p, reused, err := a.Store.ApplyProjectCommand(in.ProjectID, in.ExpectedVersion, "submit-revision:"+in.Revision.RevisionID, strings.TrimSpace(in.IdempotencyKey), func(project *domain.CorpusProject) error {
		segment, ok := project.Segments[in.Revision.SegmentID]
		if !ok {
			return domain.InvalidField("revision.segmentID", "转写片段不存在")
		}
		revision := in.Revision
		current := a.Checker.Run(revision, segment)
		if revision.BaseRevisionID != "" {
			base, ok := project.Revisions[revision.BaseRevisionID]
			if !ok {
				return domain.InvalidField("revision.baseRevisionID", "基线修订不存在")
			}
			if base.ProjectID != project.ProjectID || base.SegmentID != revision.SegmentID {
				return domain.InvalidField("revision.baseRevisionID", "基线修订不属于同一项目和片段")
			}
			reconciled, e := quality.ReconcileIssues(base.CheckIssues, current, in.Closures)
			if e != nil {
				return e
			}
			revision.CheckIssues = reconciled
		} else {
			revision.CheckIssues = current
		}
		return project.AddRevision(revision)
	})
	if err != nil {
		return Result{}, err
	}
	revision := p.Revisions[in.Revision.RevisionID]
	issues, summaries, blocking := quality.Summarize(revision.CheckIssues, "")
	status := "无阻断，可进入专家复核"
	if blocking {
		status = "存在未关闭阻断问题"
	}
	return Result{Project: p, Issues: issues, IssueSummary: summaries, Blocking: &blocking, Message: status, IdempotentReplay: reused}, nil
}
func (a *Service) Review(in ReviewInput) (Result, error) {
	p, reused, err := a.Store.ApplyProjectCommand(in.ProjectID, in.ExpectedVersion, "review-revision:"+in.RevisionID, strings.TrimSpace(in.IdempotencyKey), func(project *domain.CorpusProject) error {
		return project.Review(in.RevisionID, in.ReviewerID, in.Decision, in.Note)
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Project: p, Message: "复核结论已记录：" + string(in.Decision), IdempotentReplay: reused}, nil
}
func (a *Service) ReleaseInput(in ReleaseInput) (Result, error) {
	p, c, reused, err := a.Store.ReleaseProject(in.ProjectID, in.ExpectedVersion, "release-project:"+in.CredentialID, strings.TrimSpace(in.IdempotencyKey), in.CredentialID, func(project *domain.CorpusProject) (crypto.Credential, error) {
		return crypto.Issue(project, in.CredentialID, in.IssuedBy)
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Project: p, Credential: &c, Message: "项目已冻结并签发发布凭据", IdempotentReplay: reused}, nil
}
func (a *Service) Release(projectID, by, credentialID string) (Result, error) {
	return a.ReleaseInput(ReleaseInput{ProjectID: projectID, IssuedBy: by, CredentialID: credentialID})
}

func (a *Service) Verify(id string) (Result, error) { return a.VerifyCredential(id, nil) }
func (a *Service) VerifyCredential(id string, candidate *crypto.Credential) (Result, error) {
	stored, err := a.Store.GetCredential(strings.TrimSpace(id))
	if err != nil {
		return Result{Verification: &VerificationResult{VerificationStatus: "not_found", Reason: "credentialID 不存在"}}, err
	}
	check := stored
	if candidate != nil {
		check = *candidate
		if check.CredentialID == "" {
			check.CredentialID = id
		}
		if check.CredentialID != stored.CredentialID || check.ProjectID != stored.ProjectID {
			return Result{Verification: &VerificationResult{VerificationStatus: "invalid", Reason: "提交凭据与已持久化凭据不匹配"}}, domain.ErrInvalid
		}
	}
	if err = crypto.ValidateFormat(check); err != nil {
		return Result{Verification: &VerificationResult{VerificationStatus: "invalid_format", Reason: err.Error(), Credential: &check}}, err
	}
	p, err := a.Store.GetProject(stored.ProjectID)
	if err != nil {
		return Result{Verification: &VerificationResult{VerificationStatus: "project_not_found", Reason: "凭据关联项目不存在", Credential: &stored}}, err
	}
	if err = crypto.Verify(check, p); err != nil {
		return Result{Verification: &VerificationResult{VerificationStatus: "invalid", Reason: err.Error(), Credential: &check}}, err
	}
	verification := &VerificationResult{VerificationStatus: "valid", Credential: &stored, IssuedBy: stored.IssuedBy, IssuedAt: stored.IssuedAt, ProjectVersion: stored.ProjectVersion}
	return Result{Credential: &stored, Verification: verification, Message: "凭据验证通过"}, nil
}

func projectSummary(project *domain.CorpusProject) ProjectSummary {
	open := 0
	for _, revision := range project.HeadRevisions() {
		for _, issue := range revision.CheckIssues {
			if !issue.Closed {
				open++
			}
		}
	}
	return ProjectSummary{ProjectID: project.ProjectID, Title: project.Title, Status: project.Status, EthicsStatus: project.EthicsStatus, OwnerID: project.OwnerID, Version: project.Version, SegmentCount: len(project.Segments), RevisionCount: len(project.Revisions), OpenIssueCount: open, UpdatedAt: project.UpdatedAt}
}
func (a *Service) ListProjects(filter ProjectFilter) []ProjectSummary {
	out := []ProjectSummary{}
	for _, project := range a.Store.ListProjects() {
		if filter.Status != "" && project.Status != filter.Status {
			continue
		}
		if filter.EthicsStatus != "" && project.EthicsStatus != filter.EthicsStatus {
			continue
		}
		if filter.OwnerID != "" && project.OwnerID != filter.OwnerID {
			continue
		}
		out = append(out, projectSummary(project))
	}
	return out
}
func (a *Service) ProjectDetail(id string) (ProjectDetail, error) {
	project, err := a.Store.GetProject(id)
	if err != nil {
		return ProjectDetail{}, err
	}
	segments := make([]domain.RecordingSegment, 0, len(project.Segments))
	for _, segment := range project.Segments {
		segments = append(segments, segment)
	}
	sort.Slice(segments, func(i, j int) bool {
		if segments[i].UpdatedAt.Equal(segments[j].UpdatedAt) {
			return segments[i].SegmentID < segments[j].SegmentID
		}
		return segments[i].UpdatedAt.After(segments[j].UpdatedAt)
	})
	revisions := make([]domain.TranscriptRevision, 0, len(project.Revisions))
	for _, revision := range project.Revisions {
		revisions = append(revisions, revision)
	}
	sort.Slice(revisions, func(i, j int) bool {
		if revisions[i].SubmittedAt.Equal(revisions[j].SubmittedAt) {
			return revisions[i].RevisionID > revisions[j].RevisionID
		}
		return revisions[i].SubmittedAt.After(revisions[j].SubmittedAt)
	})
	return ProjectDetail{Summary: projectSummary(project), Project: project, Segments: segments, Revisions: revisions, Credentials: a.Store.CredentialsForProject(id)}, nil
}
func (a *Service) RevisionHistory(projectID, segmentID string) ([]RevisionHistoryItem, error) {
	project, err := a.Store.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	out := []RevisionHistoryItem{}
	for _, revision := range project.Revisions {
		if revision.SegmentID != segmentID {
			continue
		}
		open := 0
		for _, issue := range revision.CheckIssues {
			if !issue.Closed {
				open++
			}
		}
		out = append(out, RevisionHistoryItem{Revision: revision, BaseRevisionID: revision.BaseRevisionID, IssueCount: len(revision.CheckIssues), OpenIssueCount: open, ReviewDecision: revision.ReviewDecision})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Revision.SubmittedAt.Equal(out[j].Revision.SubmittedAt) {
			return out[i].Revision.RevisionID < out[j].Revision.RevisionID
		}
		return out[i].Revision.SubmittedAt.Before(out[j].Revision.SubmittedAt)
	})
	return out, nil
}
func (a *Service) Issues(projectID, revisionID, code string) (IssueReport, error) {
	project, err := a.Store.GetProject(projectID)
	if err != nil {
		return IssueReport{}, err
	}
	revision, ok := project.Revisions[revisionID]
	if !ok {
		return IssueReport{}, domain.ErrNotFound
	}
	issues, summaries, blocking := quality.Summarize(revision.CheckIssues, code)
	status := "clear"
	if blocking {
		status = "blocked"
	}
	return IssueReport{RevisionID: revisionID, Issues: issues, Summaries: summaries, Blocking: blocking, Status: status}, nil
}
func (a *Service) ReviewQueue(projectID string) ([]ReviewQueueItem, error) {
	projects := a.Store.ListProjects()
	if projectID != "" {
		project, err := a.Store.GetProject(projectID)
		if err != nil {
			return nil, err
		}
		projects = []*domain.CorpusProject{project}
	}
	out := []ReviewQueueItem{}
	for _, project := range projects {
		if project.Status == domain.StatusFrozen {
			continue
		}
		for _, revision := range project.HeadRevisions() {
			if revision.CheckedAt.IsZero() || revision.RevisionStatus == "reviewed" {
				continue
			}
			segment := project.Segments[revision.SegmentID]
			open := 0
			for _, issue := range revision.CheckIssues {
				if !issue.Closed {
					open++
				}
			}
			out = append(out, ReviewQueueItem{ProjectID: project.ProjectID, ProjectTitle: project.Title, ProjectVersion: project.Version, RevisionID: revision.RevisionID, SegmentID: revision.SegmentID, SpeakerID: segment.SpeakerID, StartMillis: segment.StartMillis, EndMillis: segment.EndMillis, BaseRevisionID: revision.BaseRevisionID, OpenIssueCount: open, SubmittedAt: revision.SubmittedAt})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SubmittedAt.Equal(out[j].SubmittedAt) {
			return out[i].RevisionID < out[j].RevisionID
		}
		return out[i].SubmittedAt.Before(out[j].SubmittedAt)
	})
	return out, nil
}
