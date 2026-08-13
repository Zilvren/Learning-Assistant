package jsonrepo

import (
	"context"
	"fmt"
	"sort"
	"time"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type relationState struct {
	NextID    int64                     `json:"next_id"`
	Relations []models.LearningRelation `json:"relations"`
}

type LearningRelationRepository struct{ store *base.JSONStore }

func loadRelations(tx *base.JSONTx) (relationState, error) {
	state := relationState{NextID: 1, Relations: []models.LearningRelation{}}
	err := tx.Load("relations.json", &state)
	if state.NextID <= 0 {
		state.NextID = 1
	}
	if state.Relations == nil {
		state.Relations = []models.LearningRelation{}
	}
	return state, err
}

func (r *LearningRelationRepository) List(ctx context.Context, sourceType string, sourceID int64) ([]models.LearningRelation, error) {
	result := []models.LearningRelation{}
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		state, err := loadRelations(tx)
		if err != nil {
			return err
		}
		for _, relation := range state.Relations {
			if (relation.FromType == sourceType && relation.FromID == sourceID) || (relation.ToType == sourceType && relation.ToID == sourceID) {
				result = append(result, relation)
			}
		}
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, err
}

func (r *LearningRelationRepository) Create(ctx context.Context, relation models.LearningRelation) (models.LearningRelation, error) {
	err := r.store.Write(ctx, func(tx *base.JSONTx) error {
		state, err := loadRelations(tx)
		if err != nil {
			return err
		}
		for _, existing := range state.Relations {
			if (existing.FromType == relation.FromType && existing.FromID == relation.FromID && existing.ToType == relation.ToType && existing.ToID == relation.ToID) ||
				(existing.FromType == relation.ToType && existing.FromID == relation.ToID && existing.ToType == relation.FromType && existing.ToID == relation.FromID) {
				relation = existing
				return nil
			}
		}
		relation.ID = state.NextID
		state.NextID++
		relation.CreatedAt = time.Now().UTC()
		state.Relations = append(state.Relations, relation)
		return tx.Save("relations.json", state)
	})
	return relation, err
}

func (r *LearningRelationRepository) Delete(ctx context.Context, id int64) error {
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		state, err := loadRelations(tx)
		if err != nil {
			return err
		}
		filtered := state.Relations[:0]
		found := false
		for _, relation := range state.Relations {
			if relation.ID == id {
				found = true
				continue
			}
			filtered = append(filtered, relation)
		}
		if !found {
			return fmt.Errorf("关联不存在")
		}
		state.Relations = filtered
		return tx.Save("relations.json", state)
	})
}
