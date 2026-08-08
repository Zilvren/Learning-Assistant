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
	// Browsing is strictly read-only. Legacy error-to-note migration used to run
	// here and could recreate a note after the user had permanently deleted it.
	// Backup/import is now the explicit migration path for both storage modes.
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
	// A library note may retain an error_problem_id for legacy compatibility,
	// but it is not ownership. Permanently deleting a note must never delete
	// the user's source error record.
	return r.Purge(ctx, id)
}

func BatchLibraryItems(ctx context.Context, action string, ids []int64, parentID *int64) error {
	r, err := libraryRepository(ctx)
	if err != nil {
		return err
	}
	return r.Batch(ctx, action, ids, parentID)
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
