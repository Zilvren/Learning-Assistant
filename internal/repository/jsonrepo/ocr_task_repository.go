package jsonrepo

import (
	"context"

	base "study-tracker-go/internal/repository"
)

type OCRTaskRepository struct{}

func (r *OCRTaskRepository) Create(ctx context.Context, task base.OCRTask) (int64, error) {
	return 0, nil
}

func (r *OCRTaskRepository) Update(ctx context.Context, id int64, task base.OCRTask) error {
	return nil
}
