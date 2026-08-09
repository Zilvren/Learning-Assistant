package postgres

import (
	"context"
	"encoding/json"

	base "study-tracker-go/internal/repository"
)

type OCRTaskRepository struct {
	store *Store
}

// Create 在存储层中创建或更新相应状态。
func (r *OCRTaskRepository) Create(ctx context.Context, task base.OCRTask) (int64, error) {
	provider := task.Provider
	if provider == "" {
		provider = "mineru"
	}
	status := task.Status
	if status == "" {
		status = "pending"
	}
	var id int64
	err := r.store.pool.QueryRow(ctx, `
		INSERT INTO ocr_tasks (
			user_id, provider, status, source_filename, mime_type, file_size,
			batch_id, task_id, result_markdown, error_message, metadata, finished_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, coalesce($11::jsonb, '{}'::jsonb), $12)
		RETURNING id
	`, r.store.userID, provider, status, task.SourceFilename, task.MimeType, task.FileSize, nullString(task.BatchID), nullString(task.TaskID), nullString(task.ResultMarkdown), nullString(task.ErrorMessage), metadataJSON(task.Metadata), task.FinishedAt).Scan(&id)
	return id, err
}

// Update 在存储层中创建或更新相应状态。
func (r *OCRTaskRepository) Update(ctx context.Context, id int64, task base.OCRTask) error {
	if id == 0 {
		return nil
	}
	_, err := r.store.pool.Exec(ctx, `
		UPDATE ocr_tasks
		SET status = coalesce(nullif($3, ''), status),
		    batch_id = coalesce(nullif($4, ''), batch_id),
		    task_id = coalesce(nullif($5, ''), task_id),
		    result_markdown = coalesce(nullif($6, ''), result_markdown),
		    error_message = coalesce(nullif($7, ''), error_message),
		    metadata = coalesce($8::jsonb, metadata),
		    finished_at = coalesce($9, finished_at)
		WHERE user_id = $1
		  AND id = $2
	`, r.store.userID, id, task.Status, task.BatchID, task.TaskID, task.ResultMarkdown, task.ErrorMessage, metadataJSON(task.Metadata), task.FinishedAt)
	return err
}

// nullString 在存储层中完成本文件定义的局部处理。
func nullString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// metadataJSON 在存储层中完成本文件定义的局部处理。
func metadataJSON(value map[string]interface{}) []byte {
	if value == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return data
}
