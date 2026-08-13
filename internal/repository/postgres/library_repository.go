package postgres

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type LibraryRepository struct{ store *Store }

const libraryColumns = `id,parent_id,original_parent_id,kind,name,mime_type,file_size,tags,pinned,current_version,error_problem_id,blob_hash,review_enabled,review_count,review_stage,last_review,next_review,created_at,updated_at,deleted_at`

// scanLibrary 在存储层中完成本文件定义的局部处理。
func scanLibrary(row pgx.Row) (models.LibraryItem, error) {
	var x models.LibraryItem
	err := row.Scan(&x.ID, &x.ParentID, &x.OriginalParent, &x.Kind, &x.Name, &x.MimeType, &x.Size, &x.Tags, &x.Pinned, &x.CurrentVersion, &x.ErrorProblemID, &x.BlobHash, &x.ReviewEnabled, &x.ReviewCount, &x.ReviewStage, &x.LastReview, &x.NextReview, &x.CreatedAt, &x.UpdatedAt, &x.DeletedAt)
	return x, err
}

// scanLibraryRows 在存储层中完成本文件定义的局部处理。
func scanLibraryRows(rows pgx.Rows) (models.LibraryItem, error) {
	var x models.LibraryItem
	err := rows.Scan(&x.ID, &x.ParentID, &x.OriginalParent, &x.Kind, &x.Name, &x.MimeType, &x.Size, &x.Tags, &x.Pinned, &x.CurrentVersion, &x.ErrorProblemID, &x.BlobHash, &x.ReviewEnabled, &x.ReviewCount, &x.ReviewStage, &x.LastReview, &x.NextReview, &x.CreatedAt, &x.UpdatedAt, &x.DeletedAt)
	return x, err
}

// List 在存储层中读取并整理所需数据。
func (r *LibraryRepository) List(ctx context.Context, f base.LibraryFilter) ([]models.LibraryItem, error) {
	query := strings.ToLower(strings.TrimSpace(f.Query))
	args := []any{r.store.userID}
	where := []string{"user_id=$1"}
	if f.Trashed {
		where = append(where, "deleted_at IS NOT NULL")
		if f.ParentID == nil {
			where = append(where, `NOT EXISTS (
				SELECT 1
				FROM library_items parent
				WHERE parent.user_id = library_items.user_id
				  AND parent.id = library_items.parent_id
				  AND parent.deleted_at IS NOT NULL
			)`)
		}
	} else {
		where = append(where, "deleted_at IS NULL")
	}
	if f.ParentID != nil {
		args = append(args, *f.ParentID)
		where = append(where, fmt.Sprintf("parent_id=$%d", len(args)))
	} else if query == "" && !f.Trashed && !f.ReviewOnly {
		where = append(where, "parent_id IS NULL")
	}
	if f.Kind != "" && f.Kind != "all" {
		args = append(args, f.Kind)
		where = append(where, fmt.Sprintf("kind=$%d", len(args)))
	}
	if strings.TrimSpace(f.Tag) != "" {
		args = append(args, strings.TrimSpace(f.Tag))
		where = append(where, fmt.Sprintf("EXISTS(SELECT 1 FROM unnest(tags) t WHERE lower(t)=lower($%d))", len(args)))
	}
	if f.ReviewOnly {
		where = append(where, "review_enabled=TRUE")
	}
	if f.DueOnly {
		where = append(where, "review_enabled=TRUE AND (next_review='' OR next_review<=CURRENT_DATE::text)")
	}
	if query != "" {
		args = append(args, query)
		parameter := len(args)
		where = append(where, fmt.Sprintf(`(
			lower(name) LIKE '%%' || $%d || '%%'
			OR EXISTS(SELECT 1 FROM unnest(tags) tag WHERE lower(tag) LIKE '%%' || $%d || '%%')
			OR (kind = 'note' AND (
				to_tsvector('simple', coalesce(search_text, '')) @@ plainto_tsquery('simple', $%d)
				OR lower(search_text) LIKE '%%' || $%d || '%%'
			))
		)`, parameter, parameter, parameter, parameter))
	}
	rows, err := r.store.pool.Query(ctx, "SELECT "+libraryColumns+" FROM library_items WHERE "+strings.Join(where, " AND ")+" ORDER BY pinned DESC,updated_at DESC", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.LibraryItem{}
	for rows.Next() {
		x, e := scanLibraryRows(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// postgresLibraryMatchesQuery remains a small compatibility helper for
// callers that only have a LibraryItem. List now performs this matching in
// PostgreSQL through search_text, avoiding per-note blob reads.
func postgresLibraryMatchesQuery(item models.LibraryItem, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || strings.Contains(strings.ToLower(item.Name+" "+strings.Join(item.Tags, " ")), query) {
		return true
	}
	if item.Kind != "note" || item.BlobHash == "" {
		return false
	}
	body, err := base.ReadBlob(item.BlobHash)
	return err == nil && strings.Contains(strings.ToLower(string(body)), query)
}

// Get 在存储层中读取并整理所需数据。
func (r *LibraryRepository) Get(ctx context.Context, id int64) (models.LibraryItem, error) {
	x, e := scanLibrary(r.store.pool.QueryRow(ctx, "SELECT "+libraryColumns+" FROM library_items WHERE user_id=$1 AND id=$2", r.store.userID, id))
	if e == pgx.ErrNoRows {
		return x, fmt.Errorf("资料不存在")
	}
	return x, e
}

// Create 在存储层中创建或更新相应状态。
func (r *LibraryRepository) Create(ctx context.Context, req models.CreateLibraryItemRequest, content []byte) (models.LibraryItem, error) {
	var out models.LibraryItem
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return out, fmt.Errorf("名称不能为空")
	}
	hash := ""
	size := int64(0)
	var err error
	if req.Kind != "folder" {
		hash, size, err = base.StoreBlob(bytes.NewReader(content))
		if err != nil {
			return out, err
		}
	}
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	name, err = r.uniqueName(ctx, tx, req.ParentID, name, 0)
	if err != nil {
		return out, err
	}
	nextReview := ""
	if req.ReviewEnabled {
		nextReview = time.Now().Format("2006-01-02")
	}
	tags := normalizeLibraryTags(req.Tags)
	searchText := ""
	if req.Kind == "note" {
		searchText = string(content)
	}
	out, err = scanLibrary(tx.QueryRow(ctx, "INSERT INTO library_items(user_id,parent_id,kind,name,mime_type,file_size,tags,current_version,blob_hash,review_enabled,next_review,error_problem_id,search_text) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING "+libraryColumns, r.store.userID, req.ParentID, req.Kind, name, req.MimeType, size, tags, boolInt(req.Kind != "folder"), hash, req.ReviewEnabled, nextReview, req.ErrorProblemID, searchText))
	if err != nil {
		return out, err
	}
	if req.Kind != "folder" {
		_, err = tx.Exec(ctx, "INSERT INTO library_versions(user_id,item_id,version,blob_hash,file_size) VALUES($1,$2,1,$3,$4)", r.store.userID, out.ID, hash, size)
		if err != nil {
			return out, err
		}
	}
	return out, tx.Commit(ctx)
}

// boolInt 在存储层中完成本文件定义的局部处理。
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// normalizeLibraryTags 在存储层中构造、编码或标准化数据。
func normalizeLibraryTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		key := strings.ToLower(tag)
		if tag == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tag)
	}
	return result
}

// Update 在存储层中创建或更新相应状态。
func (r *LibraryRepository) Update(ctx context.Context, id int64, req models.UpdateLibraryItemRequest) (models.LibraryItem, error) {
	item, err := r.Get(ctx, id)
	if err != nil {
		return item, err
	}
	if req.Name != nil {
		item.Name = strings.TrimSpace(*req.Name)
	}
	if req.Tags != nil {
		item.Tags = normalizeLibraryTags(*req.Tags)
	}
	if req.Pinned != nil {
		item.Pinned = *req.Pinned
	}
	if req.ReviewEnabled != nil {
		item.ReviewEnabled = *req.ReviewEnabled
		if item.ReviewEnabled && item.NextReview == "" {
			item.NextReview = time.Now().Format("2006-01-02")
		}
	}
	if req.ParentSet || req.ParentID != nil {
		if req.ParentID == nil {
			item.ParentID = nil
		} else {
			var invalid bool
			err = r.store.pool.QueryRow(ctx, `WITH RECURSIVE tree AS (SELECT id FROM library_items WHERE user_id=$1 AND id=$2 UNION SELECT c.id FROM library_items c JOIN tree p ON c.parent_id=p.id WHERE c.user_id=$1) SELECT NOT EXISTS(SELECT 1 FROM library_items WHERE user_id=$1 AND id=$3 AND kind='folder' AND deleted_at IS NULL) OR EXISTS(SELECT 1 FROM tree WHERE id=$3)`, r.store.userID, id, *req.ParentID).Scan(&invalid)
			if err != nil {
				return item, err
			}
			if invalid {
				return item, fmt.Errorf("不能移动到自身或子文件夹")
			}
			item.ParentID = req.ParentID
		}
	}
	item.Name, err = r.uniqueName(ctx, r.store.pool, item.ParentID, item.Name, id)
	if err != nil {
		return item, err
	}
	return scanLibrary(r.store.pool.QueryRow(ctx, "UPDATE library_items SET parent_id=$3,name=$4,tags=$5,pinned=$6,review_enabled=$7,next_review=$8,updated_at=now() WHERE user_id=$1 AND id=$2 RETURNING "+libraryColumns, r.store.userID, id, item.ParentID, item.Name, item.Tags, item.Pinned, item.ReviewEnabled, item.NextReview))
}

// SaveContent 在存储层中创建或更新相应状态。
func (r *LibraryRepository) SaveContent(ctx context.Context, id int64, content []byte, baseVersion int, checkpoint, force bool) (models.LibraryItem, error) {
	hash, size, err := base.StoreBlob(bytes.NewReader(content))
	if err != nil {
		return models.LibraryItem{}, err
	}
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return models.LibraryItem{}, err
	}
	defer tx.Rollback(ctx)
	item, err := scanLibrary(tx.QueryRow(ctx, "SELECT "+libraryColumns+" FROM library_items WHERE user_id=$1 AND id=$2 FOR UPDATE", r.store.userID, id))
	if err != nil {
		return item, err
	}
	if !force && baseVersion != item.CurrentVersion {
		return item, fmt.Errorf("版本冲突")
	}
	if hash == item.BlobHash {
		if checkpoint {
			_, err = tx.Exec(ctx, "INSERT INTO library_versions(user_id,item_id,version,blob_hash,file_size) VALUES($1,$2,$3,$4,$5) ON CONFLICT(item_id,version) DO NOTHING", r.store.userID, id, item.CurrentVersion, hash, size)
			if err != nil {
				return item, err
			}
			return item, tx.Commit(ctx)
		}
		return item, nil
	}
	searchText := ""
	if item.Kind == "note" {
		searchText = string(content)
	}
	item, err = scanLibrary(tx.QueryRow(ctx, "UPDATE library_items SET blob_hash=$3,file_size=$4,current_version=current_version+1,search_text=$5,updated_at=now() WHERE user_id=$1 AND id=$2 RETURNING "+libraryColumns, r.store.userID, id, hash, size, searchText))
	if err != nil {
		return item, err
	}
	if checkpoint {
		_, err = tx.Exec(ctx, "INSERT INTO library_versions(user_id,item_id,version,blob_hash,file_size) VALUES($1,$2,$3,$4,$5)", r.store.userID, id, item.CurrentVersion, hash, size)
		if err != nil {
			return item, err
		}
		_, err = tx.Exec(ctx, "DELETE FROM library_versions WHERE id IN (SELECT id FROM library_versions WHERE user_id=$1 AND item_id=$2 ORDER BY version DESC OFFSET 50)", r.store.userID, id)
		if err != nil {
			return item, err
		}
	}
	return item, tx.Commit(ctx)
}

// ReadContent 在存储层中读取并整理所需数据。
func (r *LibraryRepository) ReadContent(ctx context.Context, id int64) ([]byte, models.LibraryItem, error) {
	item, err := r.Get(ctx, id)
	if err != nil {
		return nil, item, err
	}
	if item.BlobHash == "" {
		return []byte{}, item, nil
	}
	b, err := base.ReadBlob(item.BlobHash)
	return b, item, err
}

// Trash 在存储层中删除、清理或撤销相应状态。
func (r *LibraryRepository) Trash(ctx context.Context, id int64) error {
	_, err := r.store.pool.Exec(ctx, "WITH RECURSIVE tree AS (SELECT id FROM library_items WHERE user_id=$1 AND id=$2 UNION ALL SELECT c.id FROM library_items c JOIN tree p ON c.parent_id=p.id WHERE c.user_id=$1) UPDATE library_items SET original_parent_id=parent_id,deleted_at=now() WHERE id IN(SELECT id FROM tree)", r.store.userID, id)
	return err
}

// Restore 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) Restore(ctx context.Context, id int64) (models.LibraryItem, error) {
	_, err := r.store.pool.Exec(ctx, "WITH RECURSIVE tree AS (SELECT id FROM library_items WHERE user_id=$1 AND id=$2 UNION ALL SELECT c.id FROM library_items c JOIN tree p ON c.parent_id=p.id WHERE c.user_id=$1) UPDATE library_items SET parent_id=COALESCE(original_parent_id,parent_id),original_parent_id=NULL,deleted_at=NULL,updated_at=now() WHERE id IN(SELECT id FROM tree)", r.store.userID, id)
	if err != nil {
		return models.LibraryItem{}, err
	}
	return r.Get(ctx, id)
}

// Purge 在存储层中删除、清理或撤销相应状态。
func (r *LibraryRepository) Purge(ctx context.Context, id int64) error {
	_, err := r.store.pool.Exec(ctx, "WITH RECURSIVE tree AS (SELECT id FROM library_items WHERE user_id=$1 AND id=$2 AND deleted_at IS NOT NULL UNION ALL SELECT c.id FROM library_items c JOIN tree p ON c.parent_id=p.id WHERE c.user_id=$1) DELETE FROM library_items WHERE id IN(SELECT id FROM tree)", r.store.userID, id)
	return err
}

type batchLibraryItem struct {
	ID        int64
	ParentID  *int64
	Kind      string
	DeletedAt *time.Time
}

// Batch performs all validation and mutations inside one database transaction.
// A failed member therefore never leaves a partially moved, restored, or
// deleted selection behind.
// Batch 在一个 PostgreSQL 事务中批量移动、恢复或删除资料库条目。
func (r *LibraryRepository) Batch(ctx context.Context, action string, ids []int64, parentID *int64) error {
	ids = uniqueLibraryIDs(ids)
	if len(ids) == 0 {
		return fmt.Errorf("至少选择一项资料")
	}
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, "SELECT id,parent_id,kind,deleted_at FROM library_items WHERE user_id=$1 AND id=ANY($2) FOR UPDATE", r.store.userID, ids)
	if err != nil {
		return err
	}
	items := make(map[int64]batchLibraryItem, len(ids))
	for rows.Next() {
		var item batchLibraryItem
		if err = rows.Scan(&item.ID, &item.ParentID, &item.Kind, &item.DeletedAt); err != nil {
			rows.Close()
			return err
		}
		items[item.ID] = item
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(items) != len(ids) {
		return fmt.Errorf("所选资料不存在或无权操作")
	}

	roots := batchLibraryRoots(ids, items)
	switch action {
	case "trash":
		for _, id := range roots {
			if items[id].DeletedAt != nil {
				return fmt.Errorf("所选资料已在回收站中")
			}
		}
		_, err = tx.Exec(ctx, `WITH RECURSIVE tree AS (
			SELECT id FROM library_items WHERE user_id=$1 AND id=ANY($2)
			UNION
			SELECT child.id FROM library_items child JOIN tree parent ON child.parent_id=parent.id WHERE child.user_id=$1
		) UPDATE library_items SET original_parent_id=COALESCE(original_parent_id,parent_id), deleted_at=now(), updated_at=now() WHERE user_id=$1 AND id IN(SELECT id FROM tree)`, r.store.userID, roots)
	case "restore":
		for _, id := range roots {
			if items[id].DeletedAt == nil {
				return fmt.Errorf("所选资料不在回收站中")
			}
		}
		_, err = tx.Exec(ctx, `WITH RECURSIVE tree AS (
			SELECT id FROM library_items WHERE user_id=$1 AND id=ANY($2)
			UNION
			SELECT child.id FROM library_items child JOIN tree parent ON child.parent_id=parent.id WHERE child.user_id=$1
		) UPDATE library_items SET parent_id=COALESCE(original_parent_id,parent_id), original_parent_id=NULL, deleted_at=NULL, updated_at=now() WHERE user_id=$1 AND id IN(SELECT id FROM tree)`, r.store.userID, roots)
	case "purge":
		for _, id := range roots {
			if items[id].DeletedAt == nil {
				return fmt.Errorf("只能永久删除回收站中的资料")
			}
		}
		_, err = tx.Exec(ctx, `WITH RECURSIVE tree AS (
			SELECT id FROM library_items WHERE user_id=$1 AND id=ANY($2) AND deleted_at IS NOT NULL
			UNION
			SELECT child.id FROM library_items child JOIN tree parent ON child.parent_id=parent.id WHERE child.user_id=$1
		) DELETE FROM library_items WHERE user_id=$1 AND id IN(SELECT id FROM tree)`, r.store.userID, roots)
	case "move":
		for _, id := range roots {
			if items[id].DeletedAt != nil {
				return fmt.Errorf("不能移动回收站中的资料")
			}
		}
		if parentID != nil {
			var targetIsFolder bool
			if err = tx.QueryRow(ctx, "SELECT kind='folder' AND deleted_at IS NULL FROM library_items WHERE user_id=$1 AND id=$2", r.store.userID, *parentID).Scan(&targetIsFolder); err != nil || !targetIsFolder {
				if err == pgx.ErrNoRows {
					return fmt.Errorf("目标文件夹不存在")
				}
				if err != nil {
					return err
				}
				return fmt.Errorf("目标不是可用文件夹")
			}
			var intoOwnTree bool
			err = tx.QueryRow(ctx, `WITH RECURSIVE tree AS (
				SELECT id FROM library_items WHERE user_id=$1 AND id=ANY($2)
				UNION
				SELECT child.id FROM library_items child JOIN tree parent ON child.parent_id=parent.id WHERE child.user_id=$1
			) SELECT EXISTS(SELECT 1 FROM tree WHERE id=$3)`, r.store.userID, roots, *parentID).Scan(&intoOwnTree)
			if err != nil {
				return err
			}
			if intoOwnTree {
				return fmt.Errorf("不能移动到自身或子文件夹")
			}
		}
		for _, id := range roots {
			item, readErr := scanLibrary(tx.QueryRow(ctx, "SELECT "+libraryColumns+" FROM library_items WHERE user_id=$1 AND id=$2 FOR UPDATE", r.store.userID, id))
			if readErr != nil {
				return readErr
			}
			name, nameErr := r.uniqueName(ctx, tx, parentID, item.Name, id)
			if nameErr != nil {
				return nameErr
			}
			if _, err = tx.Exec(ctx, "UPDATE library_items SET parent_id=$3,name=$4,updated_at=now() WHERE user_id=$1 AND id=$2", r.store.userID, id, parentID, name); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("不支持的批量操作")
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// uniqueLibraryIDs 在存储层中完成本文件定义的局部处理。
func uniqueLibraryIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// batchLibraryRoots 在存储层中完成本文件定义的局部处理。
func batchLibraryRoots(ids []int64, items map[int64]batchLibraryItem) []int64 {
	roots := make([]int64, 0, len(ids))
	for _, id := range ids {
		isChild := false
		for parentID := items[id].ParentID; parentID != nil; {
			parent, selected := items[*parentID]
			if !selected {
				break
			}
			isChild = true
			parentID = parent.ParentID
		}
		if !isChild {
			roots = append(roots, id)
		}
	}
	return roots
}

// Duplicate 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) Duplicate(ctx context.Context, id int64, parent *int64) (models.LibraryItem, error) {
	b, x, e := r.ReadContent(ctx, id)
	if e != nil {
		return x, e
	}
	return r.Create(ctx, models.CreateLibraryItemRequest{ParentID: parent, Kind: x.Kind, Name: x.Name, MimeType: x.MimeType, Tags: x.Tags, ReviewEnabled: x.ReviewEnabled}, b)
}

// Versions 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) Versions(ctx context.Context, id int64) ([]models.LibraryVersion, error) {
	rows, e := r.store.pool.Query(ctx, "SELECT id,item_id,version,blob_hash,file_size,created_at FROM library_versions WHERE user_id=$1 AND item_id=$2 ORDER BY version DESC", r.store.userID, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []models.LibraryVersion{}
	for rows.Next() {
		var v models.LibraryVersion
		if e = rows.Scan(&v.ID, &v.ItemID, &v.Version, &v.BlobHash, &v.Size, &v.CreatedAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// RestoreVersion 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) RestoreVersion(ctx context.Context, id, vid int64) (models.LibraryItem, error) {
	var hash string
	e := r.store.pool.QueryRow(ctx, "SELECT blob_hash FROM library_versions WHERE user_id=$1 AND item_id=$2 AND id=$3", r.store.userID, id, vid).Scan(&hash)
	if e != nil {
		return models.LibraryItem{}, e
	}
	b, e := base.ReadBlob(hash)
	if e != nil {
		return models.LibraryItem{}, e
	}
	x, e := r.Get(ctx, id)
	if e != nil {
		return x, e
	}
	// A restore updates the current pointer without manufacturing another
	// history entry. SaveContent still advances the internal revision token.
	return r.SaveContent(ctx, id, b, x.CurrentVersion, false, true)
}

// EnsureLegacy 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) EnsureLegacy(ctx context.Context, errs []models.ErrorProblem, subjects []string) error {
	for _, p := range errs {
		var existingID int64
		var existingKind string
		err := r.store.pool.QueryRow(ctx, "SELECT id,kind FROM library_items WHERE user_id=$1 AND error_problem_id=$2 LIMIT 1", r.store.userID, p.ID).Scan(&existingID, &existingKind)
		if err == nil && existingKind == "note" {
			continue
		}
		body := []byte(postgresLegacyMarkdown(p))
		hash, size, storeErr := base.StoreBlob(bytes.NewReader(body))
		if storeErr != nil {
			return storeErr
		}
		name := strings.TrimSpace(p.Title)
		if name == "" {
			name = fmt.Sprintf("复习笔记 #%d", p.ID)
		}
		tags := postgresMergeTags(p.Tags, p.ReasonTags, []string{p.Subject})
		lastReview := postgresParseLegacyReview(p.LastReview)
		nextReview := p.NextReview
		if nextReview == "" {
			nextReview = time.Now().Format("2006-01-02")
		}
		if err == pgx.ErrNoRows {
			errorID := p.ID
			item, createErr := r.Create(ctx, models.CreateLibraryItemRequest{Kind: "note", Name: name, MimeType: "text/markdown; charset=utf-8", Tags: tags, ReviewEnabled: true, ErrorProblemID: &errorID}, body)
			if createErr != nil {
				return createErr
			}
			existingID = item.ID
		} else if err != nil {
			return err
		}
		_, err = r.store.pool.Exec(ctx, `UPDATE library_items SET parent_id=NULL,kind='note',name=$3,mime_type='text/markdown; charset=utf-8',file_size=$4,tags=$5,blob_hash=$6,current_version=GREATEST(current_version,1),review_enabled=TRUE,review_count=$7,review_stage=$8,last_review=$9,next_review=$10,updated_at=now() WHERE user_id=$1 AND id=$2`, r.store.userID, existingID, name, size, tags, hash, p.ReviewCount, p.ReviewStage, lastReview, nextReview)
		if err != nil {
			return err
		}
		_, err = r.store.pool.Exec(ctx, "INSERT INTO library_versions(user_id,item_id,version,blob_hash,file_size) VALUES($1,$2,1,$3,$4) ON CONFLICT(item_id,version) DO NOTHING", r.store.userID, existingID, hash, size)
		if err != nil {
			return err
		}
	}
	_, err := r.store.pool.Exec(ctx, `DELETE FROM library_items f WHERE f.user_id=$1 AND f.kind='folder' AND (f.name='错题库' AND f.parent_id IS NULL OR f.name=ANY($2) AND EXISTS(SELECT 1 FROM library_items p WHERE p.id=f.parent_id AND p.user_id=$1 AND p.name='错题库')) AND NOT EXISTS(SELECT 1 FROM library_items c WHERE c.parent_id=f.id)`, r.store.userID, subjects)
	return err
}

// ListTags 在存储层中读取并整理所需数据。
func (r *LibraryRepository) ListTags(ctx context.Context) ([]string, error) {
	rows, err := r.store.pool.Query(ctx, "SELECT DISTINCT tag FROM library_items, unnest(tags) tag WHERE user_id=$1 AND deleted_at IS NULL AND btrim(tag)<>'' ORDER BY tag", r.store.userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var tag string
		if err = rows.Scan(&tag); err != nil {
			return nil, err
		}
		out = append(out, tag)
	}
	return out, rows.Err()
}

// DueReviews 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) DueReviews(ctx context.Context, day time.Time) ([]models.LibraryItem, error) {
	return r.List(ctx, base.LibraryFilter{ReviewOnly: true, DueOnly: true})
}

// Review 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) Review(ctx context.Context, id int64, reviewedAt time.Time, intervals []int) (models.LibraryItem, error) {
	return r.ReviewWithRating(ctx, id, reviewedAt, "good")
}

func (r *LibraryRepository) ReviewWithRating(ctx context.Context, id int64, reviewedAt time.Time, rating string) (models.LibraryItem, error) {
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return models.LibraryItem{}, err
	}
	defer tx.Rollback(ctx)
	item, err := scanLibrary(tx.QueryRow(ctx, "SELECT "+libraryColumns+" FROM library_items WHERE user_id=$1 AND id=$2 AND review_enabled=TRUE AND deleted_at IS NULL FOR UPDATE", r.store.userID, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return item, fmt.Errorf("复习笔记不存在")
		}
		return item, err
	}
	item.ReviewStage, item.ReviewCount, item.NextReview = models.NextReview(item.ReviewStage, item.ReviewCount, rating, reviewedAt)
	item, err = scanLibrary(tx.QueryRow(ctx, "UPDATE library_items SET review_count=$3,review_stage=$4,last_review=$5,next_review=$6,updated_at=$5 WHERE user_id=$1 AND id=$2 RETURNING "+libraryColumns, r.store.userID, id, item.ReviewCount, item.ReviewStage, reviewedAt, item.NextReview))
	if err != nil {
		return item, err
	}
	return item, tx.Commit(ctx)
}

// postgresLegacyMarkdown 在存储层中完成本文件定义的局部处理。
func postgresLegacyMarkdown(p models.ErrorProblem) string {
	return "## 题目\n\n" + strings.TrimSpace(p.Question) + "\n\n## 错解\n\n" + strings.TrimSpace(p.Wrong) + "\n\n## 正解\n\n" + strings.TrimSpace(p.Correct) + "\n\n## 错因\n\n" + strings.TrimSpace(p.Reason) + "\n"
}

// postgresMergeTags 在存储层中完成本文件定义的局部处理。
func postgresMergeTags(groups ...[]string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, group := range groups {
		for _, tag := range group {
			tag = strings.TrimSpace(tag)
			key := strings.ToLower(tag)
			if tag != "" && !seen[key] {
				seen[key] = true
				out = append(out, tag)
			}
		}
	}
	return out
}

// postgresParseLegacyReview 在存储层中完成本文件定义的局部处理。
func postgresParseLegacyReview(value *string) *time.Time {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339} {
		if parsed, err := time.ParseInLocation(layout, *value, time.Local); err == nil {
			return &parsed
		}
	}
	return nil
}

// Cleanup 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) Cleanup(ctx context.Context, before time.Time) error {
	_, err := r.store.pool.Exec(ctx, "DELETE FROM library_items WHERE user_id=$1 AND deleted_at IS NOT NULL AND deleted_at<$2", r.store.userID, before)
	return err
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// uniqueName 在存储层中完成本文件定义的局部处理。
func (r *LibraryRepository) uniqueName(ctx context.Context, q queryRower, parent *int64, name string, except int64) (string, error) {
	baseName := name
	for n := 1; ; n++ {
		var exists bool
		e := q.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM library_items WHERE user_id=$1 AND parent_id IS NOT DISTINCT FROM $2 AND lower(name)=lower($3) AND id<>$4 AND deleted_at IS NULL)", r.store.userID, parent, name, except).Scan(&exists)
		if e != nil {
			return "", e
		}
		if !exists {
			return name, nil
		}
		n++
		name = fmt.Sprintf("%s (%d)", baseName, n)
	}
}

var _ base.LibraryRepository = (*LibraryRepository)(nil)
var _ = time.Now
