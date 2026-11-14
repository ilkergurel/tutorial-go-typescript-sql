-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION notify_change() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  project uuid;
BEGIN
  IF TG_TABLE_NAME = 'test_runs' THEN
    SELECT project_id INTO project FROM test_cases
    WHERE id = CASE TG_OP WHEN 'DELETE' THEN OLD.test_case_id ELSE NEW.test_case_id END;
  ELSE
    project := CASE TG_OP WHEN 'DELETE' THEN OLD.project_id ELSE NEW.project_id END;
  END IF;
  PERFORM pg_notify('tracelite_changes',
    json_build_object('projectId', project, 'topic', TG_ARGV[0])::text);
  RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER requirements_notify AFTER INSERT OR UPDATE OR DELETE ON requirements
  FOR EACH ROW EXECUTE FUNCTION notify_change('requirements');
CREATE TRIGGER trace_links_notify AFTER INSERT OR UPDATE OR DELETE ON trace_links
  FOR EACH ROW EXECUTE FUNCTION notify_change('trace-links');
CREATE TRIGGER test_runs_notify AFTER INSERT OR UPDATE OR DELETE ON test_runs
  FOR EACH ROW EXECUTE FUNCTION notify_change('test-runs');
