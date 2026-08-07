package jsonrepo

import (
	"context"

	base "study-tracker-go/internal/repository"
)

type KnowledgeRepository struct {
	store *base.JSONStore
}

func (r *KnowledgeRepository) Load(ctx context.Context) (map[string][]string, error) {
	knowledge := map[string][]string{}
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		return tx.Load("knowledge.json", &knowledge)
	})
	return knowledge, err
}

func (r *KnowledgeRepository) Replace(ctx context.Context, knowledge map[string][]string) error {
	if knowledge == nil {
		knowledge = map[string][]string{}
	}
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		return tx.Save("knowledge.json", knowledge)
	})
}
