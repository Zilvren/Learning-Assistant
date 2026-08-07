BEGIN;

ALTER TABLE library_items ADD COLUMN IF NOT EXISTS review_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE library_items ADD COLUMN IF NOT EXISTS review_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE library_items ADD COLUMN IF NOT EXISTS review_stage INTEGER NOT NULL DEFAULT 0;
ALTER TABLE library_items ADD COLUMN IF NOT EXISTS last_review TIMESTAMPTZ;
ALTER TABLE library_items ADD COLUMN IF NOT EXISTS next_review VARCHAR(10) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_library_items_review
  ON library_items(user_id, next_review) WHERE review_enabled = TRUE AND deleted_at IS NULL;

COMMIT;
