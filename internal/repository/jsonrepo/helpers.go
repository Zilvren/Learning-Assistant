package jsonrepo

import (
	"strings"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

// normalizeReviewFields 在存储层中构造、编码或标准化数据。
func normalizeReviewFields(item *models.ErrorProblem) {
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.ReasonTags == nil {
		item.ReasonTags = []string{}
	}
}

// applyErrorUpdate 在存储层中执行流程或启动外部操作。
func applyErrorUpdate(item *models.ErrorProblem, req models.UpdateErrorRequest) {
	if req.Subject != nil {
		item.Subject = *req.Subject
	}
	if req.Title != nil {
		item.Title = *req.Title
	}
	if req.Question != nil {
		item.Question = *req.Question
	}
	if req.Wrong != nil {
		item.Wrong = *req.Wrong
	}
	if req.Correct != nil {
		item.Correct = *req.Correct
	}
	if req.Reason != nil {
		item.Reason = *req.Reason
	}
	if req.Tags != nil {
		item.Tags = *req.Tags
	}
	if req.ReasonTags != nil {
		item.ReasonTags = *req.ReasonTags
	}
}

// matchesFilter 在存储层中完成本文件定义的局部处理。
func matchesFilter(item models.ErrorProblem, filter base.ErrorFilter) bool {
	if filter.Subject != "" && filter.Subject != "全部" && item.Subject != filter.Subject {
		return false
	}
	if filter.Keyword != "" && !matchesKeyword(item, filter.Keyword) {
		return false
	}
	if filter.Tag != "" && !listContainsFold(item.Tags, filter.Tag) {
		return false
	}
	if filter.ReasonTag != "" && !listContainsFold(item.ReasonTags, filter.ReasonTag) {
		return false
	}
	return true
}

// matchesKeyword 在存储层中完成本文件定义的局部处理。
func matchesKeyword(item models.ErrorProblem, keyword string) bool {
	keyword = strings.ToLower(keyword)
	if strings.Contains(strings.ToLower(item.Question), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Title), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Reason), keyword) {
		return true
	}
	return listContainsFold(item.Tags, keyword) || listContainsFold(item.ReasonTags, keyword)
}

// listContainsFold 在存储层中读取并整理所需数据。
func listContainsFold(list []string, keyword string) bool {
	keyword = strings.ToLower(keyword)
	for _, item := range list {
		if strings.Contains(strings.ToLower(item), keyword) {
			return true
		}
	}
	return false
}
