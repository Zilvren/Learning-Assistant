package postgres

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type ErrorRepository struct {
	store *Store
}

func (r *ErrorRepository) Create(ctx context.Context, item models.ErrorProblem) (models.ErrorProblem, error) {
	normalizeProblem(&item)
	subjectID, err := (&SubjectRepository{store: r.store}).findID(ctx, item.Subject)
	if err != nil {
		return models.ErrorProblem{}, fmt.Errorf("无效科目")
	}

	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return models.ErrorProblem{}, err
	}
	defer tx.Rollback(ctx)

	id, err := r.insertProblem(ctx, tx, item, subjectID, false)
	if err != nil {
		return models.ErrorProblem{}, err
	}
	if err := r.replaceTags(ctx, tx, id, "question", item.Tags); err != nil {
		return models.ErrorProblem{}, err
	}
	if err := r.replaceTags(ctx, tx, id, "reason", item.ReasonTags); err != nil {
		return models.ErrorProblem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.ErrorProblem{}, err
	}
	item.ID = int(id)
	return item, nil
}

func (r *ErrorRepository) List(ctx context.Context, filter base.ErrorFilter) ([]models.ErrorProblem, error) {
	rows, err := r.store.pool.Query(ctx, `
		SELECT
			p.id,
			coalesce(s.name, ''),
			p.title,
			p.question,
			p.wrong_answer,
			p.correct_answer,
			p.reason,
			p.review_count,
			p.review_stage,
			p.created_at,
			p.last_reviewed_at,
			p.next_review_at
		FROM error_problems p
		LEFT JOIN subjects s ON s.user_id = p.user_id AND s.id = p.subject_id
		WHERE p.user_id = $1
		  AND p.deleted_at IS NULL
		  AND p.archived_at IS NULL
		ORDER BY p.created_at DESC, p.id DESC
	`, r.store.userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	all := []models.ErrorProblem{}
	ids := []int64{}
	for rows.Next() {
		item, err := scanProblem(rows)
		if err != nil {
			return nil, err
		}
		all = append(all, item)
		ids = append(ids, int64(item.ID))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.attachTags(ctx, all, ids); err != nil {
		return nil, err
	}

	result := []models.ErrorProblem{}
	for _, item := range all {
		normalizeProblem(&item)
		if !matchesFilter(item, filter) {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *ErrorRepository) Get(ctx context.Context, id int) (models.ErrorProblem, error) {
	rows, err := r.store.pool.Query(ctx, `
		SELECT
			p.id,
			coalesce(s.name, ''),
			p.title,
			p.question,
			p.wrong_answer,
			p.correct_answer,
			p.reason,
			p.review_count,
			p.review_stage,
			p.created_at,
			p.last_reviewed_at,
			p.next_review_at
		FROM error_problems p
		LEFT JOIN subjects s ON s.user_id = p.user_id AND s.id = p.subject_id
		WHERE p.user_id = $1
		  AND p.id = $2
		  AND p.deleted_at IS NULL
	`, r.store.userID, id)
	if err != nil {
		return models.ErrorProblem{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return models.ErrorProblem{}, notFound("错题", id)
	}
	item, err := scanProblem(rows)
	if err != nil {
		return models.ErrorProblem{}, err
	}
	item, err = r.withTags(ctx, item)
	if err != nil {
		return models.ErrorProblem{}, err
	}
	normalizeProblem(&item)
	return item, rows.Err()
}

func (r *ErrorRepository) Update(ctx context.Context, id int, req models.UpdateErrorRequest) error {
	item, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	applyErrorUpdate(&item, req)
	normalizeProblem(&item)

	subjectID, err := (&SubjectRepository{store: r.store}).findID(ctx, item.Subject)
	if err != nil {
		return fmt.Errorf("无效科目")
	}

	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE error_problems
		SET subject_id = $3,
		    title = $4,
		    question = $5,
		    wrong_answer = $6,
		    correct_answer = $7,
		    reason = $8
		WHERE user_id = $1
		  AND id = $2
		  AND deleted_at IS NULL
	`, r.store.userID, id, subjectID, item.Title, item.Question, item.Wrong, item.Correct, item.Reason)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("错题", id)
	}
	if req.Tags != nil {
		if err := r.replaceTags(ctx, tx, int64(id), "question", *req.Tags); err != nil {
			return err
		}
	}
	if req.ReasonTags != nil {
		if err := r.replaceTags(ctx, tx, int64(id), "reason", *req.ReasonTags); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *ErrorRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.store.pool.Exec(ctx, `
		UPDATE error_problems
		SET deleted_at = now()
		WHERE user_id = $1
		  AND id = $2
		  AND deleted_at IS NULL
	`, r.store.userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notFound("错题", id)
	}
	return nil
}

func (r *ErrorRepository) UpdateReview(ctx context.Context, id int, reviewedAt string, reviewCount int, reviewStage int, nextReview string) (models.ErrorProblem, error) {
	reviewed := parseDateTime(reviewedAt)
	if reviewed == nil {
		now := time.Now()
		reviewed = &now
	}
	next := parseDate(nextReview)

	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return models.ErrorProblem{}, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE error_problems
		SET review_count = $3,
		    review_stage = $4,
		    last_reviewed_at = $5,
		    next_review_at = $6
		WHERE user_id = $1
		  AND id = $2
		  AND deleted_at IS NULL
	`, r.store.userID, id, reviewCount, reviewStage, reviewed, next)
	if err != nil {
		return models.ErrorProblem{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.ErrorProblem{}, notFound("错题", id)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO review_records (user_id, error_problem_id, review_no, result, reviewed_at, next_review_at)
		VALUES ($1, $2, $3, 'remembered', $4, $5)
	`, r.store.userID, id, reviewCount, reviewed, next); err != nil {
		return models.ErrorProblem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.ErrorProblem{}, err
	}
	return r.Get(ctx, id)
}

func (r *ErrorRepository) ListTags(ctx context.Context) ([]string, error) {
	rows, err := r.store.pool.Query(ctx, `
		SELECT DISTINCT name
		FROM tags
		WHERE user_id = $1
		  AND deleted_at IS NULL
		ORDER BY name
	`, r.store.userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	return tags, rows.Err()
}

func (r *ErrorRepository) Replace(ctx context.Context, errors []models.ErrorProblem) error {
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := r.clearUserErrors(ctx, tx); err != nil {
		return err
	}
	for _, item := range errors {
		normalizeProblem(&item)
		subjectID, err := r.ensureSubjectIDTx(ctx, tx, item.Subject)
		if err != nil {
			return err
		}
		id, err := r.insertProblem(ctx, tx, item, subjectID, item.ID > 0)
		if err != nil {
			return err
		}
		if err := r.replaceTags(ctx, tx, id, "question", item.Tags); err != nil {
			return err
		}
		if err := r.replaceTags(ctx, tx, id, "reason", item.ReasonTags); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		SELECT setval(pg_get_serial_sequence('error_problems', 'id'), greatest((SELECT coalesce(max(id), 1) FROM error_problems), 1), true)
	`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *ErrorRepository) HasAny(ctx context.Context) (bool, error) {
	var exists bool
	err := r.store.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM error_problems
			WHERE user_id = $1
			  AND deleted_at IS NULL
		)
	`, r.store.userID).Scan(&exists)
	return exists, err
}

func (r *ErrorRepository) insertProblem(ctx context.Context, tx pgx.Tx, item models.ErrorProblem, subjectID int64, preserveID bool) (int64, error) {
	created := parseDateTime(item.Created)
	if created == nil {
		now := time.Now()
		created = &now
	}
	last := (*time.Time)(nil)
	if item.LastReview != nil {
		last = parseDateTime(*item.LastReview)
	}
	next := parseDate(item.NextReview)

	if preserveID {
		ok, err := r.canPreserveProblemID(ctx, tx, int64(item.ID))
		if err != nil {
			return 0, err
		}
		preserveID = ok
	}

	if preserveID {
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO error_problems (
				id, user_id, subject_id, title, question, wrong_answer, correct_answer,
				reason, source, review_count, review_stage, next_review_at,
				last_reviewed_at, created_at
			)
			OVERRIDING SYSTEM VALUE
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'import', $9, $10, $11, $12, $13)
			RETURNING id
		`, item.ID, r.store.userID, subjectID, item.Title, item.Question, item.Wrong, item.Correct, item.Reason, item.ReviewCount, item.ReviewStage, next, last, created).Scan(&id)
		return id, err
	}

	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO error_problems (
			user_id, subject_id, title, question, wrong_answer, correct_answer,
			reason, source, review_count, review_stage, next_review_at,
			last_reviewed_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'manual', $8, $9, $10, $11, $12)
		RETURNING id
	`, r.store.userID, subjectID, item.Title, item.Question, item.Wrong, item.Correct, item.Reason, item.ReviewCount, item.ReviewStage, next, last, created).Scan(&id)
	return id, err
}

func (r *ErrorRepository) canPreserveProblemID(ctx context.Context, tx pgx.Tx, id int64) (bool, error) {
	if id <= 0 {
		return false, nil
	}
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM error_problems
			WHERE id = $1
		)
	`, id).Scan(&exists)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

func (r *ErrorRepository) clearUserErrors(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM review_records WHERE user_id = $1`, r.store.userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM error_problem_tags WHERE user_id = $1`, r.store.userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM error_problems WHERE user_id = $1`, r.store.userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM tags WHERE user_id = $1`, r.store.userID); err != nil {
		return err
	}
	return nil
}

func (r *ErrorRepository) ensureSubjectIDTx(ctx context.Context, tx pgx.Tx, subject string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM subjects
		WHERE user_id = $1
		  AND name = $2
		  AND deleted_at IS NULL
	`, r.store.userID, subject).Scan(&id)
	if err == nil {
		return id, nil
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO subjects (user_id, name, sort_order)
		VALUES ($1, $2, (
			SELECT coalesce(max(sort_order), 0) + 1
			FROM subjects
			WHERE user_id = $1
		))
		RETURNING id
	`, r.store.userID, subject).Scan(&id)
	return id, err
}

func (r *ErrorRepository) replaceTags(ctx context.Context, tx pgx.Tx, problemID int64, tagType string, tags []string) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM error_problem_tags ept
		USING tags t
		WHERE ept.user_id = $1
		  AND ept.error_problem_id = $2
		  AND ept.tag_id = t.id
		  AND t.tag_type = $3
	`, r.store.userID, problemID, tagType); err != nil {
		return err
	}

	seen := map[string]bool{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		tagID, err := r.ensureTag(ctx, tx, tag, tagType)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO error_problem_tags (user_id, error_problem_id, tag_id)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING
		`, r.store.userID, problemID, tagID); err != nil {
			return err
		}
	}
	return nil
}

func (r *ErrorRepository) ensureTag(ctx context.Context, tx pgx.Tx, name string, tagType string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM tags
		WHERE user_id = $1
		  AND tag_type = $2
		  AND name = $3
		  AND deleted_at IS NULL
	`, r.store.userID, tagType, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO tags (user_id, name, tag_type)
		VALUES ($1, $2, $3)
		RETURNING id
	`, r.store.userID, name, tagType).Scan(&id)
	return id, err
}

func (r *ErrorRepository) attachTags(ctx context.Context, items []models.ErrorProblem, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := r.store.pool.Query(ctx, `
		SELECT ept.error_problem_id, t.name, t.tag_type
		FROM error_problem_tags ept
		JOIN tags t ON t.user_id = ept.user_id AND t.id = ept.tag_id
		WHERE ept.user_id = $1
		  AND ept.error_problem_id = ANY($2)
		  AND t.deleted_at IS NULL
		ORDER BY t.name
	`, r.store.userID, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	byID := map[int]*models.ErrorProblem{}
	for i := range items {
		byID[items[i].ID] = &items[i]
	}
	for rows.Next() {
		var problemID int
		var name string
		var tagType string
		if err := rows.Scan(&problemID, &name, &tagType); err != nil {
			return err
		}
		item := byID[problemID]
		if item == nil {
			continue
		}
		if tagType == "reason" {
			item.ReasonTags = append(item.ReasonTags, name)
		} else {
			item.Tags = append(item.Tags, name)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range items {
		if item := byID[items[i].ID]; item != nil {
			items[i].Tags = item.Tags
			items[i].ReasonTags = item.ReasonTags
		}
	}
	return nil
}

func (r *ErrorRepository) withTags(ctx context.Context, item models.ErrorProblem) (models.ErrorProblem, error) {
	items := []models.ErrorProblem{item}
	if err := r.attachTags(ctx, items, []int64{int64(item.ID)}); err != nil {
		return models.ErrorProblem{}, err
	}
	return items[0], nil
}

type problemScanner interface {
	Scan(dest ...interface{}) error
}

func scanProblem(scanner problemScanner) (models.ErrorProblem, error) {
	var item models.ErrorProblem
	var created time.Time
	var last pgtype.Timestamptz
	var next pgtype.Date
	if err := scanner.Scan(
		&item.ID,
		&item.Subject,
		&item.Title,
		&item.Question,
		&item.Wrong,
		&item.Correct,
		&item.Reason,
		&item.ReviewCount,
		&item.ReviewStage,
		&created,
		&last,
		&next,
	); err != nil {
		return models.ErrorProblem{}, err
	}
	item.Created = created.Local().Format(dateTimeLayout)
	if last.Valid {
		value := last.Time.Local().Format(dateTimeLayout)
		item.LastReview = &value
	}
	if next.Valid {
		item.NextReview = next.Time.Format(dateLayout)
	}
	item.Tags = []string{}
	item.ReasonTags = []string{}
	return item, nil
}

func sortProblemTags(item *models.ErrorProblem) {
	sort.Strings(item.Tags)
	sort.Strings(item.ReasonTags)
}
