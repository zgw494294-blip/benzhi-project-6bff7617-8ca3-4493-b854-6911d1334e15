package quality

import (
	"fieldlingua/internal/domain"
	"testing"
)

func TestCheck(t *testing.T) {
	x := Check([]domain.TranscriptPart{{StartMillis: 0, EndMillis: 5, SpeakerID: "sp", Text: "x"}}, domain.RecordingSegment{StartMillis: 0, EndMillis: 10})
	if len(x) != 1 || x[0].Code != "coverage_gap" {
		t.Fatalf("%v", x)
	}
}
