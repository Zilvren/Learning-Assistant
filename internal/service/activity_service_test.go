package service

import (
	"testing"

	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	"study-tracker-go/pkg/config"
)

func TestGetLearningActivityReturnsEmptyCalendarInJSONMode(t *testing.T) {
	if err := InitApp(config.Config{StorageDriver: "json", AuthEnabled: false}, jsonrepo.NewRepositories(), nil); err != nil {
		t.Fatal(err)
	}
	result, err := GetLearningActivity(background())
	if err != nil {
		t.Fatal(err)
	}
	if result.StartDate == "" || result.EndDate == "" || result.Total != 0 || result.ActiveDays != 0 || len(result.Days) != 0 {
		t.Fatalf("unexpected local activity calendar: %#v", result)
	}
}
