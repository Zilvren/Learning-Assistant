BEGIN;

CREATE TABLE IF NOT EXISTS library_items (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  parent_id BIGINT REFERENCES library_items(id) ON DELETE RESTRICT,
  original_parent_id BIGINT REFERENCES library_items(id) ON DELETE SET NULL,
  kind VARCHAR(16) NOT NULL,
  name VARCHAR(255) NOT NULL,
  mime_type VARCHAR(160) NOT NULL DEFAULT '',
  file_size BIGINT NOT NULL DEFAULT 0,
  tags TEXT[] NOT NULL DEFAULT '{}',
  pinned BOOLEAN NOT NULL DEFAULT FALSE,
  current_version INTEGER NOT NULL DEFAULT 0,
  error_problem_id BIGINT REFERENCES error_problems(id) ON DELETE CASCADE,
  blob_hash VARCHAR(64) NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT ck_library_items_kind CHECK (kind IN ('folder','note','error','file')),
  CONSTRAINT ck_library_items_size CHECK (file_size >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_library_items_user_error
  ON library_items(user_id, error_problem_id) WHERE error_problem_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_library_items_parent
  ON library_items(user_id, parent_id, pinned DESC, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_library_items_trash
  ON library_items(user_id, deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_library_items_search
  ON library_items USING GIN (to_tsvector('simple', name || ' ' || array_to_string(tags, ' ')));

CREATE TABLE IF NOT EXISTS library_versions (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  item_id BIGINT NOT NULL REFERENCES library_items(id) ON DELETE CASCADE,
  version INTEGER NOT NULL,
  blob_hash VARCHAR(64) NOT NULL,
  file_size BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(item_id, version)
);

CREATE INDEX IF NOT EXISTS idx_library_versions_item
  ON library_versions(user_id, item_id, version DESC);

COMMIT;
