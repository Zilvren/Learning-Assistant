package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	models "study-tracker-go/internal/model"
)

type SettingsRepository struct {
	store *Store
}

func (r *SettingsRepository) Load(ctx context.Context) (models.Config, error) {
	var config models.Config
	err := r.store.pool.QueryRow(ctx, `
		SELECT coalesce(mineru_token_cipher, ''), coalesce(display_name, '')
		FROM user_settings
		WHERE user_id = $1
	`, r.store.userID).Scan(&config.MineruToken, &config.Username)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.Config{}, nil
		}
		return models.Config{}, err
	}
	return config, nil
}

func (r *SettingsRepository) Save(ctx context.Context, config models.Config) error {
	_, err := r.store.pool.Exec(ctx, `
		INSERT INTO user_settings (user_id, display_name, mineru_token_cipher, settings)
		VALUES ($1, $2, $3, '{}'::jsonb)
		ON CONFLICT (user_id) DO UPDATE
		SET display_name = excluded.display_name,
		    mineru_token_cipher = excluded.mineru_token_cipher
	`, r.store.userID, config.Username, config.MineruToken)
	return err
}
