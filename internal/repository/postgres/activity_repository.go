package postgres

import (
	"context"
	"time"

	models "study-tracker-go/internal/model"
)

type ActivityRepository struct{ store *Store }

// Record 在存储层中执行当前数据访问或局部处理。
func (r *ActivityRepository) Record(ctx context.Context, event models.ActivityEvent) error {
	date := event.Date
	if date == "" {
		date = time.Now().Format(time.DateOnly)
	}
	if event.Value <= 0 {
		event.Value = 1
	}
	_, err := r.store.pool.Exec(ctx, `
		INSERT INTO user_activity_events (user_id, activity_date, event_type, source_key, value)
		VALUES ($1, $2::date, $3, nullif($4, ''), $5)
		ON CONFLICT (source_key) DO NOTHING
	`, r.store.userID, date, event.EventType, event.SourceKey, event.Value)
	return err
}

// List 在存储层中执行当前数据访问或局部处理。
func (r *ActivityRepository) List(ctx context.Context, startDate, endDate time.Time) ([]models.ActivityEvent, error) {
	rows, err := r.store.pool.Query(ctx, `
		SELECT id, activity_date::text, event_type, coalesce(source_key, ''), value, created_at
		FROM user_activity_events
		WHERE user_id = $1 AND activity_date BETWEEN $2::date AND $3::date
		ORDER BY activity_date, created_at, id
	`, r.store.userID, startDate.Format(time.DateOnly), endDate.Format(time.DateOnly))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.ActivityEvent{}
	for rows.Next() {
		var event models.ActivityEvent
		if err := rows.Scan(&event.ID, &event.Date, &event.EventType, &event.SourceKey, &event.Value, &event.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}
