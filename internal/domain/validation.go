package domain

import (
	"fmt"
	"strings"
)

// Validate 校验从持久化层恢复的完整聚合，防止损坏的关联或修订链继续参与业务命令。
// 转写内容本身的问题由 quality 包记录为可整改问题，因此这里仅校验结构不变量。
func (p *CorpusProject) Validate() error {
	if p == nil {
		return InvalidField("project", "项目不能为空")
	}
	fields := map[string]string{}
	if strings.TrimSpace(p.ProjectID) == "" {
		fields["projectID"] = "项目 ID 不能为空"
	}
	if strings.TrimSpace(p.Title) == "" {
		fields["title"] = "项目标题不能为空"
	}
	if strings.TrimSpace(p.LanguageVariant) == "" {
		fields["languageVariant"] = "语言变体不能为空"
	}
	if strings.TrimSpace(p.OwnerID) == "" {
		fields["ownerID"] = "负责人不能为空"
	}
	if len(p.CollectionSites) == 0 {
		fields["collectionSites"] = "至少登记一个采集地点"
	}
	if p.EthicsStatus != EthicsPending && p.EthicsStatus != EthicsApproved && p.EthicsStatus != EthicsRestricted {
		fields["ethicsStatus"] = "伦理状态无效"
	}
	if p.Status != StatusDraft && p.Status != StatusReady && p.Status != StatusFrozen {
		fields["status"] = "项目状态无效"
	}
	if p.Version < 1 {
		fields["version"] = "项目版本必须大于 0"
	}
	if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() || p.UpdatedAt.Before(p.CreatedAt) {
		fields["updatedAt"] = "项目时间信息无效"
	}
	if err := validateUniqueText(p.CollectionSites, "collectionSites"); err != nil {
		fields["collectionSites"] = err.Error()
	}
	if err := validateUniqueText(p.ConsentRefs, "consentRefs"); err != nil {
		fields["consentRefs"] = err.Error()
	}
	if len(fields) > 0 {
		return &ValidationError{Message: "项目基础信息无效", Fields: fields}
	}

	for key, segment := range p.Segments {
		if err := p.validateSegment(key, segment); err != nil {
			return err
		}
	}
	for key, revision := range p.Revisions {
		if err := p.validateRevision(key, revision); err != nil {
			return err
		}
	}
	if err := p.validateRevisionChains(); err != nil {
		return err
	}
	if (p.Status == StatusReady || p.Status == StatusFrozen) && !p.readyEligible() {
		return InvalidField("status", "可冻结或已冻结项目必须具有覆盖全部片段的有效复核结论")
	}
	return nil
}

func validateUniqueText(values []string, field string) error {
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("%s 不能包含空值", field)
		}
		if _, ok := seen[value]; ok {
			return fmt.Errorf("%s 不能包含重复值", field)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func (p *CorpusProject) validateSegment(key string, segment RecordingSegment) error {
	if key == "" || key != segment.SegmentID {
		return InvalidField("segments", "片段索引键与 segmentID 不一致")
	}
	if segment.ProjectID != p.ProjectID {
		return InvalidField("segments", "片段归属与项目不一致")
	}
	if strings.TrimSpace(segment.RecordingDigest) == "" || strings.TrimSpace(segment.SpeakerID) == "" {
		return InvalidField("segments", "片段缺少录音摘要或说话人")
	}
	if strings.TrimSpace(segment.ConsentRef) == "" {
		return InvalidField("segments", "片段缺少同意凭据")
	}
	if segment.StartMillis < 0 || segment.EndMillis <= segment.StartMillis {
		return InvalidField("segments", "片段时间码范围无效")
	}
	if segment.MetadataStatus != "valid" {
		return InvalidField("segments", "片段元数据状态无效")
	}
	if p.EthicsStatus == EthicsRestricted && !contains(p.ConsentRefs, segment.ConsentRef) {
		return InvalidField("segments", "受限伦理项目包含未登记的同意凭据")
	}
	return nil
}

func (p *CorpusProject) validateRevision(key string, revision TranscriptRevision) error {
	if key == "" || key != revision.RevisionID {
		return InvalidField("revisions", "修订索引键与 revisionID 不一致")
	}
	if revision.ProjectID != p.ProjectID {
		return InvalidField("revisions", "修订归属与项目不一致")
	}
	if _, ok := p.Segments[revision.SegmentID]; !ok {
		return InvalidField("revisions", "修订引用的录音片段不存在")
	}
	if strings.TrimSpace(revision.TranscriberID) == "" || strings.TrimSpace(revision.ChangeNote) == "" {
		return InvalidField("revisions", "修订缺少转写员或变更说明")
	}
	if revision.SubmittedAt.IsZero() || revision.CheckedAt.IsZero() {
		return InvalidField("revisions", "修订缺少提交或检查时间")
	}
	if revision.RevisionStatus != "submitted" && revision.RevisionStatus != "reviewed" {
		return InvalidField("revisions", "修订状态无效")
	}
	if revision.RevisionStatus == "reviewed" {
		if revision.ReviewedAt == nil || strings.TrimSpace(revision.ReviewerID) == "" {
			return InvalidField("revisions", "已复核修订缺少专家或复核时间")
		}
		if revision.ReviewDecision != ReviewPass && revision.ReviewDecision != ReviewReturn && revision.ReviewDecision != ReviewNote {
			return InvalidField("revisions", "已复核修订的决定无效")
		}
	}
	return nil
}

func (p *CorpusProject) validateRevisionChains() error {
	for id, revision := range p.Revisions {
		seen := map[string]bool{id: true}
		current := revision
		for current.BaseRevisionID != "" {
			base, ok := p.Revisions[current.BaseRevisionID]
			if !ok {
				return InvalidField("revisions", "修订引用的基线不存在")
			}
			if base.ProjectID != current.ProjectID || base.SegmentID != current.SegmentID {
				return InvalidField("revisions", "修订基线不属于同一项目和片段")
			}
			if seen[base.RevisionID] {
				return InvalidField("revisions", "修订基线形成循环引用")
			}
			seen[base.RevisionID] = true
			current = base
		}
	}
	return nil
}
