package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureActivitySchema creates an append-only activity feed for the dashboard.
// It also imports the timestamps that already exist in older databases.
func EnsureActivitySchema(ctx context.Context, pool *pgxpool.Pool) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS user_activity_events (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			activity_date DATE NOT NULL DEFAULT CURRENT_DATE,
			event_type VARCHAR(32) NOT NULL,
			source_key TEXT UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_activity_events_calendar
			ON user_activity_events(user_id, activity_date)`,
		`INSERT INTO user_activity_events (user_id, activity_date, event_type, source_key)
		 SELECT user_id, created_at::date, 'library_create', 'seed:library-create:' || id
		 FROM library_items
		 ON CONFLICT (source_key) DO NOTHING`,
		`INSERT INTO user_activity_events (user_id, activity_date, event_type, source_key)
		 SELECT user_id, updated_at::date, 'library_update', 'seed:library-update:' || id
		 FROM library_items
		 WHERE updated_at::date <> created_at::date
		 ON CONFLICT (source_key) DO NOTHING`,
		`INSERT INTO user_activity_events (user_id, activity_date, event_type, source_key)
		 SELECT user_id, created_at::date, 'error_create', 'seed:error-create:' || id
		 FROM error_problems
		 ON CONFLICT (source_key) DO NOTHING`,
		`INSERT INTO user_activity_events (user_id, activity_date, event_type, source_key)
		 SELECT user_id, updated_at::date, 'error_update', 'seed:error-update:' || id
		 FROM error_problems
		 WHERE updated_at::date <> created_at::date
		 ON CONFLICT (source_key) DO NOTHING`,
		`INSERT INTO user_activity_events (user_id, activity_date, event_type, source_key)
		 SELECT user_id, reviewed_at::date, 'error_review', 'seed:error-review:' || id
		 FROM review_records
		 ON CONFLICT (source_key) DO NOTHING`,
		`CREATE OR REPLACE FUNCTION record_user_activity_event()
		 RETURNS TRIGGER AS $$
		 BEGIN
			INSERT INTO user_activity_events (user_id, activity_date, event_type)
			VALUES (NEW.user_id, CURRENT_DATE, TG_ARGV[0]);
			RETURN NEW;
		 END;
		 $$ LANGUAGE plpgsql`,
		`DROP TRIGGER IF EXISTS trg_library_items_activity ON library_items`,
		`CREATE TRIGGER trg_library_items_activity
		 AFTER INSERT OR UPDATE ON library_items
		 FOR EACH ROW EXECUTE FUNCTION record_user_activity_event('library')`,
		`DROP TRIGGER IF EXISTS trg_error_problems_activity ON error_problems`,
		`CREATE TRIGGER trg_error_problems_activity
		 AFTER INSERT OR UPDATE ON error_problems
		 FOR EACH ROW EXECUTE FUNCTION record_user_activity_event('error')`,
		`DROP TRIGGER IF EXISTS trg_review_records_activity ON review_records`,
		`CREATE TRIGGER trg_review_records_activity
		 AFTER INSERT ON review_records
		 FOR EACH ROW EXECUTE FUNCTION record_user_activity_event('review')`,
	}
	for _, query := range queries {
		if _, err := pool.Exec(ctx, query); err != nil {
			return err
		}
	}
	return nil
}
