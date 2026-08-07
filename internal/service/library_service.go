package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/repository"
)

func libraryRepository(ctx context.Context) (repository.LibraryRepository, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	errors, err := repos.Errors.List(ctx, repository.ErrorFilter{})
	if err != nil {
		return nil, err
	}
	subjects, err := repos.Subjects.List(ctx)
	if err != nil {
		return nil, err
	}
	if err := repos.Library.EnsureLegacy(ctx, errors, subjects); err != nil {
		return nil, err
	}
	if err := repos.Library.Cleanup(ctx, time.Now().AddDate(0, 0, -30)); err != nil {
		return nil, err
	}
	return repos.Library, nil
}

func ListLibrary(ctx context.Context, filter repository.LibraryFilter) ([]models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return nil, e
	}
	return r.List(ctx, filter)
}
func GetLibraryItem(ctx context.Context, id int64) (models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	return r.Get(ctx, id)
}
func CreateLibraryItem(ctx context.Context, req models.CreateLibraryItemRequest, content []byte) (models.LibraryItem, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return models.LibraryItem{}, fmt.Errorf("名称不能为空")
	}
	if req.Kind == "note" && req.ReviewEnabled && len(content) == 0 {
		content = []byte("## 题目\n\n\n\n## 错解\n\n\n\n## 正解\n\n\n\n## 错因\n\n")
	}
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	return r.Create(ctx, req, content)
}
func UpdateLibraryItem(ctx context.Context, id int64, req models.UpdateLibraryItemRequest) (models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	return r.Update(ctx, id, req)
}
func SaveLibraryContent(ctx context.Context, id int64, req models.SaveLibraryContentRequest) (models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	return r.SaveContent(ctx, id, []byte(req.Content), req.BaseVersion, req.Checkpoint, req.Force)
}
func ReadLibraryContent(ctx context.Context, id int64) ([]byte, models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return nil, models.LibraryItem{}, e
	}
	return r.ReadContent(ctx, id)
}
func TrashLibraryItem(ctx context.Context, id int64) error {
	r, e := libraryRepository(ctx)
	if e != nil {
		return e
	}
	return r.Trash(ctx, id)
}
func RestoreLibraryItem(ctx context.Context, id int64) (models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	return r.Restore(ctx, id)
}
func PurgeLibraryItem(ctx context.Context, id int64) error {
	r, e := libraryRepository(ctx)
	if e != nil {
		return e
	}
	// 旧错题迁移出的笔记仍保留 error_problem_id 作为兼容来源。永久删除
	// 时必须一并移除该来源，否则下一次 EnsureLegacy 会把笔记重新生成。
	trashed, e := r.List(ctx, repository.LibraryFilter{Trashed: true})
	if e != nil {
		return e
	}
	legacyErrorIDs := legacyErrorIDsInTree(id, trashed)
	if e = r.Purge(ctx, id); e != nil {
		return e
	}
	if len(legacyErrorIDs) == 0 {
		return nil
	}
	repos, e := repositories(ctx)
	if e != nil {
		return e
	}
	legacyErrors, e := repos.Errors.List(ctx, repository.ErrorFilter{})
	if e != nil {
		return e
	}
	active := make(map[int]struct{}, len(legacyErrors))
	for _, legacy := range legacyErrors {
		active[legacy.ID] = struct{}{}
	}
	for _, legacyID := range legacyErrorIDs {
		if _, ok := active[legacyID]; !ok {
			continue
		}
		if e = repos.Errors.Delete(ctx, legacyID); e != nil {
			return e
		}
	}
	return nil
}

func legacyErrorIDsInTree(rootID int64, items []models.LibraryItem) []int {
	tree := map[int64]struct{}{rootID: {}}
	for changed := true; changed; {
		changed = false
		for _, item := range items {
			if item.ParentID == nil {
				continue
			}
			if _, parentIncluded := tree[*item.ParentID]; !parentIncluded {
				continue
			}
			if _, included := tree[item.ID]; !included {
				tree[item.ID] = struct{}{}
				changed = true
			}
		}
	}
	ids := make([]int, 0)
	seen := make(map[int]struct{})
	for _, item := range items {
		if _, included := tree[item.ID]; !included || item.ErrorProblemID == nil {
			continue
		}
		legacyID := int(*item.ErrorProblemID)
		if _, duplicate := seen[legacyID]; duplicate {
			continue
		}
		seen[legacyID] = struct{}{}
		ids = append(ids, legacyID)
	}
	return ids
}
func DuplicateLibraryItem(ctx context.Context, id int64, parent *int64) (models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	return r.Duplicate(ctx, id, parent)
}
func LibraryVersions(ctx context.Context, id int64) ([]models.LibraryVersion, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return nil, e
	}
	return r.Versions(ctx, id)
}
func RestoreLibraryVersion(ctx context.Context, id, vid int64) (models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	return r.RestoreVersion(ctx, id, vid)
}

func ListLibraryTags(ctx context.Context) ([]string, error) {
	r, err := libraryRepository(ctx)
	if err != nil {
		return nil, err
	}
	return r.ListTags(ctx)
}

func DueLibraryReviews(ctx context.Context) ([]models.LibraryItem, error) {
	r, err := libraryRepository(ctx)
	if err != nil {
		return nil, err
	}
	return r.DueReviews(ctx, time.Now())
}

func ReviewLibraryNote(ctx context.Context, id int64) (models.LibraryItem, error) {
	r, err := libraryRepository(ctx)
	if err != nil {
		return models.LibraryItem{}, err
	}
	return r.Review(ctx, id, time.Now(), reviewIntervals)
}
