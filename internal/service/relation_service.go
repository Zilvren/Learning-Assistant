package service

import (
	"context"
	"fmt"
	"strings"

	models "study-tracker-go/internal/model"
)

func validRelationTarget(sourceType string, id int64) bool {
	return (sourceType == "library" || sourceType == "error") && id > 0
}

func relationTargetName(ctx context.Context, sourceType string, id int64) (string, error) {
	if sourceType == "library" {
		item, err := GetLibraryItem(ctx, id)
		return item.Name, err
	}
	problem, err := GetError(ctx, int(id))
	return problem.Title, err
}

func ListLearningRelations(ctx context.Context, sourceType string, sourceID int64) ([]models.LearningRelation, error) {
	if !validRelationTarget(sourceType, sourceID) {
		return nil, fmt.Errorf("关联来源无效")
	}
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	relations, err := repos.Relations.List(ctx, sourceType, sourceID)
	if err != nil {
		return nil, err
	}
	for index := range relations {
		otherType, otherID := relations[index].ToType, relations[index].ToID
		if relations[index].ToType == sourceType && relations[index].ToID == sourceID {
			otherType, otherID = relations[index].FromType, relations[index].FromID
		}
		relations[index].TargetType = otherType
		relations[index].TargetID = otherID
		relations[index].TargetName, _ = relationTargetName(ctx, otherType, otherID)
	}
	return relations, nil
}

func CreateLearningRelation(ctx context.Context, relation models.LearningRelation) (models.LearningRelation, error) {
	relation.FromType = strings.TrimSpace(relation.FromType)
	relation.ToType = strings.TrimSpace(relation.ToType)
	relation.Label = strings.TrimSpace(relation.Label)
	if !validRelationTarget(relation.FromType, relation.FromID) || !validRelationTarget(relation.ToType, relation.ToID) {
		return relation, fmt.Errorf("关联对象无效")
	}
	if relation.FromType == relation.ToType && relation.FromID == relation.ToID {
		return relation, fmt.Errorf("不能关联到自身")
	}
	if _, err := relationTargetName(ctx, relation.FromType, relation.FromID); err != nil {
		return relation, err
	}
	if _, err := relationTargetName(ctx, relation.ToType, relation.ToID); err != nil {
		return relation, err
	}
	repos, err := repositories(ctx)
	if err != nil {
		return relation, err
	}
	return repos.Relations.Create(ctx, relation)
}

func DeleteLearningRelation(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("关联 ID 无效")
	}
	repos, err := repositories(ctx)
	if err != nil {
		return err
	}
	return repos.Relations.Delete(ctx, id)
}
