package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	models "study-tracker-go/internal/model"
)

// GetLearningActivity 在业务层中读取并整理所需数据。
func GetLearningActivity(ctx context.Context, requestedYear int) (models.LearningActivityResponse, error) {
	now := time.Now()
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
	repos, err := repositories(ctx)
	if err != nil {
		return result, err
	}
	events, err := repos.Activity.List(ctx, startDate, endDate)
	if err != nil {
		return result, err
	}
	counts := map[string]int{}
	for _, event := range events {
		amount := event.Value
		if amount <= 0 {
			amount = 1
		}
		counts[event.Date] += amount
		result.Total += amount
	}
	for date, count := range counts {
		result.Days = append(result.Days, models.LearningActivityDay{Date: date, Count: count})
	}
	sort.Slice(result.Days, func(i, j int) bool { return result.Days[i].Date < result.Days[j].Date })
	result.ActiveDays = len(result.Days)

	allEvents, err := repos.Activity.List(ctx, time.Date(2000, time.January, 1, 0, 0, 0, 0, time.Local), now)
	if err != nil {
		return result, err
	}
	yearSet := map[int]struct{}{requestedYear: {}}
	for _, event := range allEvents {
		if len(event.Date) >= 4 {
			var year int
			if _, err := fmt.Sscanf(event.Date[:4], "%d", &year); err == nil {
				yearSet[year] = struct{}{}
			}
		}
	}
	result.AvailableYears = result.AvailableYears[:0]
	for year := range yearSet {
		result.AvailableYears = append(result.AvailableYears, year)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(result.AvailableYears)))
	return result, nil
}

// recordAutomaticActivity records desktop JSON activity while PostgreSQL's
// existing triggers keep server-side events de-duplicated at the database.
func recordAutomaticActivity(ctx context.Context, eventType, sourceKey string, value int) error {
	app, err := appFor(ctx)
	if err != nil {
		return err
	}
	if app.AuthEnabled() {
		return nil
	}
	return recordActivity(ctx, eventType, sourceKey, value)
}

func recordActivity(ctx context.Context, eventType, sourceKey string, value int) error {
	repos, err := repositories(ctx)
	if err != nil {
		return err
	}
	return repos.Activity.Record(ctx, models.ActivityEvent{Date: time.Now().Format(time.DateOnly), EventType: eventType, SourceKey: sourceKey, Value: value})
}

func normalizeDailyGoal(goal models.DailyGoalSettings) models.DailyGoalSettings {
	if goal.ReviewTarget < 0 {
		goal.ReviewTarget = 0
	}
	if goal.FocusTargetMinutes < 0 {
		goal.FocusTargetMinutes = 0
	}
	if goal.NoteTarget < 0 {
		goal.NoteTarget = 0
	}
	if goal.ReviewTarget > 100 {
		goal.ReviewTarget = 100
	}
	if goal.FocusTargetMinutes > 720 {
		goal.FocusTargetMinutes = 720
	}
	if goal.NoteTarget > 30 {
		goal.NoteTarget = 30
	}
	return goal
}

func summarizePlan(events []models.ActivityEvent, goal models.DailyGoalSettings, date string) models.DailyPlan {
	plan := models.DailyPlan{Date: date, Goal: goal}
	for _, event := range events {
		amount := event.Value
		if amount <= 0 {
			amount = 1
		}
		switch event.EventType {
		case "review", "error_review", "library_review":
			plan.ReviewsCompleted += amount
		case "focus":
			plan.FocusMinutes += amount
		case "library_create":
			plan.NotesCreated += amount
		}
	}
	return plan
}

func GetDailyPlan(ctx context.Context) (models.DailyPlan, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return models.DailyPlan{}, err
	}
	today := time.Now()
	repos, err := repositories(ctx)
	if err != nil {
		return models.DailyPlan{}, err
	}
	events, err := repos.Activity.List(ctx, today, today)
	if err != nil {
		return models.DailyPlan{}, err
	}
	return summarizePlan(events, normalizeDailyGoal(config.DailyGoal), today.Format(time.DateOnly)), nil
}

func SetDailyGoal(ctx context.Context, goal models.DailyGoalSettings) (models.DailyGoalSettings, error) {
	config, err := loadConfig(ctx)
	if err != nil {
		return models.DailyGoalSettings{}, err
	}
	config.DailyGoal = normalizeDailyGoal(goal)
	if err := saveConfig(ctx, config); err != nil {
		return models.DailyGoalSettings{}, err
	}
	return config.DailyGoal, nil
}

func RecordFocusSession(ctx context.Context, minutes int, clientKey string) (models.DailyPlan, error) {
	if minutes < 1 || minutes > 240 {
		return models.DailyPlan{}, fmt.Errorf("专注时长应在 1 至 240 分钟之间")
	}
	clientKey = strings.TrimSpace(clientKey)
	if clientKey == "" {
		return models.DailyPlan{}, fmt.Errorf("缺少专注记录标识")
	}
	day := time.Now().Format(time.DateOnly)
	if err := recordActivity(ctx, "focus", "focus:"+day+":"+clientKey, minutes); err != nil {
		return models.DailyPlan{}, err
	}
	return GetDailyPlan(ctx)
}

func GetWeeklyReport(ctx context.Context) (models.WeeklyReport, error) {
	today := time.Now()
	start := time.Date(today.Year(), today.Month(), today.Day()-6, 0, 0, 0, 0, today.Location())
	repos, err := repositories(ctx)
	if err != nil {
		return models.WeeklyReport{}, err
	}
	events, err := repos.Activity.List(ctx, start, today)
	if err != nil {
		return models.WeeklyReport{}, err
	}
	report := models.WeeklyReport{StartDate: start.Format(time.DateOnly), EndDate: today.Format(time.DateOnly), ByEventType: map[string]int{}, WeakSubjects: []string{}}
	activeDates := map[string]bool{}
	for _, event := range events {
		amount := event.Value
		if amount <= 0 {
			amount = 1
		}
		report.TotalActivity += amount
		report.ByEventType[event.EventType] += amount
		activeDates[event.Date] = true
		switch event.EventType {
		case "focus":
			report.FocusMinutes += amount
		case "review", "error_review", "library_review":
			report.Reviews += amount
		case "library_create":
			report.NotesCreated += amount
		}
	}
	report.ActiveDays = len(activeDates)
	errors, err := GetAllErrors(ctx, "", "", "", "")
	if err == nil {
		counts := map[string]int{}
		for _, item := range errors {
			if item.NextReview != "" && item.NextReview <= today.Format(time.DateOnly) {
				counts[item.Subject]++
			}
		}
		type subjectCount struct {
			subject string
			count   int
		}
		ordered := []subjectCount{}
		for subject, count := range counts {
			ordered = append(ordered, subjectCount{subject, count})
		}
		sort.Slice(ordered, func(i, j int) bool {
			if ordered[i].count == ordered[j].count {
				return ordered[i].subject < ordered[j].subject
			}
			return ordered[i].count > ordered[j].count
		})
		for _, value := range ordered {
			report.WeakSubjects = append(report.WeakSubjects, value.subject)
		}
	}
	return report, nil
}
