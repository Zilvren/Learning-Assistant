package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type SettingsRepository struct {
	store *Store
}

// Load 在存储层中读取并整理所需数据。
func (r *SettingsRepository) Load(ctx context.Context) (models.Config, error) {
	var config models.Config
	var storedToken string
	var settings []byte
	err := r.store.pool.QueryRow(ctx, `
		SELECT coalesce(mineru_token_cipher, ''), coalesce(display_name, ''), settings
		FROM user_settings
		WHERE user_id = $1
	`, r.store.userID).Scan(&storedToken, &config.Username, &settings)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.Config{}, nil
		}
		return models.Config{}, err
	}
	config.MineruToken, err = base.OpenSecret(storedToken)
	if err == nil && len(settings) > 0 {
		var extra struct {
			DailyGoal     models.DailyGoalSettings `json:"daily_goal"`
			DeepSeekToken string                   `json:"deepseek_token"`
			DeepSeekModel string                   `json:"deepseek_model"`
		}
		if json.Unmarshal(settings, &extra) == nil {
			config.DailyGoal = extra.DailyGoal
			config.DeepSeekToken, err = base.OpenSecret(extra.DeepSeekToken)
			config.DeepSeekModel = extra.DeepSeekModel
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
	settings, err := json.Marshal(struct {
		DailyGoal     models.DailyGoalSettings `json:"daily_goal"`
		DeepSeekToken string                   `json:"deepseek_token"`
		DeepSeekModel string                   `json:"deepseek_model"`
	}{DailyGoal: config.DailyGoal, DeepSeekToken: sealedDeepSeekToken, DeepSeekModel: config.DeepSeekModel})
	if err != nil {
		return err
	}
	_, err = r.store.pool.Exec(ctx, `
		INSERT INTO user_settings (user_id, display_name, mineru_token_cipher, settings)
		VALUES ($1, $2, $3, $4::jsonb)
		ON CONFLICT (user_id) DO UPDATE
		SET display_name = excluded.display_name,
		    mineru_token_cipher = excluded.mineru_token_cipher,
		    settings = user_settings.settings || excluded.settings
	`, r.store.userID, config.Username, sealedToken, settings)
	return err
}
