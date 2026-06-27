package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"study-tracker-go/models"
	"study-tracker-go/store"
)

func CreateError(req models.AddErrorRequest) (models.ErrorProblem, error) {

	req.Subject = strings.TrimSpace(req.Subject)
	req.Question = strings.TrimSpace(req.Question)

	if !SubjectExists(req.Subject) {
		return models.ErrorProblem{}, fmt.Errorf("无效科目")
	}

	if req.Question == "" {
		return models.ErrorProblem{}, fmt.Errorf("题目不能为空")
	}

	//处理默认值
	if req.Wrong == "" {
		req.Wrong = "未记录"
	}
	if req.Correct == "" {
		req.Correct = "未记录"
	}
	if req.Reason == "" {
		req.Reason = "未记录"
	}

	if req.Title == "" {
		req.Title = firstRunes(req.Question, 40) //默认取前40个字符
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}
	if req.ReasonTags == nil {
		req.ReasonTags = []string{}
	}

	// 读取已有错题，计算新 ID
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return models.ErrorProblem{}, err
	}

	nextID := 1
	for _, item := range errors {
		if item.ID >= nextID {
			nextID = item.ID + 1
		}
	}

	//构造新错题
	now := time.Now()

	item := models.ErrorProblem{
		ID:          nextID,
		Subject:     req.Subject,
		Title:       req.Title,
		Question:    req.Question,
		Wrong:       req.Wrong,
		Correct:     req.Correct,
		Reason:      req.Reason,
		Tags:        req.Tags,
		ReasonTags:  req.ReasonTags,
		Created:     now.Format("2006-01-02 15:04:05"),
		ReviewCount: 0,
		LastReview:  nil, // 还没复习过
		ReviewStage: 0,
		NextReview:  now.Format("2006-01-02"),
	}

	errors = append(errors, item)

	if err := store.SaveJSON("errors.json", errors); err != nil {
		return models.ErrorProblem{}, err
	}
	return item, nil
}

func firstRunes(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max])
}

func GetAllErrors(subject, keyword, tag, reasonTag string) ([]models.ErrorProblem, error) {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return nil, err
	}

	if errors == nil {
		return []models.ErrorProblem{}, nil
	}

	result := []models.ErrorProblem{}
	for _, item := range errors {
		// 补全缺失的默认值
		normalizeReviewFields(&item)

		// 逐项筛选
		if subject != "" && subject != "全部" && item.Subject != subject {
			continue
		}
		if keyword != "" && !matchesKeyword(item, keyword) {
			continue
		}
		if tag != "" && !listContainsFold(item.Tags, tag) {
			continue
		}
		if reasonTag != "" && !listContainsFold(item.ReasonTags, reasonTag) {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}
func normalizeReviewFields(item *models.ErrorProblem) {
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.ReasonTags == nil {
		item.ReasonTags = []string{}
	}
	if item.NextReview == "" {
		if len(item.Created) >= 10 {
			item.NextReview = item.Created[:10]
		} else {
			item.NextReview = time.Now().Format("2006-01-02")
		}
	}
}

// matchesKeyword 检查错题是否匹配关键词
func matchesKeyword(item models.ErrorProblem, keyword string) bool {
	keyword = strings.ToLower(keyword)
	if strings.Contains(strings.ToLower(item.Question), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Title), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Reason), keyword) {
		return true
	}
	return listContainsFold(item.Tags, keyword) || listContainsFold(item.ReasonTags, keyword)
}

// listContainsFold 检查字符串切片是否包含关键词（忽略大小写）
func listContainsFold(list []string, keyword string) bool {
	keyword = strings.ToLower(keyword)
	for _, item := range list {
		if strings.Contains(strings.ToLower(item), keyword) {
			return true
		}
	}
	return false
}

func UpdateError(id int, req models.UpdateErrorRequest) error {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return err
	}
	for i := range errors {
		if id != errors[i].ID {
			continue
		}

		if req.Subject != nil {
			if !SubjectExists(*req.Subject) {
				return fmt.Errorf("无效科目")
			}
			errors[i].Subject = *req.Subject
		}
		if req.Title != nil {
			errors[i].Title = *req.Title
		}
		if req.Question != nil {
			if strings.TrimSpace(*req.Question) == "" {
				return fmt.Errorf("题目不能为空")
			}
			errors[i].Question = *req.Question
		}
		if req.Wrong != nil {
			errors[i].Wrong = *req.Wrong
		}
		if req.Correct != nil {
			errors[i].Correct = *req.Correct
		}
		if req.Reason != nil {
			errors[i].Reason = *req.Reason
		}
		if req.Tags != nil {
			errors[i].Tags = *req.Tags
		}
		if req.ReasonTags != nil {
			errors[i].ReasonTags = *req.ReasonTags
		}

		return store.SaveJSON("errors.json", errors)
	}
	return fmt.Errorf("未找到错题 #%d", id)
}

// DeleteError 删除一条错题
func DeleteError(id int) error {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return err
	}

	found := false
	remaining := []models.ErrorProblem{}
	for _, item := range errors {
		if item.ID == id {
			found = true
			continue
		}
		remaining = append(remaining, item)
	}

	if !found {
		return fmt.Errorf("未找到错题 #%d", id)
	}

	return store.SaveJSON("errors.json", remaining)
}

// 艾宾浩斯复习间隔（天数）
var reviewIntervals = []int{0, 1, 2, 4, 7, 15}

// ReviewError 标记复习一条错题
func ReviewError(id int) (models.ErrorProblem, error) {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return models.ErrorProblem{}, err
	}

	for i := range errors {
		if errors[i].ID != id {
			continue
		}

		nowText := time.Now().Format("2006-01-02 15:04:05")
		errors[i].ReviewCount++
		errors[i].LastReview = &nowText // &取地址，因为 LastReview 是 *string
		errors[i].ReviewStage = errors[i].ReviewCount
		errors[i].NextReview = nextReviewDate(errors[i].ReviewCount)

		if err := store.SaveJSON("errors.json", errors); err != nil {
			return models.ErrorProblem{}, err
		}
		return errors[i], nil
	}
	return models.ErrorProblem{}, fmt.Errorf("未找到错题 #%d", id)
}

func nextReviewDate(reviewCount int) string {
	index := reviewCount
	if index < 0 {
		index = 0
	}
	if index >= len(reviewIntervals) {
		index = len(reviewIntervals) - 1
	}
	return time.Now().AddDate(0, 0, reviewIntervals[index]).Format("2006-01-02")
}

// GetAllTags 获取所有标签（从错题中提取，去重排序）
func GetAllTags() ([]string, error) {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	for _, item := range errors {
		for _, tag := range item.Tags {
			seen[tag] = true
		}
		for _, tag := range item.ReasonTags {
			seen[tag] = true
		}
	}

	tags := []string{}
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags) // 排序，前端显示更稳定
	return tags, nil
}
