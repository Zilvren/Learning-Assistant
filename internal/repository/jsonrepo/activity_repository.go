package jsonrepo

import (
	"context"
	"sort"
	"time"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type activityState struct {
	NextID int64                  `json:"next_id"`
	Events []models.ActivityEvent `json:"events"`
}

type ActivityRepository struct{ store *base.JSONStore }

// loadActivity 在存储层中执行当前数据访问或局部处理。
func loadActivity(tx *base.JSONTx) (activityState, error) {
	state := activityState{NextID: 1, Events: []models.ActivityEvent{}}
	err := tx.Load("activity.json", &state)
	if state.NextID <= 0 {
		state.NextID = 1
	}
	if state.Events == nil {
		state.Events = []models.ActivityEvent{}
	}
	return state, err
}

// Record 写入去重的本地学习活动事件，使桌面 JSON 模式提供与 PostgreSQL 部署相同的学习热力图信号。
func (r *ActivityRepository) Record(ctx context.Context, event models.ActivityEvent) error {
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		state, err := loadActivity(tx)
		if err != nil {
			return err
		}
		for _, existing := range state.Events {
			if event.SourceKey != "" && existing.SourceKey == event.SourceKey {
				return nil
			}
		}
		now := time.Now()
		if event.Date == "" {
			event.Date = now.Format(time.DateOnly)
		}
		if event.Value <= 0 {
			event.Value = 1
		}
		if event.CreatedAt.IsZero() {
			event.CreatedAt = now.UTC()
		}
		event.ID = state.NextID
		state.NextID++
		state.Events = append(state.Events, event)
		return tx.Save("activity.json", state)
	})
}

// List 在存储层中执行当前数据访问或局部处理。
func (r *ActivityRepository) List(ctx context.Context, startDate, endDate time.Time) ([]models.ActivityEvent, error) {
	result := []models.ActivityEvent{}
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		state, err := loadActivity(tx)
		if err != nil {
			return err
		}
		start := startDate.Format(time.DateOnly)
		end := endDate.Format(time.DateOnly)
		for _, event := range state.Events {
			if event.Date >= start && event.Date <= end {
				result = append(result, event)
			}
		}
		return nil
	})
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date == result[j].Date {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].Date < result[j].Date
	})
	return result, err
}
