package application

import (
	"errors"
	"fieldlingua/internal/domain"
	"fieldlingua/internal/persistence"
	"fieldlingua/internal/quality"
	"sync"
	"testing"
)

func TestRemediationReleaseAndRestart(t *testing.T) {
	dir := t.TempDir()
	service := New(persistence.New(dir))
	created, err := service.Create(CreateProjectInput{ProjectID: "p", Title: "濒危语言访谈", LanguageVariant: "北部变体", OwnerID: "owner", CollectionSites: []string{"村落"}, EthicsStatus: domain.EthicsRestricted, ConsentRefs: []string{"consent-1"}, IdempotencyKey: "create-1"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Project.(*domain.CorpusProject).Version != 1 {
		t.Fatal("建档版本应为 1")
	}
	_, err = service.AddSegment(AddSegmentInput{ProjectID: "p", ExpectedVersion: 1, Segment: domain.RecordingSegment{SegmentID: "s1", ProjectID: "other", RecordingDigest: "digest", SpeakerID: "speaker", StartMillis: 0, EndMillis: 1000, ConsentRef: "consent-1"}})
	if err == nil {
		t.Fatal("项目归属不一致必须拒绝")
	}
	project, _ := service.Store.GetProject("p")
	if project.Version != 1 || len(project.Segments) != 0 {
		t.Fatal("失败请求改变了项目")
	}
	_, err = service.AddSegment(AddSegmentInput{ProjectID: "p", ExpectedVersion: 1, IdempotencyKey: "segment-1", Segment: domain.RecordingSegment{SegmentID: "s1", RecordingDigest: "digest", SpeakerID: "speaker", StartMillis: 0, EndMillis: 1000, ConsentRef: "consent-1", ContextNote: "原始语境"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.SubmitRevision(SubmitRevisionInput{ProjectID: "p", ExpectedVersion: 2, Revision: domain.TranscriptRevision{RevisionID: "r1", SegmentID: "s1", TranscriberID: "writer", ChangeNote: "初稿", TranscriptSegments: []domain.TranscriptPart{{StartMillis: 0, EndMillis: 500, SpeakerID: "speaker", Text: "上半段"}}}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.SubmitRevision(SubmitRevisionInput{ProjectID: "p", ExpectedVersion: 3, IdempotencyKey: "revision-2", Revision: domain.TranscriptRevision{RevisionID: "r2", SegmentID: "s1", BaseRevisionID: "r1", TranscriberID: "writer", ChangeNote: "补全缺口", TranscriptSegments: []domain.TranscriptPart{{StartMillis: 0, EndMillis: 1000, SpeakerID: "speaker", Text: "完整文本"}}}, Closures: []quality.ClosureEvidence{{Code: "coverage_gap", StartMillis: 500, EndMillis: 1000, ClosureNote: "依据录音补全尾部"}}})
	if err != nil {
		t.Fatal(err)
	}
	if *second.Blocking {
		t.Fatal("整改后不应继续阻断")
	}
	_, err = service.Review(ReviewInput{ProjectID: "p", RevisionID: "r2", ReviewerID: "expert", Decision: domain.ReviewPass, ExpectedVersion: 4, IdempotencyKey: "review-2"})
	if err != nil {
		t.Fatal(err)
	}
	released, err := service.ReleaseInput(ReleaseInput{ProjectID: "p", IssuedBy: "publisher", CredentialID: "credential-p", ExpectedVersion: 5, IdempotencyKey: "release-p"})
	if err != nil {
		t.Fatal(err)
	}
	if released.Credential.ProjectVersion != 6 || len(released.Credential.RevisionIDs) != 1 || released.Credential.RevisionIDs[0] != "r2" {
		t.Fatalf("冻结清单错误: %+v", released.Credential)
	}
	restarted := New(persistence.New(dir))
	verified, err := restarted.Verify("credential-p")
	if err != nil {
		t.Fatal(err)
	}
	if verified.Verification.VerificationStatus != "valid" {
		t.Fatal("重启后凭据应有效")
	}
	replay, err := restarted.Create(CreateProjectInput{ProjectID: "p", Title: "重复", LanguageVariant: "x", OwnerID: "x", CollectionSites: []string{"x"}, EthicsStatus: domain.EthicsApproved, IdempotencyKey: "create-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !replay.IdempotentReplay {
		t.Fatal("重启后应复用建档结果")
	}
}

func TestVersionConflictIsAtomic(t *testing.T) {
	service := New(persistence.New(t.TempDir()))
	_, err := service.Create(CreateProjectInput{ProjectID: "p", Title: "项目", LanguageVariant: "变体", OwnerID: "owner", CollectionSites: []string{"地点"}, EthicsStatus: domain.EthicsApproved})
	if err != nil {
		t.Fatal(err)
	}
	inputs := []AddSegmentInput{{ProjectID: "p", ExpectedVersion: 1, Segment: domain.RecordingSegment{SegmentID: "a", RecordingDigest: "a", SpeakerID: "s", StartMillis: 0, EndMillis: 10, ConsentRef: "c"}}, {ProjectID: "p", ExpectedVersion: 1, Segment: domain.RecordingSegment{SegmentID: "b", RecordingDigest: "b", SpeakerID: "s", StartMillis: 0, EndMillis: 10, ConsentRef: "c"}}}
	var wait sync.WaitGroup
	wait.Add(2)
	errs := make(chan error, 2)
	for _, input := range inputs {
		go func(value AddSegmentInput) { defer wait.Done(); _, e := service.AddSegment(value); errs <- e }(input)
	}
	wait.Wait()
	close(errs)
	success, conflicts := 0, 0
	for e := range errs {
		if e == nil {
			success++
		} else if errors.Is(e, domain.ErrVersionConflict) {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("成功=%d 冲突=%d", success, conflicts)
	}
	project, _ := service.Store.GetProject("p")
	if project.Version != 2 || len(project.Segments) != 1 {
		t.Fatal("并发冲突后快照不一致")
	}
}
