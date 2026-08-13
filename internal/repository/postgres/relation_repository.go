package postgres

import (
	"context"
	"fmt"

	models "study-tracker-go/internal/model"
)

type LearningRelationRepository struct{ store *Store }

func (r *LearningRelationRepository) List(ctx context.Context, sourceType string, sourceID int64) ([]models.LearningRelation, error) {
	rows, err := r.store.pool.Query(ctx, `
		SELECT id, from_type, from_id, to_type, to_id, label, created_at
		FROM learning_relations
		WHERE user_id = $1 AND ((from_type = $2 AND from_id = $3) OR (to_type = $2 AND to_id = $3))
		ORDER BY created_at DESC, id DESC
	`, r.store.userID, sourceType, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.LearningRelation{}
	for rows.Next() {
		var relation models.LearningRelation
		if err := rows.Scan(&relation.ID, &relation.FromType, &relation.FromID, &relation.ToType, &relation.ToID, &relation.Label, &relation.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, relation)
	}
	return result, rows.Err()
}

func (r *LearningRelationRepository) Create(ctx context.Context, relation models.LearningRelation) (models.LearningRelation, error) {
	err := r.store.pool.QueryRow(ctx, `
		INSERT INTO learning_relations (user_id, from_type, from_id, to_type, to_id, label)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, from_type, from_id, to_type, to_id)
		DO UPDATE SET label = excluded.label
		RETURNING id, created_at
	`, r.store.userID, relation.FromType, relation.FromID, relation.ToType, relation.ToID, relation.Label).Scan(&relation.ID, &relation.CreatedAt)
	return relation, err
}

func (r *LearningRelationRepository) Delete(ctx context.Context, id int64) error {
	command, err := r.store.pool.Exec(ctx, "DELETE FROM learning_relations WHERE user_id=$1 AND id=$2", r.store.userID, id)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return fmt.Errorf("关联不存在")
	}
	return nil
}
