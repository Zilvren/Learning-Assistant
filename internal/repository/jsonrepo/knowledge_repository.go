package jsonrepo

import (
	"context"

	base "study-tracker-go/internal/repository"
)

type KnowledgeRepository struct{}

func (r *KnowledgeRepository) Load(ctx context.Context) (map[string][]string, error) {
	var knowledge map[string][]string
	if err := base.LoadJSON("knowledge.json", &knowledge); err != nil {
		return nil, err
	}
	if knowledge == nil {
		return map[string][]string{}, nil
	}
	return knowledge, nil
}

func (r *KnowledgeRepository) Replace(ctx context.Context, knowledge map[string][]string) error {
	if knowledge == nil {
		knowledge = map[string][]string{}
	}
	return base.SaveJSON("knowledge.json", knowledge)
}
