package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type KnowledgeRepository struct {
	store *Store
}

func (r *KnowledgeRepository) Load(ctx context.Context) (map[string][]string, error) {
	rows, err := r.store.pool.Query(ctx, `
		SELECT coalesce(s.name, '未分类'), k.content
		FROM knowledge_items k
		LEFT JOIN subjects s ON s.user_id = k.user_id AND s.id = k.subject_id
		WHERE k.user_id = $1
		  AND k.deleted_at IS NULL
		ORDER BY coalesce(s.sort_order, 0), s.id, k.id
	`, r.store.userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	knowledge := map[string][]string{}
	for rows.Next() {
		var subject string
		var content string
		if err := rows.Scan(&subject, &content); err != nil {
			return nil, err
		}
		knowledge[subject] = append(knowledge[subject], content)
	}
	return knowledge, rows.Err()
}

func (r *KnowledgeRepository) Replace(ctx context.Context, knowledge map[string][]string) error {
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		DELETE FROM knowledge_items
		WHERE user_id = $1
	`, r.store.userID); err != nil {
		return err
	}

	for subject, items := range knowledge {
		subjectID, err := r.ensureSubjectID(ctx, tx, subject)
		if err != nil {
			return err
		}
		for _, content := range items {
			if content == "" {
				continue
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO knowledge_items (user_id, subject_id, content, source)
				VALUES ($1, $2, $3, 'import')
			`, r.store.userID, subjectID, content); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func (r *KnowledgeRepository) ensureSubjectID(ctx context.Context, tx pgx.Tx, subject string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM subjects
		WHERE user_id = $1
		  AND name = $2
		  AND deleted_at IS NULL
	`, r.store.userID, subject).Scan(&id)
	if err == nil {
		return id, nil
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO subjects (user_id, name, sort_order)
		VALUES ($1, $2, (
			SELECT coalesce(max(sort_order), 0) + 1
			FROM subjects
			WHERE user_id = $1
		))
		RETURNING id
	`, r.store.userID, subject).Scan(&id)
	return id, err
}
