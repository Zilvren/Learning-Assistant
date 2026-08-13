package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/repository"
)

// libraryRepository 在业务层中完成本文件定义的局部处理。
func libraryRepository(ctx context.Context) (repository.LibraryRepository, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	// Browsing is strictly read-only. Legacy error-to-note migration used to run
	// here and could recreate a note after the user had permanently deleted it.
	// Backup/import is now the explicit migration path for both storage modes.
	if err := repos.Library.Cleanup(ctx, time.Now().AddDate(0, 0, -30)); err != nil {
		return nil, err
	}
	return repos.Library, nil
}

// ListLibrary 在业务层中读取并整理所需数据。
func ListLibrary(ctx context.Context, filter repository.LibraryFilter) ([]models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return nil, e
	}
	return r.List(ctx, filter)
}

// GetLibraryItem 在业务层中读取并整理所需数据。
func GetLibraryItem(ctx context.Context, id int64) (models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	return r.Get(ctx, id)
}

// CreateLibraryItem 在业务层中创建或更新相应状态。
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
	item, err := r.Create(ctx, req, content)
	if err == nil && req.Kind != "folder" {
		_ = recordAutomaticActivity(ctx, "library_create", fmt.Sprintf("library:create:%d:%s", item.ID, time.Now().Format(time.DateOnly)), 1)
	}
	return item, err
}

// UpdateLibraryItem 在业务层中创建或更新相应状态。
func UpdateLibraryItem(ctx context.Context, id int64, req models.UpdateLibraryItemRequest) (models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	item, err := r.Update(ctx, id, req)
	if err == nil {
		_ = recordAutomaticActivity(ctx, "library_update", fmt.Sprintf("library:update:%d:%s", id, time.Now().Format(time.DateOnly)), 1)
	}
	return item, err
}

// SaveLibraryContent 在业务层中创建或更新相应状态。
func SaveLibraryContent(ctx context.Context, id int64, req models.SaveLibraryContentRequest) (models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	item, err := r.SaveContent(ctx, id, []byte(req.Content), req.BaseVersion, req.Checkpoint, req.Force)
	if err == nil {
		_ = recordAutomaticActivity(ctx, "library_update", fmt.Sprintf("library:update:%d:%s", id, time.Now().Format(time.DateOnly)), 1)
	}
	return item, err
}

// ReadLibraryContent 在业务层中读取并整理所需数据。
func ReadLibraryContent(ctx context.Context, id int64) ([]byte, models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return nil, models.LibraryItem{}, e
	}
	return r.ReadContent(ctx, id)
}

// TrashLibraryItem 在业务层中删除、清理或撤销相应状态。
func TrashLibraryItem(ctx context.Context, id int64) error {
	r, e := libraryRepository(ctx)
	if e != nil {
		return e
	}
	return r.Trash(ctx, id)
}

// RestoreLibraryItem 在业务层中完成本文件定义的局部处理。
func RestoreLibraryItem(ctx context.Context, id int64) (models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	return r.Restore(ctx, id)
}

// PurgeLibraryItem 在业务层中删除、清理或撤销相应状态。
func PurgeLibraryItem(ctx context.Context, id int64) error {
	r, e := libraryRepository(ctx)
	if e != nil {
		return e
	}
	// A library note may retain an error_problem_id for legacy compatibility,
	// but it is not ownership. Permanently deleting a note must never delete
	// the user's source error record.
	return r.Purge(ctx, id)
}

// BatchLibraryItems 在业务层中完成本文件定义的局部处理。
func BatchLibraryItems(ctx context.Context, action string, ids []int64, parentID *int64) error {
	r, err := libraryRepository(ctx)
	if err != nil {
		return err
	}
	return r.Batch(ctx, action, ids, parentID)
}

// DuplicateLibraryItem 在业务层中完成本文件定义的局部处理。
func DuplicateLibraryItem(ctx context.Context, id int64, parent *int64) (models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	return r.Duplicate(ctx, id, parent)
}

// LibraryVersions 在业务层中完成本文件定义的局部处理。
func LibraryVersions(ctx context.Context, id int64) ([]models.LibraryVersion, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return nil, e
	}
	return r.Versions(ctx, id)
}

// RestoreLibraryVersion 在业务层中完成本文件定义的局部处理。
func RestoreLibraryVersion(ctx context.Context, id, vid int64) (models.LibraryItem, error) {
	r, e := libraryRepository(ctx)
	if e != nil {
		return models.LibraryItem{}, e
	}
	return r.RestoreVersion(ctx, id, vid)
}

// ListLibraryTags 在业务层中读取并整理所需数据。
func ListLibraryTags(ctx context.Context) ([]string, error) {
	r, err := libraryRepository(ctx)
	if err != nil {
		return nil, err
	}
	return r.ListTags(ctx)
}

// DueLibraryReviews 在业务层中完成本文件定义的局部处理。
func DueLibraryReviews(ctx context.Context) ([]models.LibraryItem, error) {
	r, err := libraryRepository(ctx)
	if err != nil {
		return nil, err
	}
	return r.DueReviews(ctx, time.Now())
}

// ReviewLibraryNote 在业务层中完成本文件定义的局部处理。
func ReviewLibraryNote(ctx context.Context, id int64, ratings ...string) (models.LibraryItem, error) {
	r, err := libraryRepository(ctx)
	if err != nil {
		return models.LibraryItem{}, err
	}
	rating := "good"
	if len(ratings) > 0 {
		rating = models.NormalizeReviewRating(ratings[0])
	}
	item, err := r.ReviewWithRating(ctx, id, time.Now(), rating)
	if err == nil {
		_ = recordAutomaticActivity(ctx, "review", fmt.Sprintf("review:library:%d:%s", id, time.Now().Format(time.DateOnly)), 1)
	}
	return item, err
}
