package crypto

import (
	"fieldlingua/internal/domain"
	"testing"
)

func TestCredential(t *testing.T) {
	p, _ := domain.NewProject("p", "t", "v", "o", []string{"s"}, domain.EthicsApproved)
	p.AddSegment(domain.RecordingSegment{SegmentID: "s", ProjectID: "p", RecordingDigest: "d", SpeakerID: "sp", StartMillis: 0, EndMillis: 1, ConsentRef: "c"})
	p.AddRevision(domain.TranscriptRevision{RevisionID: "r", ProjectID: "p", SegmentID: "s", TranscriberID: "t", ChangeNote: "x"})
	c, e := Issue(p, "c", "a")
	if e != nil || Verify(c, p) != nil {
		t.Fatal(e)
	}
}
