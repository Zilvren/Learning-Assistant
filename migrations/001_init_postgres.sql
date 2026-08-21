-- 学习追踪器的 PostgreSQL 初始架构。
-- 目标版本：PostgreSQL 14 及以上。

BEGIN;

CREATE OR REPLACE FUNCTION touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE users (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  username VARCHAR(64) NOT NULL,
  email VARCHAR(255),
  password_hash TEXT NOT NULL,
  status VARCHAR(24) NOT NULL DEFAULT 'active',
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT ck_users_status CHECK (status IN ('active', 'disabled', 'deleted'))
);

CREATE UNIQUE INDEX uq_users_username_active
  ON users (lower(username))
  WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uq_users_email_active
  ON users (lower(email))
  WHERE email IS NOT NULL AND deleted_at IS NULL;

CREATE TRIGGER trg_users_touch_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE user_settings (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  display_name VARCHAR(80),
  mineru_token_cipher TEXT,
  settings JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_settings_settings_gin
  ON user_settings USING GIN (settings jsonb_path_ops);

CREATE TRIGGER trg_user_settings_touch_updated_at
BEFORE UPDATE ON user_settings
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE subjects (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name VARCHAR(120) NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX uq_subjects_user_id_id
  ON subjects (user_id, id);

CREATE UNIQUE INDEX uq_subjects_user_name_active
  ON subjects (user_id, lower(name))
  WHERE deleted_at IS NULL;

CREATE INDEX idx_subjects_user_sort
  ON subjects (user_id, sort_order, id)
  WHERE deleted_at IS NULL;

CREATE TRIGGER trg_subjects_touch_updated_at
BEFORE UPDATE ON subjects
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE error_problems (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject_id BIGINT,
  title VARCHAR(255) NOT NULL DEFAULT '',
  question TEXT NOT NULL DEFAULT '',
  wrong_answer TEXT NOT NULL DEFAULT '',
  correct_answer TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  difficulty SMALLINT NOT NULL DEFAULT 0,
  source VARCHAR(32) NOT NULL DEFAULT 'manual',
  review_count INTEGER NOT NULL DEFAULT 0,
  review_stage INTEGER NOT NULL DEFAULT 0,
  next_review_at DATE,
  last_reviewed_at TIMESTAMPTZ,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT ck_error_problems_difficulty CHECK (difficulty BETWEEN 0 AND 5),
  CONSTRAINT ck_error_problems_review_count CHECK (review_count >= 0),
  CONSTRAINT ck_error_problems_review_stage CHECK (review_stage >= 0),
  CONSTRAINT ck_error_problems_source CHECK (source IN ('manual', 'ocr', 'import', 'sync')),
  CONSTRAINT fk_error_problems_subject
    FOREIGN KEY (user_id, subject_id)
    REFERENCES subjects(user_id, id)
);

CREATE UNIQUE INDEX uq_error_problems_user_id_id
  ON error_problems (user_id, id);

CREATE INDEX idx_error_problems_user_next_review
  ON error_problems (user_id, next_review_at, difficulty DESC, created_at)
  WHERE deleted_at IS NULL AND archived_at IS NULL;

CREATE INDEX idx_error_problems_user_subject_created
  ON error_problems (user_id, subject_id, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX idx_error_problems_user_created
  ON error_problems (user_id, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX idx_error_problems_metadata_gin
  ON error_problems USING GIN (metadata jsonb_path_ops);

CREATE INDEX idx_error_problems_search_gin
  ON error_problems USING GIN (
    to_tsvector(
      'simple',
      coalesce(title, '') || ' ' ||
      coalesce(question, '') || ' ' ||
      coalesce(correct_answer, '') || ' ' ||
      coalesce(reason, '')
    )
  );

CREATE TRIGGER trg_error_problems_touch_updated_at
BEFORE UPDATE ON error_problems
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE tags (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name VARCHAR(80) NOT NULL,
  tag_type VARCHAR(24) NOT NULL DEFAULT 'question',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT ck_tags_type CHECK (tag_type IN ('question', 'reason'))
);

CREATE UNIQUE INDEX uq_tags_user_id_id
  ON tags (user_id, id);

CREATE UNIQUE INDEX uq_tags_user_type_name_active
  ON tags (user_id, tag_type, lower(name))
  WHERE deleted_at IS NULL;

CREATE INDEX idx_tags_user_type
  ON tags (user_id, tag_type, name)
  WHERE deleted_at IS NULL;

CREATE TRIGGER trg_tags_touch_updated_at
BEFORE UPDATE ON tags
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE error_problem_tags (
  user_id BIGINT NOT NULL,
  error_problem_id BIGINT NOT NULL,
  tag_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (error_problem_id, tag_id),
  CONSTRAINT fk_error_problem_tags_problem
    FOREIGN KEY (user_id, error_problem_id)
    REFERENCES error_problems(user_id, id)
    ON DELETE CASCADE,
  CONSTRAINT fk_error_problem_tags_tag
    FOREIGN KEY (user_id, tag_id)
    REFERENCES tags(user_id, id)
    ON DELETE CASCADE
);

CREATE INDEX idx_error_problem_tags_user_tag
  ON error_problem_tags (user_id, tag_id, error_problem_id);

CREATE TABLE review_records (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id BIGINT NOT NULL,
  error_problem_id BIGINT NOT NULL,
  review_no INTEGER NOT NULL,
  result VARCHAR(24) NOT NULL,
  reviewed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  next_review_at DATE,
  duration_seconds INTEGER,
  note TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT ck_review_records_review_no CHECK (review_no > 0),
  CONSTRAINT ck_review_records_result CHECK (result IN ('remembered', 'forgotten', 'skipped')),
  CONSTRAINT ck_review_records_duration CHECK (duration_seconds IS NULL OR duration_seconds >= 0),
  CONSTRAINT fk_review_records_problem
    FOREIGN KEY (user_id, error_problem_id)
    REFERENCES error_problems(user_id, id)
    ON DELETE CASCADE
);

CREATE INDEX idx_review_records_user_reviewed
  ON review_records (user_id, reviewed_at DESC);

CREATE INDEX idx_review_records_user_problem_reviewed
  ON review_records (user_id, error_problem_id, reviewed_at DESC);

CREATE INDEX idx_review_records_metadata_gin
  ON review_records USING GIN (metadata jsonb_path_ops);

CREATE TABLE ocr_tasks (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider VARCHAR(32) NOT NULL DEFAULT 'mineru',
  status VARCHAR(24) NOT NULL DEFAULT 'pending',
  source_filename VARCHAR(255) NOT NULL DEFAULT '',
  mime_type VARCHAR(120) NOT NULL DEFAULT '',
  file_size BIGINT NOT NULL DEFAULT 0,
  batch_id VARCHAR(128),
  task_id VARCHAR(128),
  result_markdown TEXT,
  error_message TEXT,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at TIMESTAMPTZ,
  CONSTRAINT ck_ocr_tasks_status CHECK (status IN ('pending', 'uploading', 'processing', 'succeeded', 'failed', 'canceled')),
  CONSTRAINT ck_ocr_tasks_file_size CHECK (file_size >= 0)
);

CREATE INDEX idx_ocr_tasks_user_created
  ON ocr_tasks (user_id, created_at DESC);

CREATE UNIQUE INDEX uq_ocr_tasks_user_id_id
  ON ocr_tasks (user_id, id);

CREATE INDEX idx_ocr_tasks_user_status
  ON ocr_tasks (user_id, status, created_at DESC);

CREATE INDEX idx_ocr_tasks_provider_task_id
  ON ocr_tasks (provider, task_id)
  WHERE task_id IS NOT NULL;

CREATE INDEX idx_ocr_tasks_metadata_gin
  ON ocr_tasks USING GIN (metadata jsonb_path_ops);

CREATE TRIGGER trg_ocr_tasks_touch_updated_at
BEFORE UPDATE ON ocr_tasks
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE attachments (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  error_problem_id BIGINT,
  ocr_task_id BIGINT,
  file_name VARCHAR(255) NOT NULL,
  mime_type VARCHAR(120) NOT NULL DEFAULT '',
  file_size BIGINT NOT NULL DEFAULT 0,
  storage_type VARCHAR(24) NOT NULL DEFAULT 'local',
  storage_key TEXT NOT NULL,
  checksum VARCHAR(128),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT ck_attachments_storage_type CHECK (storage_type IN ('inline', 'local', 'object')),
  CONSTRAINT ck_attachments_file_size CHECK (file_size >= 0),
  CONSTRAINT ck_attachments_owner CHECK (error_problem_id IS NOT NULL OR ocr_task_id IS NOT NULL),
  CONSTRAINT fk_attachments_problem
    FOREIGN KEY (user_id, error_problem_id)
    REFERENCES error_problems(user_id, id)
    ON DELETE CASCADE,
  CONSTRAINT fk_attachments_ocr_task
    FOREIGN KEY (user_id, ocr_task_id)
    REFERENCES ocr_tasks(user_id, id)
    ON DELETE CASCADE
);

CREATE INDEX idx_attachments_user_problem
  ON attachments (user_id, error_problem_id, created_at DESC)
  WHERE error_problem_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX idx_attachments_user_ocr_task
  ON attachments (user_id, ocr_task_id, created_at DESC)
  WHERE ocr_task_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX idx_attachments_checksum
  ON attachments (checksum)
  WHERE checksum IS NOT NULL;

CREATE TRIGGER trg_attachments_touch_updated_at
BEFORE UPDATE ON attachments
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE knowledge_items (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  subject_id BIGINT,
  content TEXT NOT NULL,
  source VARCHAR(32) NOT NULL DEFAULT 'manual',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT ck_knowledge_items_source CHECK (source IN ('manual', 'ocr', 'analysis', 'import')),
  CONSTRAINT fk_knowledge_items_subject
    FOREIGN KEY (user_id, subject_id)
    REFERENCES subjects(user_id, id)
);

CREATE INDEX idx_knowledge_items_user_subject
  ON knowledge_items (user_id, subject_id, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX idx_knowledge_items_metadata_gin
  ON knowledge_items USING GIN (metadata jsonb_path_ops);

CREATE INDEX idx_knowledge_items_search_gin
  ON knowledge_items USING GIN (to_tsvector('simple', coalesce(content, '')));

CREATE TRIGGER trg_knowledge_items_touch_updated_at
BEFORE UPDATE ON knowledge_items
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE refresh_tokens (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash VARCHAR(128) NOT NULL,
  user_agent TEXT,
  ip_address INET,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_refresh_tokens_token_hash
  ON refresh_tokens (token_hash);

CREATE INDEX idx_refresh_tokens_user_active
  ON refresh_tokens (user_id, expires_at DESC)
  WHERE revoked_at IS NULL;

COMMIT;
