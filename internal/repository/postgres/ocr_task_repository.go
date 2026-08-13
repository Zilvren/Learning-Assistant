package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
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
			batch_id, task_id, result_markdown, input_blob_hash, error_message, metadata, finished_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, coalesce($12::jsonb, '{}'::jsonb), $13)
		RETURNING id
	`, r.store.userID, provider, status, task.SourceFilename, task.MimeType, task.FileSize, nullString(task.BatchID), nullString(task.TaskID), nullString(task.ResultMarkdown), nullString(task.InputBlobHash), nullString(task.ErrorMessage), metadataJSON(task.Metadata), task.FinishedAt).Scan(&id)
	return id, err
}

func scanOCRTask(row pgx.Row) (base.OCRTask, error) {
	var task base.OCRTask
	var metadata []byte
	err := row.Scan(&task.ID, &task.Provider, &task.Status, &task.SourceFilename, &task.MimeType, &task.FileSize, &task.BatchID, &task.TaskID, &task.ResultMarkdown, &task.InputBlobHash, &task.ErrorMessage, &metadata, &task.CreatedAt, &task.UpdatedAt, &task.FinishedAt)
	if err == nil && len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &task.Metadata)
	}
	return task, err
}

const ocrTaskColumns = "id,provider,status,source_filename,mime_type,file_size,coalesce(batch_id,''),coalesce(task_id,''),coalesce(result_markdown,''),coalesce(input_blob_hash,''),coalesce(error_message,''),metadata,created_at,updated_at,finished_at"

func (r *OCRTaskRepository) Get(ctx context.Context, id int64) (base.OCRTask, error) {
	task, err := scanOCRTask(r.store.pool.QueryRow(ctx, "SELECT "+ocrTaskColumns+" FROM ocr_tasks WHERE user_id=$1 AND id=$2", r.store.userID, id))
	if err == pgx.ErrNoRows {
		return task, fmt.Errorf("OCR 任务不存在")
	}
	return task, err
}

func (r *OCRTaskRepository) List(ctx context.Context, limit int) ([]base.OCRTask, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := r.store.pool.Query(ctx, "SELECT "+ocrTaskColumns+" FROM ocr_tasks WHERE user_id=$1 ORDER BY created_at DESC, id DESC LIMIT $2", r.store.userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []base.OCRTask{}
	for rows.Next() {
		task, err := scanOCRTask(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, task)
	}
	return result, rows.Err()
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
		    error_message = CASE WHEN $3 = 'queued' THEN NULL ELSE coalesce(nullif($7, ''), error_message) END,
		    metadata = coalesce($8::jsonb, metadata),
		    finished_at = CASE WHEN $3 = 'queued' THEN NULL ELSE coalesce($9, finished_at) END
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
