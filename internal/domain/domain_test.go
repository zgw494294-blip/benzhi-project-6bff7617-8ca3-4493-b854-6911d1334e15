package domain

import "testing"

func TestProjectFlow(t *testing.T) {
	p, e := NewProject("p", "标题", "变体", "o", []string{"地点"}, EthicsApproved)
	if e != nil {
		t.Fatal(e)
	}
	if e = p.AddSegment(RecordingSegment{SegmentID: "s", ProjectID: "p", RecordingDigest: "d", SpeakerID: "sp", StartMillis: 0, EndMillis: 10, ConsentRef: "c"}); e != nil {
		t.Fatal(e)
	}
	if e = p.AddRevision(TranscriptRevision{RevisionID: "r", ProjectID: "p", SegmentID: "s", TranscriberID: "t", ChangeNote: "初稿"}); e != nil {
		t.Fatal(e)
	}
	if e = p.Validate(); e != nil {
		t.Fatal(e)
	}
}

func TestValidateRejectsBrokenRevisionChain(t *testing.T) {
	p, err := NewProject("p", "标题", "变体", "o", []string{"地点"}, EthicsApproved)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.AddSegment(RecordingSegment{SegmentID: "s", ProjectID: "p", RecordingDigest: "d", SpeakerID: "sp", StartMillis: 0, EndMillis: 10, ConsentRef: "c"}); err != nil {
		t.Fatal(err)
	}
	if err = p.AddRevision(TranscriptRevision{RevisionID: "r", ProjectID: "p", SegmentID: "s", TranscriberID: "t", ChangeNote: "初稿"}); err != nil {
		t.Fatal(err)
	}
	revision := p.Revisions["r"]
	revision.BaseRevisionID = "r"
	p.Revisions["r"] = revision
	if err = p.Validate(); err == nil {
		t.Fatal("循环修订基线必须被拒绝")
	}
}
