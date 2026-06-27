package service

import (
	"fmt"
	"strings"

	"study-tracker-go/store"
)

func GetAllSubjects() ([]string, error) {
	var subjects []string
	if err := store.LoadJSON("subjects.json", &subjects); err != nil {
		return nil, err
	}
	if subjects == nil {
		subjects = []string{}
	}

	return subjects, nil
}

func AddSubject(name string) ([]string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("科目名称不能为空")
	}

	subjects, err := GetAllSubjects()
	if err != nil {
		return nil, err
	}

	for _, s := range subjects {
		if s == name {
			return nil, fmt.Errorf("科目已存在")
		}
	}

	subjects = append(subjects, name)

	if err := store.SaveJSON("subjects.json", subjects); err != nil {
		return nil, err
	}
	return subjects, nil
}

// SubjectExists 检查科目是否存在
func SubjectExists(name string) bool {
	subjects, err := GetAllSubjects()
	if err != nil {
		return false
	}
	for _, s := range subjects {
		if s == name {
			return true
		}
	}
	return false
}

// DeleteSubject 删除一个科目
func DeleteSubject(name string) ([]string, error) {
	subjects, err := GetAllSubjects()
	if err != nil {
		return nil, err
	}

	found := false
	remaining := []string{}
	for _, s := range subjects {
		if s == name {
			found = true
			continue // 跳过这个，不加入 remaining
		}
		remaining = append(remaining, s)
	}

	if !found {
		return nil, fmt.Errorf("科目不存在")
	}

	if err := store.SaveJSON("subjects.json", remaining); err != nil {
		return nil, err
	}
	return remaining, nil
}
