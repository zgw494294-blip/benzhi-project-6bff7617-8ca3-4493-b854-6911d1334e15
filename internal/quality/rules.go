package quality

import (
	"fieldlingua/internal/domain"
	"sort"
	"strings"
)

// Check 对同一输入始终生成相同顺序的问题清单。
func Check(parts []domain.TranscriptPart, seg domain.RecordingSegment) []domain.Issue {
	ordered := append([]domain.TranscriptPart(nil), parts...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].StartMillis != ordered[j].StartMillis {
			return ordered[i].StartMillis < ordered[j].StartMillis
		}
		if ordered[i].EndMillis != ordered[j].EndMillis {
			return ordered[i].EndMillis < ordered[j].EndMillis
		}
		if ordered[i].SpeakerID != ordered[j].SpeakerID {
			return ordered[i].SpeakerID < ordered[j].SpeakerID
		}
		return ordered[i].Text < ordered[j].Text
	})
	issues := []domain.Issue{}
	if len(ordered) == 0 {
		return []domain.Issue{{Code: "missing_text", Message: "缺少转写分段", StartMillis: seg.StartMillis, EndMillis: seg.EndMillis}}
	}
	covered := seg.StartMillis
	for _, part := range ordered {
		if part.StartMillis < seg.StartMillis || part.EndMillis > seg.EndMillis || part.EndMillis <= part.StartMillis {
			issues = append(issues, domain.Issue{Code: "time_range", Message: "时间码超出录音片段范围或不是正向区间", StartMillis: part.StartMillis, EndMillis: part.EndMillis})
		}
		if part.StartMillis > covered {
			end := part.StartMillis
			if end > seg.EndMillis {
				end = seg.EndMillis
			}
			if covered < end {
				issues = append(issues, domain.Issue{Code: "coverage_gap", Message: "时间码覆盖存在缺口", StartMillis: covered, EndMillis: end})
			}
		} else if part.StartMillis < covered && part.EndMillis > seg.StartMillis {
			end := covered
			if part.EndMillis < end {
				end = part.EndMillis
			}
			if part.StartMillis < end {
				issues = append(issues, domain.Issue{Code: "overlap", Message: "相邻分段存在重叠", StartMillis: part.StartMillis, EndMillis: end})
			}
		}
		if part.SpeakerID == "" {
			issues = append(issues, domain.Issue{Code: "speaker_missing", Message: "缺少说话人引用", StartMillis: part.StartMillis, EndMillis: part.EndMillis})
		} else if seg.SpeakerID != "" && part.SpeakerID != seg.SpeakerID {
			issues = append(issues, domain.Issue{Code: "speaker_invalid", Message: "说话人引用与录音片段不一致", StartMillis: part.StartMillis, EndMillis: part.EndMillis})
		}
		if strings.TrimSpace(part.Text) == "" {
			issues = append(issues, domain.Issue{Code: "text_missing", Message: "缺少必填转写文本", StartMillis: part.StartMillis, EndMillis: part.EndMillis})
		}
		if part.EndMillis > covered {
			covered = part.EndMillis
		}
	}
	if covered < seg.EndMillis {
		issues = append(issues, domain.Issue{Code: "coverage_gap", Message: "时间码未覆盖片段尾部", StartMillis: covered, EndMillis: seg.EndMillis})
	}
	sortIssues(issues)
	return issues
}

func sortIssues(issues []domain.Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].StartMillis != issues[j].StartMillis {
			return issues[i].StartMillis < issues[j].StartMillis
		}
		if issues[i].EndMillis != issues[j].EndMillis {
			return issues[i].EndMillis < issues[j].EndMillis
		}
		if issues[i].Code != issues[j].Code {
			return issues[i].Code < issues[j].Code
		}
		return issues[i].Message < issues[j].Message
	})
}
