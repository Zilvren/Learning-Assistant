package service

import (
	"testing"

	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	"study-tracker-go/pkg/config"
)

// TestGetLearningActivityReturnsEmptyCalendarInJSONMode 在业务层中验证对应场景的行为与边界条件。
func TestGetLearningActivityReturnsEmptyCalendarInJSONMode(t *testing.T) {
	if err := InitApp(config.Config{StorageDriver: "json", AuthEnabled: false}, jsonrepo.NewRepositories(), nil); err != nil {
		t.Fatal(err)
	}
	result, err := GetLearningActivity(background(), 2025)
	if err != nil {
		t.Fatal(err)
	}
	if result.StartDate != "2025-01-01" || result.EndDate != "2025-12-31" || result.Total != 0 || result.ActiveDays != 0 || len(result.Days) != 0 || len(result.AvailableYears) != 1 || result.AvailableYears[0] != 2025 {
		t.Fatalf("unexpected local activity calendar: %#v", result)
	}
}
