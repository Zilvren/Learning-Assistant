package jsonrepo

import (
	"context"
	"fmt"
	"sort"
	"time"

	base "study-tracker-go/internal/repository"
)

type ocrTaskState struct {
	NextID int64          `json:"next_id"`
	Tasks  []base.OCRTask `json:"tasks"`
}

type OCRTaskRepository struct{ store *base.JSONStore }

func loadOCRTasks(tx *base.JSONTx) (ocrTaskState, error) {
	state := ocrTaskState{NextID: 1, Tasks: []base.OCRTask{}}
	err := tx.Load("ocr_tasks.json", &state)
	if state.NextID <= 0 {
		state.NextID = 1
	}
	if state.Tasks == nil {
		state.Tasks = []base.OCRTask{}
	}
	return state, err
}

// Create 在存储层中创建或更新相应状态。
func (r *OCRTaskRepository) Create(ctx context.Context, task base.OCRTask) (int64, error) {
	var id int64
	err := r.store.Write(ctx, func(tx *base.JSONTx) error {
		state, err := loadOCRTasks(tx)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if task.Status == "" {
			task.Status = "queued"
		}
		if task.Provider == "" {
			task.Provider = "mineru"
		}
		task.ID = state.NextID
		task.CreatedAt, task.UpdatedAt = now, now
		state.NextID++
		state.Tasks = append(state.Tasks, task)
		id = task.ID
		return tx.Save("ocr_tasks.json", state)
	})
	return id, err
}

// Update 在存储层中创建或更新相应状态。
func (r *OCRTaskRepository) Update(ctx context.Context, id int64, task base.OCRTask) error {
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		state, err := loadOCRTasks(tx)
		if err != nil {
			return err
		}
		for index := range state.Tasks {
			current := &state.Tasks[index]
			if current.ID != id {
				continue
			}
			if task.Status != "" {
				current.Status = task.Status
				if task.Status == "queued" {
					current.ErrorMessage = ""
					current.FinishedAt = nil
				}
			}
			if task.BatchID != "" {
				current.BatchID = task.BatchID
			}
			if task.TaskID != "" {
				current.TaskID = task.TaskID
			}
			if task.ResultMarkdown != "" {
				current.ResultMarkdown = task.ResultMarkdown
			}
			if task.ErrorMessage != "" {
				current.ErrorMessage = task.ErrorMessage
			}
			if task.Metadata != nil {
				current.Metadata = task.Metadata
			}
			if task.FinishedAt != nil {
				current.FinishedAt = task.FinishedAt
			}
			current.UpdatedAt = time.Now().UTC()
			return tx.Save("ocr_tasks.json", state)
		}
		return fmt.Errorf("OCR 任务不存在")
	})
}

func (r *OCRTaskRepository) Get(ctx context.Context, id int64) (base.OCRTask, error) {
	var result base.OCRTask
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		state, err := loadOCRTasks(tx)
		if err != nil {
			return err
		}
		for _, task := range state.Tasks {
			if task.ID == id {
				result = task
				return nil
			}
		}
		return fmt.Errorf("OCR 任务不存在")
	})
	return result, err
}

func (r *OCRTaskRepository) List(ctx context.Context, limit int) ([]base.OCRTask, error) {
	result := []base.OCRTask{}
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		state, err := loadOCRTasks(tx)
		if err != nil {
			return err
		}
		result = append(result, state.Tasks...)
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, err
}
