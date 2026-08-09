package jsonrepo

import (
	"context"

	base "study-tracker-go/internal/repository"
)

type OCRTaskRepository struct{}

// Create 在存储层中创建或更新相应状态。
func (r *OCRTaskRepository) Create(ctx context.Context, task base.OCRTask) (int64, error) {
	return 0, nil
}

// Update 在存储层中创建或更新相应状态。
func (r *OCRTaskRepository) Update(ctx context.Context, id int64, task base.OCRTask) error {
	return nil
}
