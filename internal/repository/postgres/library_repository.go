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

func scanLibrary(row pgx.Row) (models.LibraryItem, error) {
	var x models.LibraryItem
	err := row.Scan(&x.ID, &x.ParentID, &x.OriginalParent, &x.Kind, &x.Name, &x.MimeType, &x.Size, &x.Tags, &x.Pinned, &x.CurrentVersion, &x.ErrorProblemID, &x.BlobHash, &x.ReviewEnabled, &x.ReviewCount, &x.ReviewStage, &x.LastReview, &x.NextReview, &x.CreatedAt, &x.UpdatedAt, &x.DeletedAt)
	return x, err
}
func scanLibraryRows(rows pgx.Rows) (models.LibraryItem, error) {
	var x models.LibraryItem
	err := rows.Scan(&x.ID, &x.ParentID, &x.OriginalParent, &x.Kind, &x.Name, &x.MimeType, &x.Size, &x.Tags, &x.Pinned, &x.CurrentVersion, &x.ErrorProblemID, &x.BlobHash, &x.ReviewEnabled, &x.ReviewCount, &x.ReviewStage, &x.LastReview, &x.NextReview, &x.CreatedAt, &x.UpdatedAt, &x.DeletedAt)
	return x, err
}

func (r *LibraryRepository) List(ctx context.Context, f base.LibraryFilter) ([]models.LibraryItem, error) {
	args := []any{r.store.userID}
	where := []string{"user_id=$1"}
	if f.Trashed {
		where = append(where, "deleted_at IS NOT NULL")
	} else {
		where = append(where, "deleted_at IS NULL")
	}
	if f.ParentID != nil {
		args = append(args, *f.ParentID)
		where = append(where, fmt.Sprintf("parent_id=$%d", len(args)))
	} else if f.Query == "" && !f.Trashed && !f.ReviewOnly {
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
	if strings.TrimSpace(f.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(f.Query))+"%")
		where = append(where, fmt.Sprintf("(lower(name) LIKE $%d OR EXISTS(SELECT 1 FROM unnest(tags) t WHERE lower(t) LIKE $%d))", len(args), len(args)))
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
func (r *LibraryRepository) Get(ctx context.Context, id int64) (models.LibraryItem, error) {
	x, e := scanLibrary(r.store.pool.QueryRow(ctx, "SELECT "+libraryColumns+" FROM library_items WHERE user_id=$1 AND id=$2", r.store.userID, id))
	if e == pgx.ErrNoRows {
		return x, fmt.Errorf("资料不存在")
	}
	return x, e
}
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
	out, err = scanLibrary(tx.QueryRow(ctx, "INSERT INTO library_items(user_id,parent_id,kind,name,mime_type,file_size,tags,current_version,blob_hash,review_enabled,next_review) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING "+libraryColumns, r.store.userID, req.ParentID, req.Kind, name, req.MimeType, size, req.Tags, boolInt(req.Kind != "folder"), hash, req.ReviewEnabled, nextReview))
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
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func (r *LibraryRepository) Update(ctx context.Context, id int64, req models.UpdateLibraryItemRequest) (models.LibraryItem, error) {
	item, err := r.Get(ctx, id)
	if err != nil {
		return item, err
	}
	if req.Name != nil {
		item.Name = strings.TrimSpace(*req.Name)
	}
	if req.Tags != nil {
		item.Tags = *req.Tags
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
	if req.ParentID != nil {
		var invalid bool
		err = r.store.pool.QueryRow(ctx, `WITH RECURSIVE tree AS (SELECT id FROM library_items WHERE user_id=$1 AND id=$2 UNION ALL SELECT c.id FROM library_items c JOIN tree p ON c.parent_id=p.id WHERE c.user_id=$1) SELECT NOT EXISTS(SELECT 1 FROM library_items WHERE user_id=$1 AND id=$3 AND kind='folder' AND deleted_at IS NULL) OR EXISTS(SELECT 1 FROM tree WHERE id=$3)`, r.store.userID, id, *req.ParentID).Scan(&invalid)
		if err != nil {
			return item, err
		}
		if invalid {
			return item, fmt.Errorf("不能移动到自身或子文件夹")
		}
		item.ParentID = req.ParentID
	}
	item.Name, err = r.uniqueName(ctx, r.store.pool, item.ParentID, item.Name, id)
	if err != nil {
		return item, err
	}
	return scanLibrary(r.store.pool.QueryRow(ctx, "UPDATE library_items SET parent_id=$3,name=$4,tags=$5,pinned=$6,review_enabled=$7,next_review=$8,updated_at=now() WHERE user_id=$1 AND id=$2 RETURNING "+libraryColumns, r.store.userID, id, item.ParentID, item.Name, item.Tags, item.Pinned, item.ReviewEnabled, item.NextReview))
}
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
	item, err = scanLibrary(tx.QueryRow(ctx, "UPDATE library_items SET blob_hash=$3,file_size=$4,current_version=current_version+1,updated_at=now() WHERE user_id=$1 AND id=$2 RETURNING "+libraryColumns, r.store.userID, id, hash, size))
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
func (r *LibraryRepository) Trash(ctx context.Context, id int64) error {
	_, err := r.store.pool.Exec(ctx, "WITH RECURSIVE tree AS (SELECT id FROM library_items WHERE user_id=$1 AND id=$2 UNION ALL SELECT c.id FROM library_items c JOIN tree p ON c.parent_id=p.id WHERE c.user_id=$1) UPDATE library_items SET original_parent_id=parent_id,deleted_at=now() WHERE id IN(SELECT id FROM tree)", r.store.userID, id)
	return err
}
func (r *LibraryRepository) Restore(ctx context.Context, id int64) (models.LibraryItem, error) {
	_, err := r.store.pool.Exec(ctx, "WITH RECURSIVE tree AS (SELECT id FROM library_items WHERE user_id=$1 AND id=$2 UNION ALL SELECT c.id FROM library_items c JOIN tree p ON c.parent_id=p.id WHERE c.user_id=$1) UPDATE library_items SET parent_id=COALESCE(original_parent_id,parent_id),original_parent_id=NULL,deleted_at=NULL,updated_at=now() WHERE id IN(SELECT id FROM tree)", r.store.userID, id)
	if err != nil {
		return models.LibraryItem{}, err
	}
	return r.Get(ctx, id)
}
func (r *LibraryRepository) Purge(ctx context.Context, id int64) error {
	_, err := r.store.pool.Exec(ctx, "WITH RECURSIVE tree AS (SELECT id FROM library_items WHERE user_id=$1 AND id=$2 AND deleted_at IS NOT NULL UNION ALL SELECT c.id FROM library_items c JOIN tree p ON c.parent_id=p.id WHERE c.user_id=$1) DELETE FROM library_items WHERE id IN(SELECT id FROM tree)", r.store.userID, id)
	return err
}
func (r *LibraryRepository) Duplicate(ctx context.Context, id int64, parent *int64) (models.LibraryItem, error) {
	b, x, e := r.ReadContent(ctx, id)
	if e != nil {
		return x, e
	}
	return r.Create(ctx, models.CreateLibraryItemRequest{ParentID: parent, Kind: x.Kind, Name: x.Name, MimeType: x.MimeType, Tags: x.Tags, ReviewEnabled: x.ReviewEnabled}, b)
}
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
			item, createErr := r.Create(ctx, models.CreateLibraryItemRequest{Kind: "note", Name: name, MimeType: "text/markdown; charset=utf-8", Tags: tags, ReviewEnabled: true}, body)
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

func (r *LibraryRepository) DueReviews(ctx context.Context, day time.Time) ([]models.LibraryItem, error) {
	return r.List(ctx, base.LibraryFilter{ReviewOnly: true, DueOnly: true})
}

func (r *LibraryRepository) Review(ctx context.Context, id int64, reviewedAt time.Time, intervals []int) (models.LibraryItem, error) {
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
	item.ReviewCount++
	item.ReviewStage = item.ReviewCount
	if len(intervals) == 0 {
		intervals = []int{0}
	}
	index := item.ReviewCount
	if index >= len(intervals) {
		index = len(intervals) - 1
	}
	if index < 0 {
		index = 0
	}
	item, err = scanLibrary(tx.QueryRow(ctx, "UPDATE library_items SET review_count=$3,review_stage=$4,last_review=$5,next_review=$6,updated_at=$5 WHERE user_id=$1 AND id=$2 RETURNING "+libraryColumns, r.store.userID, id, item.ReviewCount, item.ReviewStage, reviewedAt, reviewedAt.AddDate(0, 0, intervals[index]).Format("2006-01-02")))
	if err != nil {
		return item, err
	}
	return item, tx.Commit(ctx)
}

func postgresLegacyMarkdown(p models.ErrorProblem) string {
	return "## 题目\n\n" + strings.TrimSpace(p.Question) + "\n\n## 错解\n\n" + strings.TrimSpace(p.Wrong) + "\n\n## 正解\n\n" + strings.TrimSpace(p.Correct) + "\n\n## 错因\n\n" + strings.TrimSpace(p.Reason) + "\n"
}

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

func (r *LibraryRepository) Cleanup(ctx context.Context, before time.Time) error {
	_, err := r.store.pool.Exec(ctx, "DELETE FROM library_items WHERE user_id=$1 AND deleted_at IS NOT NULL AND deleted_at<$2", r.store.userID, before)
	return err
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

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
