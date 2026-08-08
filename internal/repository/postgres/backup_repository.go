package postgres

import (
	"context"
	"fmt"
	"sort"
	"time"

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
	library, err := r.exportLibrary(ctx)
	if err != nil {
		return base.BackupData{}, err
	}
	return base.BackupData{
		Subjects:  &subjects,
		Errors:    &errors,
		Config:    &config,
		Knowledge: &knowledge,
		Library:   library,
	}, nil
}

func (r *BackupRepository) Import(ctx context.Context, data base.BackupData) error {
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	errorRepo := &ErrorRepository{store: r.store}
	if data.Library != nil {
		// parent_id uses ON DELETE RESTRICT. Remove leaves in order so importing
		// a ZIP also works when the current library already has folders.
		for {
			result, deleteErr := tx.Exec(ctx, `
				DELETE FROM library_items item
				WHERE item.user_id = $1
				  AND NOT EXISTS (
					  SELECT 1
					  FROM library_items child
					  WHERE child.user_id = item.user_id
					    AND child.parent_id = item.id
				  )
			`, r.store.userID)
			if deleteErr != nil {
				return deleteErr
			}
			if result.RowsAffected() == 0 {
				break
			}
		}
	}
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
	errorIDMap := map[int]int64{}
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
			if item.ID > 0 {
				errorIDMap[item.ID] = id
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
	if data.Library != nil {
		if err := r.importLibrary(ctx, tx, *data.Library, errorIDMap); err != nil {
			return err
		}
		if err := r.recordLibraryImportActivity(ctx, tx, *data.Library); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *BackupRepository) exportLibrary(ctx context.Context) (*base.LibraryBackup, error) {
	library := &base.LibraryBackup{
		SchemaVersion: 2,
		NextID:        1,
		NextVersionID: 1,
		Items:         []models.LibraryItem{},
		Versions:      []models.LibraryVersion{},
	}
	rows, err := r.store.pool.Query(ctx, "SELECT "+libraryColumns+" FROM library_items WHERE user_id = $1 ORDER BY id", r.store.userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		item, scanErr := scanLibraryRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		library.Items = append(library.Items, item)
		if item.ID >= library.NextID {
			library.NextID = item.ID + 1
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	versionRows, err := r.store.pool.Query(ctx, `
		SELECT id, item_id, version, blob_hash, file_size, created_at
		FROM library_versions
		WHERE user_id = $1
		ORDER BY id
	`, r.store.userID)
	if err != nil {
		return nil, err
	}
	defer versionRows.Close()
	for versionRows.Next() {
		var version models.LibraryVersion
		if err := versionRows.Scan(&version.ID, &version.ItemID, &version.Version, &version.BlobHash, &version.Size, &version.CreatedAt); err != nil {
			return nil, err
		}
		library.Versions = append(library.Versions, version)
		if version.ID >= library.NextVersionID {
			library.NextVersionID = version.ID + 1
		}
	}
	if err := versionRows.Err(); err != nil {
		return nil, err
	}
	return library, nil
}

func (r *BackupRepository) importLibrary(ctx context.Context, tx pgx.Tx, library base.LibraryBackup, errorIDMap map[int]int64) error {
	items := append([]models.LibraryItem(nil), library.Items...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	itemIDMap := make(map[int64]int64, len(items))
	remaining := append([]models.LibraryItem(nil), items...)

	for len(remaining) > 0 {
		next := remaining[:0]
		inserted := false
		for _, item := range remaining {
			if item.ID <= 0 {
				return fmt.Errorf("资料库备份包含无效项目 ID")
			}
			if _, exists := itemIDMap[item.ID]; exists {
				return fmt.Errorf("资料库备份包含重复项目 ID：%d", item.ID)
			}
			if !validBackupLibraryKind(item.Kind) {
				return fmt.Errorf("资料库备份包含无效项目类型：%s", item.Kind)
			}
			var parentID any
			if item.ParentID != nil {
				mappedParent, ok := itemIDMap[*item.ParentID]
				if !ok {
					next = append(next, item)
					continue
				}
				parentID = mappedParent
			}

			createdAt := backupLibraryTimestamp(item.CreatedAt)
			updatedAt := backupLibraryTimestamp(item.UpdatedAt)
			currentVersion := item.CurrentVersion
			if currentVersion < 0 {
				currentVersion = 0
			}
			fileSize := item.Size
			if fileSize < 0 {
				fileSize = 0
			}
			var errorProblemID any
			if item.ErrorProblemID != nil {
				if mappedErrorID, ok := errorIDMap[*item.ErrorProblemID]; ok {
					errorProblemID = mappedErrorID
				}
			}
			var newID int64
			err := tx.QueryRow(ctx, `
				INSERT INTO library_items (
					user_id, parent_id, kind, name, mime_type, file_size, tags, pinned,
					current_version, error_problem_id, blob_hash, review_enabled,
					review_count, review_stage, last_review, next_review, created_at,
					updated_at, deleted_at
				)
				VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
					$13, $14, $15, $16, $17, $18, $19
				)
				RETURNING id
			`, r.store.userID, parentID, item.Kind, item.Name, item.MimeType, fileSize,
				normalizeLibraryTags(item.Tags), item.Pinned, currentVersion, errorProblemID,
				item.BlobHash, item.ReviewEnabled, item.ReviewCount, item.ReviewStage,
				item.LastReview, item.NextReview, createdAt, updatedAt, item.DeletedAt,
			).Scan(&newID)
			if err != nil {
				return err
			}
			itemIDMap[item.ID] = newID
			inserted = true
		}
		if !inserted {
			return fmt.Errorf("资料库备份存在缺失或循环的文件夹层级")
		}
		remaining = next
	}

	for _, item := range items {
		if item.OriginalParent == nil {
			continue
		}
		newID := itemIDMap[item.ID]
		if originalParentID, ok := itemIDMap[*item.OriginalParent]; ok {
			if _, err := tx.Exec(ctx, `UPDATE library_items SET original_parent_id = $3 WHERE user_id = $1 AND id = $2`, r.store.userID, newID, originalParentID); err != nil {
				return err
			}
		}
	}

	for _, version := range library.Versions {
		itemID, ok := itemIDMap[version.ItemID]
		if !ok {
			return fmt.Errorf("资料库备份版本引用了不存在的项目：%d", version.ItemID)
		}
		if version.Version <= 0 {
			return fmt.Errorf("资料库备份包含无效版本号")
		}
		fileSize := version.Size
		if fileSize < 0 {
			fileSize = 0
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO library_versions (user_id, item_id, version, blob_hash, file_size, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, r.store.userID, itemID, version.Version, version.BlobHash, fileSize, backupLibraryTimestamp(version.CreatedAt)); err != nil {
			return err
		}
	}
	return nil
}

func validBackupLibraryKind(kind string) bool {
	return kind == "folder" || kind == "note" || kind == "file" || kind == "error"
}

// recordLibraryImportActivity records the act of bringing learning material
// into the current account. Imported files retain their original timestamps,
// so relying only on item triggers would otherwise place every activity on an
// old date and leave today's dashboard empty after a restore.
func (r *BackupRepository) recordLibraryImportActivity(ctx context.Context, tx pgx.Tx, library base.LibraryBackup) error {
	location := time.FixedZone("CST", 8*60*60)
	now := time.Now().In(location)
	day := now.Format(time.DateOnly)
	prefix := fmt.Sprintf("library-import:%d:%d", r.store.userID, now.UnixNano())
	index := 0
	for _, item := range library.Items {
		if item.Kind == "folder" {
			continue
		}
		index++
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_activity_events (user_id, activity_date, event_type, source_key)
			VALUES ($1, $2::date, 'library_import', $3)
		`, r.store.userID, day, fmt.Sprintf("%s:%d", prefix, index)); err != nil {
			return err
		}
	}
	return nil
}

func backupLibraryTimestamp(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value
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
			UNION ALL
			SELECT 1 FROM library_items WHERE user_id = $1
		)
	`, r.store.userID).Scan(&exists)
	return exists, err
}

func (r *BackupRepository) saveConfig(ctx context.Context, tx pgx.Tx, config *models.Config) error {
	sealedToken, err := base.SealSecret(config.MineruToken)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO user_settings (user_id, display_name, mineru_token_cipher, settings)
		VALUES ($1, $2, $3, '{}'::jsonb)
		ON CONFLICT (user_id) DO UPDATE
		SET display_name = excluded.display_name,
		    mineru_token_cipher = CASE
				WHEN excluded.mineru_token_cipher = '' THEN user_settings.mineru_token_cipher
				ELSE excluded.mineru_token_cipher
			END
	`, r.store.userID, config.Username, sealedToken)
	return err
}
