package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/repository"
)

// SearchLearning searches notes, files and mistakes in one request. Results
// carry a short contextual snippet so the user can judge a match before
// opening an item.
func SearchLearning(ctx context.Context, query string) ([]models.SearchHit, error) {
	query = strings.TrimSpace(query)
	if len([]rune(query)) < 2 {
		return nil, fmt.Errorf("请输入至少两个字符进行搜索")
	}
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	library, err := repos.Library.List(ctx, repository.LibraryFilter{Query: query})
	if err != nil {
		return nil, err
	}
	result := make([]models.SearchHit, 0, len(library))
	for _, item := range library {
		hit := models.SearchHit{SourceType: "library", ID: item.ID, Title: item.Name, Subtitle: librarySubtitle(item), Tags: item.Tags, MatchField: libraryMatchField(item, query)}
		if item.Kind == "note" {
			if body, _, err := ReadLibraryContent(ctx, item.ID); err == nil {
				hit.Snippet = searchSnippet(string(body), query)
			}
		}
		result = append(result, hit)
	}
	errors, err := repos.Errors.List(ctx, repository.ErrorFilter{Keyword: query})
	if err != nil {
		return nil, err
	}
	for _, item := range errors {
		tags := append(append([]string{}, item.Tags...), item.ReasonTags...)
		result = append(result, models.SearchHit{SourceType: "error", ID: int64(item.ID), Title: item.Title, Subtitle: item.Subject, Tags: tags, MatchField: errorMatchField(item, query), Snippet: searchSnippet(strings.Join([]string{item.Question, item.Wrong, item.Correct, item.Reason}, " "), query)})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].SourceType != result[j].SourceType {
			return result[i].SourceType == "library"
		}
		return strings.Compare(result[i].Title, result[j].Title) < 0
	})
	if len(result) > 80 {
		result = result[:80]
	}
	return result, nil
}

func librarySubtitle(item models.LibraryItem) string {
	if item.Kind == "note" {
		return "笔记"
	}
	if item.Kind == "folder" {
		return "文件夹"
	}
	return item.MimeType
}
func libraryMatchField(item models.LibraryItem, query string) string {
	if strings.Contains(strings.ToLower(item.Name), strings.ToLower(query)) {
		return "标题"
	}
	for _, tag := range item.Tags {
		if strings.Contains(strings.ToLower(tag), strings.ToLower(query)) {
			return "标签"
		}
	}
	return "正文"
}
func errorMatchField(item models.ErrorProblem, query string) string {
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(item.Title), q) {
		return "标题"
	}
	if strings.Contains(strings.ToLower(item.Subject), q) {
		return "科目"
	}
	for _, tag := range append(append([]string{}, item.Tags...), item.ReasonTags...) {
		if strings.Contains(strings.ToLower(tag), q) {
			return "标签"
		}
	}
	return "内容"
}
func searchSnippet(text, query string) string {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return ""
	}
	lower, needle := strings.ToLower(text), strings.ToLower(query)
	byteIndex := strings.Index(lower, needle)
	runes := []rune(text)
	if byteIndex < 0 {
		if len(runes) > 110 {
			return string(runes[:110]) + "…"
		}
		return text
	}
	index := len([]rune(lower[:byteIndex]))
	start := index - 44
	if start < 0 {
		start = 0
	}
	end := index + len([]rune(query)) + 70
	if end > len(runes) {
		end = len(runes)
	}
	prefix, suffix := "", ""
	if start > 0 {
		prefix = "…"
	}
	if end < len(runes) {
		suffix = "…"
	}
	return prefix + string(runes[start:end]) + suffix
}
