package postgres

import (
	"fmt"
	"strings"
	"time"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

const (
	dateLayout     = "2006-01-02"
	dateTimeLayout = "2006-01-02 15:04:05"
)

func parseDate(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.ParseInLocation(dateLayout, value, time.Local)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseDateTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.ParseInLocation(dateTimeLayout, value, time.Local)
	if err != nil {
		if date := parseDate(value); date != nil {
			return date
		}
		return nil
	}
	return &parsed
}

func formatDate(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Local().Format(dateLayout)
}

func formatDateTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Local().Format(dateTimeLayout)
}

func normalizeProblem(item *models.ErrorProblem) {
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.ReasonTags == nil {
		item.ReasonTags = []string{}
	}
	if item.NextReview == "" {
		item.NextReview = time.Now().Format(dateLayout)
	}
}

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

func listContainsFold(list []string, keyword string) bool {
	keyword = strings.ToLower(keyword)
	for _, item := range list {
		if strings.Contains(strings.ToLower(item), keyword) {
			return true
		}
	}
	return false
}

func notFound(kind string, id int) error {
	return fmt.Errorf("未找到%s #%d", kind, id)
}
