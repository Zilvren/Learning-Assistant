package service

import (
	"context"
	"sort"
	"strings"
	"time"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/repository"
)

// ReviewInbox 汇总笔记和错题复习，使学习者无需记住到期项目最初来自哪个功能。
func ReviewInbox(ctx context.Context) ([]models.ReviewInboxItem, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	today := time.Now()
	day := today.Format(time.DateOnly)
	notes, err := repos.Library.DueReviews(ctx, today)
	if err != nil {
		return nil, err
	}
	items := make([]models.ReviewInboxItem, 0, len(notes))
	for _, note := range notes {
		items = append(items, models.ReviewInboxItem{SourceType: "library", ID: note.ID, Title: note.Name, Tags: note.Tags, NextReview: note.NextReview, ReviewStage: note.ReviewStage, OverdueDays: overdueDays(note.NextReview, today), Preview: "复习笔记"})
	}
	errors, err := repos.Errors.List(ctx, emptyErrorFilter())
	if err != nil {
		return nil, err
	}
	for _, problem := range errors {
		next := strings.TrimSpace(problem.NextReview)
		if next != "" && next > day {
			continue
		}
		tags := append([]string{}, problem.Tags...)
		tags = append(tags, problem.ReasonTags...)
		items = append(items, models.ReviewInboxItem{SourceType: "error", ID: int64(problem.ID), Title: problem.Title, Subject: problem.Subject, Tags: tags, NextReview: next, ReviewStage: problem.ReviewStage, OverdueDays: overdueDays(next, today), Preview: previewText(problem.Question)})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].OverdueDays != items[j].OverdueDays {
			return items[i].OverdueDays > items[j].OverdueDays
		}
		if items[i].NextReview != items[j].NextReview {
			return items[i].NextReview < items[j].NextReview
		}
		if items[i].SourceType != items[j].SourceType {
			return items[i].SourceType == "error"
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}

// emptyErrorFilter 在业务层中执行当前流程或局部处理。
func emptyErrorFilter() repository.ErrorFilter { return repository.ErrorFilter{} }

// overdueDays 在业务层中执行当前流程或局部处理。
func overdueDays(value string, now time.Time) int {
	date, err := time.ParseInLocation(time.DateOnly, value, now.Location())
	if err != nil || value == "" {
		return 0
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := int(start.Sub(date).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// previewText 在业务层中执行当前流程或局部处理。
func previewText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len([]rune(value)) > 96 {
		return string([]rune(value)[:96]) + "…"
	}
	return value
}
