package jsonrepo

import (
	"context"
	"fmt"
	"sort"
	"time"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type ErrorRepository struct {
	store *base.JSONStore
}

// Create 在存储层中创建或更新相应状态。
func (r *ErrorRepository) Create(ctx context.Context, item models.ErrorProblem) (models.ErrorProblem, error) {
	err := r.store.Write(ctx, func(tx *base.JSONTx) error {
		errors, err := loadErrors(tx)
		if err != nil {
			return err
		}
		nextID := 1
		for _, existing := range errors {
			if existing.ID >= nextID {
				nextID = existing.ID + 1
			}
		}
		item.ID = nextID
		errors = append(errors, item)
		return tx.Save("errors.json", errors)
	})
	return item, err
}

// List 在存储层中读取并整理所需数据。
func (r *ErrorRepository) List(ctx context.Context, filter base.ErrorFilter) ([]models.ErrorProblem, error) {
	result := []models.ErrorProblem{}
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		errors, err := loadErrors(tx)
		if err != nil {
			return err
		}
		for _, item := range errors {
			normalizeReviewFields(&item)
			if matchesFilter(item, filter) {
				result = append(result, item)
			}
		}
		return nil
	})
	return result, err
}

// Get 在存储层中读取并整理所需数据。
func (r *ErrorRepository) Get(ctx context.Context, id int) (models.ErrorProblem, error) {
	var result models.ErrorProblem
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		errors, err := loadErrors(tx)
		if err != nil {
			return err
		}
		for _, item := range errors {
			if item.ID == id {
				normalizeReviewFields(&item)
				result = item
				return nil
			}
		}
		return fmt.Errorf("未找到错题 #%d", id)
	})
	return result, err
}

// Update 在存储层中创建或更新相应状态。
func (r *ErrorRepository) Update(ctx context.Context, id int, req models.UpdateErrorRequest) error {
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		errors, err := loadErrors(tx)
		if err != nil {
			return err
		}
		for i := range errors {
			if errors[i].ID == id {
				applyErrorUpdate(&errors[i], req)
				return tx.Save("errors.json", errors)
			}
		}
		return fmt.Errorf("未找到错题 #%d", id)
	})
}

// Delete 在存储层中删除、清理或撤销相应状态。
func (r *ErrorRepository) Delete(ctx context.Context, id int) error {
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		errors, err := loadErrors(tx)
		if err != nil {
			return err
		}
		found := false
		remaining := make([]models.ErrorProblem, 0, len(errors))
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
		return tx.Save("errors.json", remaining)
	})
}

// Review 在存储层中完成本文件定义的局部处理。
func (r *ErrorRepository) Review(ctx context.Context, id int, reviewedAt time.Time, intervals []int) (models.ErrorProblem, error) {
	var result models.ErrorProblem
	err := r.store.Write(ctx, func(tx *base.JSONTx) error {
		errors, err := loadErrors(tx)
		if err != nil {
			return err
		}
		for i := range errors {
			if errors[i].ID != id {
				continue
			}
			count := errors[i].ReviewCount + 1
			index := count
			if index < 0 {
				index = 0
			}
			if len(intervals) == 0 {
				intervals = []int{0}
			}
			if index >= len(intervals) {
				index = len(intervals) - 1
			}
			reviewed := reviewedAt.Format("2006-01-02 15:04:05")
			errors[i].ReviewCount = count
			errors[i].ReviewStage = count
			errors[i].LastReview = &reviewed
			errors[i].NextReview = reviewedAt.AddDate(0, 0, intervals[index]).Format("2006-01-02")
			if err := tx.Save("errors.json", errors); err != nil {
				return err
			}
			result = errors[i]
			return nil
		}
		return fmt.Errorf("未找到错题 #%d", id)
	})
	return result, err
}

// ListTags 在存储层中读取并整理所需数据。
func (r *ErrorRepository) ListTags(ctx context.Context) ([]string, error) {
	tags := []string{}
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		errors, err := loadErrors(tx)
		if err != nil {
			return err
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
		for tag := range seen {
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		return nil
	})
	return tags, err
}

// Replace 在存储层中创建或更新相应状态。
func (r *ErrorRepository) Replace(ctx context.Context, errors []models.ErrorProblem) error {
	if errors == nil {
		errors = []models.ErrorProblem{}
	}
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		return tx.Save("errors.json", errors)
	})
}

// HasAny 在存储层中校验输入或判断当前条件。
func (r *ErrorRepository) HasAny(ctx context.Context) (bool, error) {
	hasAny := false
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		errors, err := loadErrors(tx)
		if err != nil {
			return err
		}
		hasAny = len(errors) > 0
		return nil
	})
	return hasAny, err
}

// loadErrors 在存储层中读取并整理所需数据。
func loadErrors(tx *base.JSONTx) ([]models.ErrorProblem, error) {
	errors := []models.ErrorProblem{}
	if err := tx.Load("errors.json", &errors); err != nil {
		return nil, err
	}
	return errors, nil
}
