package jsonrepo

import (
	"context"
	"fmt"
	"strings"

	base "study-tracker-go/internal/repository"
)

type SubjectRepository struct{}

func (r *SubjectRepository) List(ctx context.Context) ([]string, error) {
	var subjects []string
	if err := base.LoadJSON("subjects.json", &subjects); err != nil {
		return nil, err
	}
	if subjects == nil {
		return []string{}, nil
	}
	return subjects, nil
}

func (r *SubjectRepository) Exists(ctx context.Context, name string) (bool, error) {
	subjects, err := r.List(ctx)
	if err != nil {
		return false, err
	}
	for _, subject := range subjects {
		if subject == name {
			return true, nil
		}
	}
	return false, nil
}

func (r *SubjectRepository) Create(ctx context.Context, name string) ([]string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("科目名称不能为空")
	}
	subjects, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, subject := range subjects {
		if subject == name {
			return nil, fmt.Errorf("科目已存在")
		}
	}
	subjects = append(subjects, name)
	if err := base.SaveJSON("subjects.json", subjects); err != nil {
		return nil, err
	}
	return subjects, nil
}

func (r *SubjectRepository) Delete(ctx context.Context, name string) ([]string, error) {
	subjects, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	found := false
	remaining := []string{}
	for _, subject := range subjects {
		if subject == name {
			found = true
			continue
		}
		remaining = append(remaining, subject)
	}
	if !found {
		return nil, fmt.Errorf("科目不存在")
	}
	if err := base.SaveJSON("subjects.json", remaining); err != nil {
		return nil, err
	}
	return remaining, nil
}

func (r *SubjectRepository) Replace(ctx context.Context, subjects []string) error {
	if subjects == nil {
		subjects = []string{}
	}
	return base.SaveJSON("subjects.json", subjects)
}
