package credential_slice_alias

import (
	"fieldlingua/internal/crypto"
	"fieldlingua/internal/domain"
	"fieldlingua/internal/persistence"
	"testing"
)

func TestCredentialReadDoesNotExposeMutableSlice(t *testing.T) {
	project, err := domain.NewProject("p-credential-alias", "凭据项目", "方言", "owner", []string{"地点"}, domain.EthicsApproved)
	if err != nil {
		t.Fatal(err)
	}
	if err = project.AddSegment(domain.RecordingSegment{SegmentID: "s1", ProjectID: project.ProjectID, RecordingDigest: "digest", SpeakerID: "speaker", StartMillis: 0, EndMillis: 1000, ConsentRef: "consent"}); err != nil {
		t.Fatal(err)
	}
	if err = project.AddRevision(domain.TranscriptRevision{RevisionID: "r1", ProjectID: project.ProjectID, SegmentID: "s1", TranscriberID: "writer", ChangeNote: "初稿"}); err != nil {
		t.Fatal(err)
	}
	credential, err := crypto.Issue(project, "cred-alias", "publisher")
	if err != nil {
		t.Fatal(err)
	}
	store := persistence.New(t.TempDir())
	store.PutProject(project)
	store.PutCredential(credential)
	read, err := store.GetCredential(credential.CredentialID)
	if err != nil {
		t.Fatal(err)
	}
	originalRevisionID := read.RevisionIDs[0]
	read.RevisionIDs[0] = "tampered"
	again, err := store.GetCredential(credential.CredentialID)
	if err != nil {
		t.Fatal(err)
	}
	if again.RevisionIDs[0] != originalRevisionID {
		t.Errorf("读取凭据的切片修改污染了存储，得到 revisionID=%q", again.RevisionIDs[0])
	}
	credential.RevisionIDs[0] = originalRevisionID
	store.PutCredential(credential)
	listed := store.CredentialsForProject(project.ProjectID)
	if len(listed) != 1 {
		t.Fatalf("项目凭据数量错误：%d", len(listed))
	}
	listed[0].RevisionIDs[0] = "tampered-list"
	again, err = store.GetCredential(credential.CredentialID)
	if err != nil {
		t.Fatal(err)
	}
	if again.RevisionIDs[0] != originalRevisionID {
		t.Errorf("项目凭据列表的切片修改污染了存储，得到 revisionID=%q", again.RevisionIDs[0])
	}
	if err := crypto.Verify(again, project); err != nil {
		t.Errorf("读取凭据被外部切片修改后不应导致核验失败：%v", err)
	}
}
