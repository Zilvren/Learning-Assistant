BEGIN;

-- Keep creation, editing and review events distinct. Daily note targets must
-- not be satisfied by autosaves, while the heatmap can still show all study
-- activity. The source key remains one event of each kind per item per day.
CREATE OR REPLACE FUNCTION record_user_activity_event()
RETURNS TRIGGER AS $$
DECLARE
  event_day DATE;
  event_type TEXT;
BEGIN
  IF TG_ARGV[0] = 'review' THEN
    event_day := (NEW.reviewed_at AT TIME ZONE 'Asia/Shanghai')::date;
    event_type := 'review';
  ELSIF TG_ARGV[0] = 'library' THEN
    IF TG_OP = 'INSERT' THEN
      event_day := (NEW.created_at AT TIME ZONE 'Asia/Shanghai')::date;
      event_type := 'library_create';
    ELSE
      event_day := (NEW.updated_at AT TIME ZONE 'Asia/Shanghai')::date;
      event_type := 'library_update';
    END IF;
  ELSIF TG_ARGV[0] = 'error' THEN
    IF TG_OP = 'INSERT' THEN
      event_day := (NEW.created_at AT TIME ZONE 'Asia/Shanghai')::date;
      event_type := 'error_create';
    ELSE
      event_day := (NEW.updated_at AT TIME ZONE 'Asia/Shanghai')::date;
      event_type := 'error_update';
    END IF;
  ELSE
    event_day := CURRENT_DATE;
    event_type := TG_ARGV[0];
  END IF;

  INSERT INTO user_activity_events (user_id, activity_date, event_type, source_key)
  VALUES (NEW.user_id, event_day, event_type, event_type || ':' || NEW.id || ':' || event_day::text)
  ON CONFLICT (source_key) DO NOTHING;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMIT;
