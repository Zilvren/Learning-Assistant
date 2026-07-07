package jsonrepo

import (
	"context"
	"fmt"
	"sort"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type ErrorRepository struct{}

func (r *ErrorRepository) Create(ctx context.Context, item models.ErrorProblem) (models.ErrorProblem, error) {
	errors, err := r.load()
	if err != nil {
		return models.ErrorProblem{}, err
	}
	nextID := 1
	for _, existing := range errors {
		if existing.ID >= nextID {
			nextID = existing.ID + 1
		}
	}
	item.ID = nextID
	errors = append(errors, item)
	if err := r.save(errors); err != nil {
		return models.ErrorProblem{}, err
	}
	return item, nil
}

func (r *ErrorRepository) List(ctx context.Context, filter base.ErrorFilter) ([]models.ErrorProblem, error) {
	errors, err := r.load()
	if err != nil {
		return nil, err
	}
	result := []models.ErrorProblem{}
	for _, item := range errors {
		normalizeReviewFields(&item)
		if !matchesFilter(item, filter) {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *ErrorRepository) Get(ctx context.Context, id int) (models.ErrorProblem, error) {
	errors, err := r.load()
	if err != nil {
		return models.ErrorProblem{}, err
	}
	for _, item := range errors {
		if item.ID == id {
			normalizeReviewFields(&item)
			return item, nil
		}
	}
	return models.ErrorProblem{}, fmt.Errorf("未找到错题 #%d", id)
}

func (r *ErrorRepository) Update(ctx context.Context, id int, req models.UpdateErrorRequest) error {
	errors, err := r.load()
	if err != nil {
		return err
	}
	for i := range errors {
		if errors[i].ID != id {
			continue
		}
		applyErrorUpdate(&errors[i], req)
		return r.save(errors)
	}
	return fmt.Errorf("未找到错题 #%d", id)
}

func (r *ErrorRepository) Delete(ctx context.Context, id int) error {
	errors, err := r.load()
	if err != nil {
		return err
	}
	found := false
	remaining := []models.ErrorProblem{}
	for _, item := range errors {
		if item.ID == id {
			found = true
			continue
		}
		remaining = append(remaining, item)
	}
	if !found {
		return fmt.Errorf("未找到错题 #%d", id)
	}
	return r.save(remaining)
}

func (r *ErrorRepository) UpdateReview(ctx context.Context, id int, reviewedAt string, reviewCount int, reviewStage int, nextReview string) (models.ErrorProblem, error) {
	errors, err := r.load()
	if err != nil {
		return models.ErrorProblem{}, err
	}
	for i := range errors {
		if errors[i].ID != id {
			continue
		}
		errors[i].ReviewCount = reviewCount
		errors[i].ReviewStage = reviewStage
		errors[i].LastReview = &reviewedAt
		errors[i].NextReview = nextReview
		if err := r.save(errors); err != nil {
			return models.ErrorProblem{}, err
		}
		return errors[i], nil
	}
	return models.ErrorProblem{}, fmt.Errorf("未找到错题 #%d", id)
}

func (r *ErrorRepository) ListTags(ctx context.Context) ([]string, error) {
	errors, err := r.load()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, item := range errors {
		for _, tag := range item.Tags {
			seen[tag] = true
		}
		for _, tag := range item.ReasonTags {
			seen[tag] = true
		}
	}
	tags := make([]string, 0, len(seen))
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags, nil
}

func (r *ErrorRepository) Replace(ctx context.Context, errors []models.ErrorProblem) error {
	if errors == nil {
		errors = []models.ErrorProblem{}
	}
	return r.save(errors)
}

func (r *ErrorRepository) HasAny(ctx context.Context) (bool, error) {
	errors, err := r.load()
	if err != nil {
		return false, err
	}
	return len(errors) > 0, nil
}

func (r *ErrorRepository) load() ([]models.ErrorProblem, error) {
	var errors []models.ErrorProblem
	if err := base.LoadJSON("errors.json", &errors); err != nil {
		return nil, err
	}
	if errors == nil {
		errors = []models.ErrorProblem{}
	}
	return errors, nil
}

func (r *ErrorRepository) save(errors []models.ErrorProblem) error {
	return base.SaveJSON("errors.json", errors)
}
