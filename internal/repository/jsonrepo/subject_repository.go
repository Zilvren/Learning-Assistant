package jsonrepo

import (
	"context"
	"fmt"
	"strings"

	base "study-tracker-go/internal/repository"
)

type SubjectRepository struct {
	store *base.JSONStore
}

// List 在存储层中读取并整理所需数据。
func (r *SubjectRepository) List(ctx context.Context) ([]string, error) {
	subjects := []string{}
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		return loadSubjects(tx, &subjects)
	})
	return subjects, err
}

// Exists 在存储层中完成本文件定义的局部处理。
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

// Create 在存储层中创建或更新相应状态。
func (r *SubjectRepository) Create(ctx context.Context, name string) ([]string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("科目名称不能为空")
	}
	var subjects []string
	err := r.store.Write(ctx, func(tx *base.JSONTx) error {
		if err := loadSubjects(tx, &subjects); err != nil {
			return err
		}
		for _, subject := range subjects {
			if subject == name {
				return fmt.Errorf("科目已存在")
			}
		}
		subjects = append(subjects, name)
		return tx.Save("subjects.json", subjects)
	})
	return subjects, err
}

// Delete 在存储层中删除、清理或撤销相应状态。
func (r *SubjectRepository) Delete(ctx context.Context, name string) ([]string, error) {
	var remaining []string
	err := r.store.Write(ctx, func(tx *base.JSONTx) error {
		subjects := []string{}
		if err := loadSubjects(tx, &subjects); err != nil {
			return err
		}
		found := false
		remaining = []string{}
		for _, subject := range subjects {
			if subject == name {
				found = true
				continue
			}
			remaining = append(remaining, subject)
		}
		if !found {
			return fmt.Errorf("科目不存在")
		}
		return tx.Save("subjects.json", remaining)
	})
	return remaining, err
}

// Replace 在存储层中创建或更新相应状态。
func (r *SubjectRepository) Replace(ctx context.Context, subjects []string) error {
	if subjects == nil {
		subjects = []string{}
	}
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		return tx.Save("subjects.json", subjects)
	})
}

// loadSubjects 在存储层中读取并整理所需数据。
func loadSubjects(tx *base.JSONTx, subjects *[]string) error {
	if err := tx.Load("subjects.json", subjects); err != nil {
		return err
	}
	if *subjects == nil {
		*subjects = []string{}
	}
	return nil
}
