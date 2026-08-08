BEGIN;

-- Autosave may update one note many times in a day. Count a source item only
-- once per calendar day instead of turning keystrokes into study activity.
CREATE OR REPLACE FUNCTION record_user_activity_event()
RETURNS TRIGGER AS $$
DECLARE
  event_day DATE;
BEGIN
  IF TG_ARGV[0] = 'review' THEN
    event_day := (NEW.reviewed_at AT TIME ZONE 'Asia/Shanghai')::date;
  ELSE
    event_day := (NEW.updated_at AT TIME ZONE 'Asia/Shanghai')::date;
  END IF;
  INSERT INTO user_activity_events (user_id, activity_date, event_type, source_key)
  VALUES (NEW.user_id, event_day, TG_ARGV[0], TG_ARGV[0] || ':' || NEW.id || ':' || event_day::text)
  ON CONFLICT (source_key) DO NOTHING;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_library_items_activity ON library_items;
CREATE TRIGGER trg_library_items_activity
AFTER INSERT OR UPDATE ON library_items
FOR EACH ROW EXECUTE FUNCTION record_user_activity_event('library');

DROP TRIGGER IF EXISTS trg_error_problems_activity ON error_problems;
CREATE TRIGGER trg_error_problems_activity
AFTER INSERT OR UPDATE ON error_problems
FOR EACH ROW EXECUTE FUNCTION record_user_activity_event('error');

DROP TRIGGER IF EXISTS trg_review_records_activity ON review_records;
CREATE TRIGGER trg_review_records_activity
AFTER INSERT ON review_records
FOR EACH ROW EXECUTE FUNCTION record_user_activity_event('review');

COMMIT;
