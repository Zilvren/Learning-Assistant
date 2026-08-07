package service

import (
	"context"
	"fmt"
	"time"

	models "study-tracker-go/internal/model"
)

const activityCalendarDays = 183

func GetLearningActivity(ctx context.Context) (models.LearningActivityResponse, error) {
	endDate := time.Now().UTC().Truncate(24 * time.Hour)
	startDate := endDate.AddDate(0, 0, -(activityCalendarDays - 1))
	result := models.LearningActivityResponse{
		StartDate: startDate.Format(time.DateOnly),
		EndDate:   endDate.Format(time.DateOnly),
		Days:      []models.LearningActivityDay{},
	}
	if !AuthEnabled() {
		return result, nil
	}
	userID, ok := UserIDFromContext(ctx)
	if !ok || userID <= 0 {
		return result, fmt.Errorf("未登录")
	}
	defaultMu.RLock()
	pool := pgPool
	defaultMu.RUnlock()
	if pool == nil {
		return result, fmt.Errorf("PostgreSQL 连接池未初始化")
	}
	rows, err := pool.Query(ctx, `
		SELECT activity_date::text, COUNT(*)
		FROM user_activity_events
		WHERE user_id = $1
		  AND activity_date BETWEEN $2::date AND $3::date
		GROUP BY activity_date
		ORDER BY activity_date
	`, userID, result.StartDate, result.EndDate)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var day models.LearningActivityDay
		if err := rows.Scan(&day.Date, &day.Count); err != nil {
			return result, err
		}
		result.Days = append(result.Days, day)
		result.Total += day.Count
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	result.ActiveDays = len(result.Days)
	return result, nil
}
