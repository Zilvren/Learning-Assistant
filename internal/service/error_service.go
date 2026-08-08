package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/repository"
)

func CreateError(ctx context.Context, req models.AddErrorRequest) (models.ErrorProblem, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return models.ErrorProblem{}, err
	}

	req.Subject = strings.TrimSpace(req.Subject)
	req.Question = strings.TrimSpace(req.Question)
	if req.Subject == "" {
		return models.ErrorProblem{}, fmt.Errorf("无效科目")
	}
	exists, err := repos.Subjects.Exists(ctx, req.Subject)
	if err != nil {
		return models.ErrorProblem{}, err
	}
	if !exists {
		return models.ErrorProblem{}, fmt.Errorf("无效科目")
	}
	if req.Question == "" {
		return models.ErrorProblem{}, fmt.Errorf("题目不能为空")
	}

	if req.Wrong == "" {
		req.Wrong = "未记录"
	}
	if req.Correct == "" {
		req.Correct = "未记录"
	}
	if req.Reason == "" {
		req.Reason = "未记录"
	}
	if req.Title == "" {
		req.Title = firstRunes(req.Question, 40)
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}
	if req.ReasonTags == nil {
		req.ReasonTags = []string{}
	}

	now := time.Now()
	item := models.ErrorProblem{
		Subject:     req.Subject,
		Title:       req.Title,
		Question:    req.Question,
		Wrong:       req.Wrong,
		Correct:     req.Correct,
		Reason:      req.Reason,
		Tags:        req.Tags,
		ReasonTags:  req.ReasonTags,
		Created:     now.Format("2006-01-02 15:04:05"),
		ReviewCount: 0,
		LastReview:  nil,
		ReviewStage: 0,
		NextReview:  now.Format("2006-01-02"),
	}
	created, err := repos.Errors.Create(ctx, item)
	if err != nil {
		return models.ErrorProblem{}, err
	}
	all, _ := repos.Errors.List(ctx, repository.ErrorFilter{})
	subjects, _ := repos.Subjects.List(ctx)
	if err := repos.Library.EnsureLegacy(ctx, all, subjects); err != nil {
		return models.ErrorProblem{}, err
	}
	if req.ParentID != nil {
		matches, _ := repos.Library.List(ctx, repository.LibraryFilter{Kind: "error", Query: created.Title})
		for _, node := range matches {
			if node.ErrorProblemID != nil && *node.ErrorProblemID == created.ID {
				_, err = repos.Library.Update(ctx, node.ID, models.UpdateLibraryItemRequest{ParentID: req.ParentID, ParentSet: true})
				break
			}
		}
		if err != nil {
			return models.ErrorProblem{}, err
		}
	}
	return created, nil
}

func firstRunes(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max])
}

func GetAllErrors(ctx context.Context, subject, keyword, tag, reasonTag string) ([]models.ErrorProblem, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	return repos.Errors.List(ctx, repository.ErrorFilter{
		Subject:   subject,
		Keyword:   keyword,
		Tag:       tag,
		ReasonTag: reasonTag,
	})
}

func UpdateError(ctx context.Context, id int, req models.UpdateErrorRequest) error {
	repos, err := repositories(ctx)
	if err != nil {
		return err
	}
	if req.Subject != nil {
		trimmed := strings.TrimSpace(*req.Subject)
		if trimmed == "" {
			return fmt.Errorf("无效科目")
		}
		exists, err := repos.Subjects.Exists(ctx, trimmed)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("无效科目")
		}
		*req.Subject = trimmed
	}
	if req.Question != nil && strings.TrimSpace(*req.Question) == "" {
		return fmt.Errorf("题目不能为空")
	}
	if err := repos.Errors.Update(ctx, id, req); err != nil {
		return err
	}
	all, _ := repos.Errors.List(ctx, repository.ErrorFilter{})
	subjects, _ := repos.Subjects.List(ctx)
	return repos.Library.EnsureLegacy(ctx, all, subjects)
}

func DeleteError(ctx context.Context, id int) error {
	repos, err := repositories(ctx)
	if err != nil {
		return err
	}
	return repos.Errors.Delete(ctx, id)
}

var reviewIntervals = []int{0, 1, 2, 4, 7, 15}

func ReviewError(ctx context.Context, id int) (models.ErrorProblem, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return models.ErrorProblem{}, err
	}
	return repos.Errors.Review(ctx, id, time.Now(), reviewIntervals)
}

func GetAllTags(ctx context.Context) ([]string, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	return repos.Errors.ListTags(ctx)
}
