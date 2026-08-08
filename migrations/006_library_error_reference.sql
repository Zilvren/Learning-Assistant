BEGIN;

-- A library note may reference a legacy error, but it does not own it.
-- Deleting either side must not silently remove the other user's record.
ALTER TABLE library_items
  DROP CONSTRAINT IF EXISTS library_items_error_problem_id_fkey;

ALTER TABLE library_items
  ADD CONSTRAINT library_items_error_problem_id_fkey
  FOREIGN KEY (error_problem_id)
  REFERENCES error_problems(id)
  ON DELETE SET NULL;

COMMIT;
