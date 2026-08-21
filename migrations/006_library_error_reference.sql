BEGIN;

-- 资料库笔记可引用旧版错题，但不拥有该错题。
-- 删除任意一方绝不能静默移除另一方的用户记录。
ALTER TABLE library_items
  DROP CONSTRAINT IF EXISTS library_items_error_problem_id_fkey;

ALTER TABLE library_items
  ADD CONSTRAINT library_items_error_problem_id_fkey
  FOREIGN KEY (error_problem_id)
  REFERENCES error_problems(id)
  ON DELETE SET NULL;

COMMIT;
