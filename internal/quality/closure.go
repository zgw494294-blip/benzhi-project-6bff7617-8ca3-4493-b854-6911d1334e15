package quality

import (
	"fieldlingua/internal/domain"
	"fmt"
	"strings"
	"time"
)

type ClosureEvidence struct {
	Code        string `json:"code"`
	StartMillis int64  `json:"startMillis"`
	EndMillis   int64  `json:"endMillis"`
	ClosureNote string `json:"closureNote"`
}

func issueKey(code string, start, end int64) string { return fmt.Sprintf("%s:%d:%d", code, start, end) }

func ReconcileIssues(previous, current []domain.Issue, evidence []ClosureEvidence) ([]domain.Issue, error) {
	notes := map[string]string{}
	for _, item := range evidence {
		notes[issueKey(item.Code, item.StartMillis, item.EndMillis)] = strings.TrimSpace(item.ClosureNote)
	}
	currentByKey := map[string]domain.Issue{}
	for _, issue := range current {
		currentByKey[issueKey(issue.Code, issue.StartMillis, issue.EndMillis)] = issue
	}
	result := make([]domain.Issue, 0, len(previous)+len(current))
	seen := map[string]bool{}
	now := time.Now().UTC()
	for _, old := range previous {
		key := issueKey(old.Code, old.StartMillis, old.EndMillis)
		if existing, ok := currentByKey[key]; ok {
			existing.Closed = false
			existing.ClosureNote = ""
			existing.ClosedAt = nil
			result = append(result, existing)
			seen[key] = true
			continue
		}
		if old.Closed {
			result = append(result, old)
			continue
		}
		note := notes[key]
		if note == "" {
			return nil, domain.InvalidField("closures", "已消失的基线问题必须逐项提供非空 closureNote")
		}
		old.Closed = true
		old.ClosureNote = note
		old.ClosedAt = &now
		result = append(result, old)
	}
	for _, issue := range current {
		if !seen[issueKey(issue.Code, issue.StartMillis, issue.EndMillis)] {
			result = append(result, issue)
		}
	}
	sortIssues(result)
	return result, nil
}

func CloseIssues(previous, current []domain.Issue, note string) []domain.Issue {
	evidence := make([]ClosureEvidence, 0, len(previous))
	for _, issue := range previous {
		evidence = append(evidence, ClosureEvidence{issue.Code, issue.StartMillis, issue.EndMillis, note})
	}
	result, err := ReconcileIssues(previous, current, evidence)
	if err != nil {
		return append([]domain.Issue(nil), previous...)
	}
	return result
}
func Blocking(issues []domain.Issue) bool {
	for _, issue := range issues {
		if !issue.Closed {
			return true
		}
	}
	return false
}

type IssueSummary struct {
	Code           string `json:"code"`
	Count          int    `json:"count"`
	OpenCount      int    `json:"openCount"`
	ClosedCount    int    `json:"closedCount"`
	EarliestMillis int64  `json:"earliestMillis"`
	LatestMillis   int64  `json:"latestMillis"`
}

func Summarize(issues []domain.Issue, code string) ([]domain.Issue, []IssueSummary, bool) {
	filtered := []domain.Issue{}
	byCode := map[string]*IssueSummary{}
	for _, issue := range issues {
		if code != "" && issue.Code != code {
			continue
		}
		filtered = append(filtered, issue)
		summary := byCode[issue.Code]
		if summary == nil {
			summary = &IssueSummary{Code: issue.Code, EarliestMillis: issue.StartMillis, LatestMillis: issue.EndMillis}
			byCode[issue.Code] = summary
		}
		summary.Count++
		if issue.Closed {
			summary.ClosedCount++
		} else {
			summary.OpenCount++
		}
		if issue.StartMillis < summary.EarliestMillis {
			summary.EarliestMillis = issue.StartMillis
		}
		if issue.EndMillis > summary.LatestMillis {
			summary.LatestMillis = issue.EndMillis
		}
	}
	sortIssues(filtered)
	summaries := make([]IssueSummary, 0, len(byCode))
	for _, value := range byCode {
		summaries = append(summaries, *value)
	}
	for i := 0; i < len(summaries); i++ {
		for j := i + 1; j < len(summaries); j++ {
			if summaries[j].Code < summaries[i].Code {
				summaries[i], summaries[j] = summaries[j], summaries[i]
			}
		}
	}
	return filtered, summaries, Blocking(issues)
}
