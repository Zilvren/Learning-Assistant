BEGIN;

ALTER TABLE user_activity_events
  ADD COLUMN IF NOT EXISTS value INTEGER NOT NULL DEFAULT 1;

ALTER TABLE ocr_tasks
  ADD COLUMN IF NOT EXISTS input_blob_hash VARCHAR(64) NOT NULL DEFAULT '';

ALTER TABLE ocr_tasks DROP CONSTRAINT IF EXISTS ck_ocr_tasks_status;
ALTER TABLE ocr_tasks ADD CONSTRAINT ck_ocr_tasks_status CHECK (status IN ('queued', 'pending', 'uploading', 'processing', 'succeeded', 'failed', 'canceled'));

CREATE TABLE IF NOT EXISTS learning_relations (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  from_type VARCHAR(16) NOT NULL,
  from_id BIGINT NOT NULL,
  to_type VARCHAR(16) NOT NULL,
  to_id BIGINT NOT NULL,
  label VARCHAR(120) NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_learning_relation_types CHECK (from_type IN ('library', 'error') AND to_type IN ('library', 'error')),
  CONSTRAINT ck_learning_relation_distinct CHECK (from_type <> to_type OR from_id <> to_id),
  CONSTRAINT uq_learning_relations UNIQUE (user_id, from_type, from_id, to_type, to_id)
);

CREATE INDEX IF NOT EXISTS idx_learning_relations_from ON learning_relations(user_id, from_type, from_id);
CREATE INDEX IF NOT EXISTS idx_learning_relations_to ON learning_relations(user_id, to_type, to_id);

ALTER TABLE review_records DROP CONSTRAINT IF EXISTS ck_review_records_result;
ALTER TABLE review_records ADD CONSTRAINT ck_review_records_result CHECK (result IN ('remembered', 'forgotten', 'skipped', 'again', 'hard', 'good', 'easy'));

ALTER TABLE library_items ADD COLUMN IF NOT EXISTS search_text TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_library_items_search_text
  ON library_items USING GIN (to_tsvector('simple', coalesce(search_text, '')));

COMMIT;
