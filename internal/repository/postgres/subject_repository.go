package postgres

import (
	"context"
	"fmt"
	"strings"
)

type SubjectRepository struct {
	store *Store
}

func (r *SubjectRepository) List(ctx context.Context) ([]string, error) {
	rows, err := r.store.pool.Query(ctx, `
		SELECT name
		FROM subjects
		WHERE user_id = $1
		  AND deleted_at IS NULL
		ORDER BY sort_order, id
	`, r.store.userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subjects := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		subjects = append(subjects, name)
	}
	return subjects, rows.Err()
}

func (r *SubjectRepository) Exists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := r.store.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM subjects
			WHERE user_id = $1
			  AND name = $2
			  AND deleted_at IS NULL
		)
	`, r.store.userID, strings.TrimSpace(name)).Scan(&exists)
	return exists, err
}

func (r *SubjectRepository) Create(ctx context.Context, name string) ([]string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("科目名称不能为空")
	}
	var maxSort int
	_ = r.store.pool.QueryRow(ctx, `
		SELECT coalesce(max(sort_order), 0)
		FROM subjects
		WHERE user_id = $1
	`, r.store.userID).Scan(&maxSort)

	_, err := r.store.pool.Exec(ctx, `
		INSERT INTO subjects (user_id, name, sort_order)
		VALUES ($1, $2, $3)
	`, r.store.userID, name, maxSort+1)
	if err != nil {
		return nil, err
	}
	return r.List(ctx)
}

func (r *SubjectRepository) Delete(ctx context.Context, name string) ([]string, error) {
	tag, err := r.store.pool.Exec(ctx, `
		UPDATE subjects
		SET deleted_at = now()
		WHERE user_id = $1
		  AND name = $2
		  AND deleted_at IS NULL
	`, r.store.userID, name)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("科目不存在")
	}
	return r.List(ctx)
}

func (r *SubjectRepository) Replace(ctx context.Context, subjects []string) error {
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE subjects
		SET deleted_at = now()
		WHERE user_id = $1
		  AND deleted_at IS NULL
	`, r.store.userID); err != nil {
		return err
	}
	for i, name := range subjects {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO subjects (user_id, name, sort_order)
			VALUES ($1, $2, $3)
		`, r.store.userID, name, i+1); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *SubjectRepository) findID(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.store.pool.QueryRow(ctx, `
		SELECT id
		FROM subjects
		WHERE user_id = $1
		  AND name = $2
		  AND deleted_at IS NULL
	`, r.store.userID, strings.TrimSpace(name)).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *SubjectRepository) ensureID(ctx context.Context, name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("科目名称不能为空")
	}
	id, err := r.findID(ctx, name)
	if err == nil {
		return id, nil
	}
	var newID int64
	err = r.store.pool.QueryRow(ctx, `
		INSERT INTO subjects (user_id, name, sort_order)
		VALUES ($1, $2, (
			SELECT coalesce(max(sort_order), 0) + 1
			FROM subjects
			WHERE user_id = $1
		))
		RETURNING id
	`, r.store.userID, name).Scan(&newID)
	return newID, err
}
