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

// Record writes a de-duplicated local activity event, so the JSON desktop
// mode provides the same learning heatmap signal as PostgreSQL deployments.
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
