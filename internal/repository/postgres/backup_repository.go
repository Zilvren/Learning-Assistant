package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type BackupRepository struct {
	store *Store
}

func (r *BackupRepository) Export(ctx context.Context) (base.BackupData, error) {
	subjects, err := (&SubjectRepository{store: r.store}).List(ctx)
	if err != nil {
		return base.BackupData{}, err
	}
	errors, err := (&ErrorRepository{store: r.store}).List(ctx, base.ErrorFilter{})
	if err != nil {
		return base.BackupData{}, err
	}
	config, err := (&SettingsRepository{store: r.store}).Load(ctx)
	if err != nil {
		return base.BackupData{}, err
	}
	knowledge, err := (&KnowledgeRepository{store: r.store}).Load(ctx)
	if err != nil {
		return base.BackupData{}, err
	}
	return base.BackupData{
		Subjects:  &subjects,
		Errors:    &errors,
		Config:    &config,
		Knowledge: &knowledge,
	}, nil
}

func (r *BackupRepository) Import(ctx context.Context, data base.BackupData) error {
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	errorRepo := &ErrorRepository{store: r.store}
	if data.Errors != nil {
		if err := errorRepo.clearUserErrors(ctx, tx); err != nil {
			return err
		}
	}
	if data.Knowledge != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM knowledge_items WHERE user_id = $1`, r.store.userID); err != nil {
			return err
		}
	}
	if data.Subjects != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE subjects
			SET deleted_at = now()
			WHERE user_id = $1
			  AND deleted_at IS NULL
		`, r.store.userID); err != nil {
			return err
		}
		for i, subject := range *data.Subjects {
			if subject == "" {
				continue
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO subjects (user_id, name, sort_order)
				VALUES ($1, $2, $3)
			`, r.store.userID, subject, i+1); err != nil {
				return err
			}
		}
	}
	if data.Config != nil {
		if err := r.saveConfig(ctx, tx, data.Config); err != nil {
			return err
		}
	}
	if data.Errors != nil {
		for _, item := range *data.Errors {
			normalizeProblem(&item)
			subjectID, err := errorRepo.ensureSubjectIDTx(ctx, tx, item.Subject)
			if err != nil {
				return err
			}
			id, err := errorRepo.insertProblem(ctx, tx, item, subjectID, item.ID > 0)
			if err != nil {
				return err
			}
			if err := errorRepo.replaceTags(ctx, tx, id, "question", item.Tags); err != nil {
				return err
			}
			if err := errorRepo.replaceTags(ctx, tx, id, "reason", item.ReasonTags); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `
			SELECT setval(pg_get_serial_sequence('error_problems', 'id'), greatest((SELECT coalesce(max(id), 1) FROM error_problems), 1), true)
		`); err != nil {
			return err
		}
	}
	if data.Knowledge != nil {
		for subject, items := range *data.Knowledge {
			subjectID, err := errorRepo.ensureSubjectIDTx(ctx, tx, subject)
			if err != nil {
				return err
			}
			for _, content := range items {
				if content == "" {
					continue
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO knowledge_items (user_id, subject_id, content, source)
					VALUES ($1, $2, $3, 'import')
				`, r.store.userID, subjectID, content); err != nil {
					return err
				}
			}
		}
	}
	return tx.Commit(ctx)
}

func (r *BackupRepository) HasData(ctx context.Context) (bool, error) {
	var exists bool
	err := r.store.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM subjects WHERE user_id = $1 AND deleted_at IS NULL
			UNION ALL
			SELECT 1 FROM error_problems WHERE user_id = $1 AND deleted_at IS NULL
			UNION ALL
			SELECT 1 FROM user_settings WHERE user_id = $1
			UNION ALL
			SELECT 1 FROM knowledge_items WHERE user_id = $1 AND deleted_at IS NULL
		)
	`, r.store.userID).Scan(&exists)
	return exists, err
}

func (r *BackupRepository) saveConfig(ctx context.Context, tx pgx.Tx, config *models.Config) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_settings (user_id, display_name, mineru_token_cipher, settings)
		VALUES ($1, $2, $3, '{}'::jsonb)
		ON CONFLICT (user_id) DO UPDATE
		SET display_name = excluded.display_name,
		    mineru_token_cipher = excluded.mineru_token_cipher
	`, r.store.userID, config.Username, config.MineruToken)
	return err
}
