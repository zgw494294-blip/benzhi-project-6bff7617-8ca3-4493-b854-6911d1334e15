package domain

import (
	"encoding/json"
	"sort"
)

func (p *CorpusProject) NormalizedRevisions() []TranscriptRevision {
	out := make([]TranscriptRevision, 0, len(p.Revisions))
	for _, r := range p.Revisions {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RevisionID < out[j].RevisionID })
	return out
}

// HeadRevisions 返回每个修订链尚未被后续版本替代的当前版本。
func (p *CorpusProject) HeadRevisions() []TranscriptRevision {
	referenced := map[string]bool{}
	for _, r := range p.Revisions {
		if r.BaseRevisionID != "" {
			referenced[r.BaseRevisionID] = true
		}
	}
	out := []TranscriptRevision{}
	for id, r := range p.Revisions {
		if !referenced[id] {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RevisionID < out[j].RevisionID })
	return out
}
func (p *CorpusProject) Canonical() ([]byte, error) {
	type canonicalSegment struct {
		SegmentID, RecordingDigest, SpeakerID string
		StartMillis, EndMillis                int64
		ContextNote, ConsentRef               string
	}
	type canonicalRevision struct {
		RevisionID, SegmentID, BaseRevisionID string
		TranscriptSegments                    []TranscriptPart
	}
	type c struct {
		ProjectID       string              `json:"projectID"`
		Version         int                 `json:"version"`
		Title           string              `json:"title"`
		LanguageVariant string              `json:"languageVariant"`
		Segments        []canonicalSegment  `json:"segments"`
		Revisions       []canonicalRevision `json:"revisions"`
	}
	segments := make([]canonicalSegment, 0, len(p.Segments))
	for _, s := range p.Segments {
		segments = append(segments, canonicalSegment{s.SegmentID, s.RecordingDigest, s.SpeakerID, s.StartMillis, s.EndMillis, s.ContextNote, s.ConsentRef})
	}
	sort.Slice(segments, func(i, j int) bool { return segments[i].SegmentID < segments[j].SegmentID })
	revisions := make([]canonicalRevision, 0, len(p.Revisions))
	for _, r := range p.HeadRevisions() {
		parts := append([]TranscriptPart(nil), r.TranscriptSegments...)
		sort.SliceStable(parts, func(i, j int) bool {
			if parts[i].StartMillis != parts[j].StartMillis {
				return parts[i].StartMillis < parts[j].StartMillis
			}
			if parts[i].EndMillis != parts[j].EndMillis {
				return parts[i].EndMillis < parts[j].EndMillis
			}
			if parts[i].SpeakerID != parts[j].SpeakerID {
				return parts[i].SpeakerID < parts[j].SpeakerID
			}
			return parts[i].Text < parts[j].Text
		})
		revisions = append(revisions, canonicalRevision{r.RevisionID, r.SegmentID, r.BaseRevisionID, parts})
	}
	return json.Marshal(c{p.ProjectID, p.Version, p.Title, p.LanguageVariant, segments, revisions})
}
