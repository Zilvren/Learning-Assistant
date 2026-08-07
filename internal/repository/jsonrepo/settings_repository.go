package jsonrepo

import (
	"context"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type SettingsRepository struct {
	store *base.JSONStore
}

func (r *SettingsRepository) Load(ctx context.Context) (models.Config, error) {
	var config models.Config
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		return tx.Load("config.json", &config)
	})
	return config, err
}

func (r *SettingsRepository) Save(ctx context.Context, config models.Config) error {
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		return tx.Save("config.json", config)
	})
}
