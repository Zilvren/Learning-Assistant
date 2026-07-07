package jsonrepo

import (
	"context"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type SettingsRepository struct{}

func (r *SettingsRepository) Load(ctx context.Context) (models.Config, error) {
	var config models.Config
	if err := base.LoadJSON("config.json", &config); err != nil {
		return models.Config{}, err
	}
	return config, nil
}

func (r *SettingsRepository) Save(ctx context.Context, config models.Config) error {
	return base.SaveJSON("config.json", config)
}
