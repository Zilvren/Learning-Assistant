package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	models "study-tracker-go/internal/model"
)

// GetLearningActivity 在业务层中读取并整理所需数据。
func GetLearningActivity(ctx context.Context, requestedYear int) (models.LearningActivityResponse, error) {
	now := time.Now().UTC()
	currentYear := now.Year()
	if requestedYear == 0 {
		requestedYear = currentYear
	}
	if requestedYear < 2000 || requestedYear > currentYear {
		return models.LearningActivityResponse{}, fmt.Errorf("请选择 2000 至 %d 年之间的学习记录", currentYear)
	}
	startDate := time.Date(requestedYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(requestedYear, time.December, 31, 0, 0, 0, 0, time.UTC)
	result := models.LearningActivityResponse{
		StartDate:      startDate.Format(time.DateOnly),
		EndDate:        endDate.Format(time.DateOnly),
		Days:           []models.LearningActivityDay{},
		AvailableYears: []int{requestedYear},
	}
	app, err := appFor(ctx)
	if err != nil {
		return result, err
	}
	if !app.AuthEnabled() {
		return result, nil
	}
	userID, ok := UserIDFromContext(ctx)
	if !ok || userID <= 0 {
		return result, fmt.Errorf("未登录")
	}
	pool := app.pool
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

	yearRows, err := pool.Query(ctx, `
		SELECT DISTINCT EXTRACT(YEAR FROM activity_date)::int
		FROM user_activity_events
		WHERE user_id = $1
		ORDER BY 1 DESC
	`, userID)
	if err != nil {
		return result, err
	}
	defer yearRows.Close()
	yearSet := map[int]struct{}{requestedYear: {}}
	for yearRows.Next() {
		var year int
		if err := yearRows.Scan(&year); err != nil {
			return result, err
		}
		yearSet[year] = struct{}{}
	}
	if err := yearRows.Err(); err != nil {
		return result, err
	}
	result.AvailableYears = result.AvailableYears[:0]
	for year := range yearSet {
		result.AvailableYears = append(result.AvailableYears, year)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(result.AvailableYears)))
	return result, nil
}
