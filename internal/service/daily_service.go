package service

import (
	"context"
	"math/rand"
	"sort"
	"time"

	models "study-tracker-go/internal/model"
)

// 知识点库（硬编码默认值，如果 knowledge.json 不存在就用这个）
var defaultKnowledge = map[string][]string{
	"数学": {"未整理"},
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GetDailyPush 生成每日推送数据
func GetDailyPush(ctx context.Context) (models.DailyPushResult, error) {
	errors, err := GetAllErrors(ctx, "", "", "", "")
	if err != nil {
		return models.DailyPushResult{}, err
	}
	subjects, err := GetAllSubjects(ctx)
	if err != nil {
		return models.DailyPushResult{}, err
	}

	today := time.Now().Format("2006-01-02")
	due := []models.ErrorProblem{}
	overdue := 0
	reviewed := 0

	for _, item := range errors {
		if item.ReviewCount > 0 {
			reviewed++
		}
		next := item.NextReview
		if next == "" {
			next = today
		}
		if next <= today {
			due = append(due, item)
		}
		if next < today {
			overdue++
		}
	}

	// 按到期日排序
	sort.Slice(due, func(i, j int) bool {
		if due[i].NextReview == due[j].NextReview {
			return due[i].ID < due[j].ID
		}
		return due[i].NextReview < due[j].NextReview
	})

	// 每个科目随机选一条知识点
	knowledgeBase := getKnowledgeBase(ctx)
	knowledge := map[string]string{}
	for _, subject := range subjects {
		tips := knowledgeBase[subject]
		if len(tips) > 0 {
			knowledge[subject] = tips[rand.Intn(len(tips))]
		}
	}

	// 建议文案
	advice := "今天没有到期错题，可以新增整理或轻量回看近期内容"
	if len(due) > 0 {
		advice = "今天有到期错题，建议先清空复习队列"
	}
	if overdue > 0 {
		advice = "今天有逾期错题，建议优先处理最早到期的题目"
	}

	return models.DailyPushResult{
		Date:         today,
		TotalErrors:  len(errors),
		Reviewed:     reviewed,
		DueCount:     len(due),
		OverdueCount: overdue,
		Knowledge:    knowledge,
		WeakErrors:   due,
		Advice:       advice,
	}, nil
}

func getKnowledgeBase(ctx context.Context) map[string][]string {
	repos, err := repositories(ctx)
	if err != nil {
		return defaultKnowledge
	}
	kb, err := repos.Knowledge.Load(ctx)
	if err != nil {
		return defaultKnowledge
	}
	if kb == nil {
		return defaultKnowledge
	}
	return kb
}
