package quality

import "fieldlingua/internal/domain"

type Checker struct{}

func (Checker) Run(r domain.TranscriptRevision, s domain.RecordingSegment) []domain.Issue {
	return Check(r.TranscriptSegments, s)
}
