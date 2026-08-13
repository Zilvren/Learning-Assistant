package jsonrepo

import (
	"context"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type SettingsRepository struct {
	store *base.JSONStore
}

// Load 在存储层中读取并整理所需数据。
func (r *SettingsRepository) Load(ctx context.Context) (models.Config, error) {
	var config models.Config
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		return tx.Load("config.json", &config)
	})
	if err == nil {
		config.MineruToken, err = base.OpenSecret(config.MineruToken)
		if err == nil {
			config.DeepSeekToken, err = base.OpenSecret(config.DeepSeekToken)
		}
	}
	return config, err
}

// Save 在存储层中创建或更新相应状态。
func (r *SettingsRepository) Save(ctx context.Context, config models.Config) error {
	sealedToken, err := base.SealSecret(config.MineruToken)
	if err != nil {
		return err
	}
	sealedDeepSeekToken, err := base.SealSecret(config.DeepSeekToken)
	if err != nil {
		return err
	}
	config.MineruToken = sealedToken
	config.DeepSeekToken = sealedDeepSeekToken
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		return tx.Save("config.json", config)
	})
}
