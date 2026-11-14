-- LEARN(fullstack.tracelite.step-13-audit): a trigger writes the audit trail; the application names the user with set_config.
CREATE TABLE requirements (  -- step 2's tables, abridged
  id uuid PRIMARY KEY, project_id uuid NOT NULL, parent_id uuid REFERENCES requirements,
  code text NOT NULL, title text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE trace_links (
  project_id uuid NOT NULL, requirement_id uuid NOT NULL REFERENCES requirements ON DELETE CASCADE,
  test_case_id uuid NOT NULL, PRIMARY KEY (project_id, requirement_id, test_case_id));

-- migrations/006_audit.sql (test_cases_audit, the same as requirements_audit, left out)
-- +goose Up
CREATE TABLE audit_log (
  id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  at         timestamptz NOT NULL DEFAULT now(),          -- when the transaction started
  xact       xid8 NOT NULL DEFAULT pg_current_xact_id(),  -- groups one transaction's rows
  actor      text NOT NULL,
  action     text NOT NULL DEFAULT '',                    -- the application's name for the change, if any
  project_id uuid NOT NULL,                               -- no foreign key: the record outlives the project
  table_name text NOT NULL,
  row_id     uuid NOT NULL,
  op         text NOT NULL,
  changes    jsonb NOT NULL
);
CREATE INDEX audit_log_row ON audit_log (row_id, id);
CREATE INDEX audit_log_project ON audit_log (project_id, id);
-- +goose StatementBegin
CREATE FUNCTION audit_change() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  old_row jsonb := CASE WHEN TG_OP <> 'INSERT' THEN to_jsonb(OLD) - 'created_at' - 'updated_at' END;
  new_row jsonb := CASE WHEN TG_OP <> 'DELETE' THEN to_jsonb(NEW) - 'created_at' - 'updated_at' END;
  diff    jsonb;
BEGIN
  IF TG_OP = 'UPDATE' THEN
    SELECT jsonb_object_agg(key, jsonb_build_array(old_row -> key, value)) INTO diff
    FROM jsonb_each(new_row) WHERE old_row -> key IS DISTINCT FROM value;
    IF diff IS NULL THEN RETURN NULL; END IF;  -- only the timestamps changed: nothing to record
  END IF;
  INSERT INTO audit_log (actor, action, project_id, table_name, row_id, op, changes)
  VALUES (coalesce(nullif(current_setting('tracelite.actor', true), ''), 'db:' || session_user),
          coalesce(current_setting('tracelite.action', true), ''),
          (coalesce(new_row, old_row) ->> 'project_id')::uuid, TG_TABLE_NAME,
          (coalesce(new_row, old_row) ->> TG_ARGV[0])::uuid, TG_OP,
          coalesce(diff, new_row, old_row));
  RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER requirements_audit AFTER INSERT OR UPDATE OR DELETE ON requirements
  FOR EACH ROW EXECUTE FUNCTION audit_change('id');
CREATE TRIGGER trace_links_audit AFTER INSERT OR UPDATE OR DELETE ON trace_links
  FOR EACH ROW EXECUTE FUNCTION audit_change('requirement_id');  -- a link belongs to its requirement's history
-- end of 006_audit.sql

-- Request 1, by Ayşe. Store.WithTx starts every transaction like this.
SELECT set_config('tracelite.actor', 'ayşe', true);
INSERT INTO requirements (id, project_id, parent_id, code, title) VALUES
  ('01937b6e-0000-7000-8000-000000000001', '01937b6e-0000-7000-8000-00000000000a', NULL, 'AUTH', 'Kimlik doğrulama'),
  ('01937b6e-0000-7000-8000-000000000002', '01937b6e-0000-7000-8000-00000000000a', '01937b6e-0000-7000-8000-000000000001', 'AUTH-LOGIN', 'Parola ile giriş'),
  ('01937b6e-0000-7000-8000-000000000005', '01937b6e-0000-7000-8000-00000000000a', '01937b6e-0000-7000-8000-000000000001', 'AUTH-RESET', 'Parola sıfırlama');
INSERT INTO trace_links VALUES
  ('01937b6e-0000-7000-8000-00000000000a', '01937b6e-0000-7000-8000-000000000005', '01937b6e-0000-7000-8000-0000000000c3');

-- Request 2, by Mehmet: a move the service labels, an edit, then the same edit saved again.
SELECT set_config('tracelite.actor', 'mehmet', true), set_config('tracelite.action', 'move', true);
UPDATE requirements SET parent_id = '01937b6e-0000-7000-8000-000000000002', updated_at = now() WHERE code = 'AUTH-RESET';
SELECT set_config('tracelite.action', '', true);
UPDATE requirements SET title = 'Parola sıfırlama e-postası', updated_at = now() WHERE code = 'AUTH-RESET';
UPDATE requirements SET title = 'Parola sıfırlama e-postası', updated_at = now() WHERE code = 'AUTH-RESET';

-- Request 3, by Ayşe: deleting AUTH-RESET cascades to its link.
SELECT set_config('tracelite.actor', 'ayşe', true);
DELETE FROM requirements WHERE code = 'AUTH-RESET';

-- The history of AUTH-RESET, as the detail page shows it, after the requirement is gone.
SELECT id, actor, action, table_name, op, changes - 'id' - 'project_id' AS changes
FROM audit_log WHERE row_id = '01937b6e-0000-7000-8000-000000000005' ORDER BY id;
