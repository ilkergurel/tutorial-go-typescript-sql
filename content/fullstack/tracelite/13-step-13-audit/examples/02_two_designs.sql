CREATE TABLE requirements (  -- step 2's table, abridged, with rows from before the audit began
  id uuid PRIMARY KEY, parent_id uuid REFERENCES requirements,
  code text NOT NULL, title text NOT NULL, status text NOT NULL DEFAULT 'draft');
INSERT INTO requirements (id, parent_id, code, title) VALUES
  ('01937b6e-0000-7000-8000-000000000001', NULL, 'AUTH', 'Kimlik doğrulama'),
  ('01937b6e-0000-7000-8000-000000000002', '01937b6e-0000-7000-8000-000000000001', 'AUTH-LOGIN', 'Parola ile giriş'),
  ('01937b6e-0000-7000-8000-000000000005', '01937b6e-0000-7000-8000-000000000001', 'AUTH-RESET', 'Parola sıfırlama');

-- Design 1: example 1's trigger, condensed to who, which row and what changed.
CREATE TABLE audit_log (id int GENERATED ALWAYS AS IDENTITY, actor text, code text, op text, changes jsonb);
CREATE FUNCTION audit_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO audit_log (actor, code, op, changes)
  VALUES (coalesce(nullif(current_setting('tracelite.actor', true), ''), 'db:' || session_user),
          coalesce(NEW.code, OLD.code), TG_OP,
          (SELECT jsonb_object_agg(key, jsonb_build_array(to_jsonb(OLD) -> key, value))
           FROM jsonb_each(to_jsonb(NEW)) WHERE to_jsonb(OLD) -> key IS DISTINCT FROM value));
  RETURN NULL;
END $$;
CREATE TRIGGER requirements_audit AFTER INSERT OR UPDATE OR DELETE ON requirements
  FOR EACH ROW EXECUTE FUNCTION audit_change();

-- Design 2: the service writes one event per use case, in the same transaction.
CREATE TABLE change_events (
  id      int GENERATED ALWAYS AS IDENTITY,
  actor   text NOT NULL,
  kind    text NOT NULL,   -- requirement.moved, requirement.renamed, ...
  subject text NOT NULL,   -- the requirement's code at the time
  data    jsonb NOT NULL);

-- After each step: how many records each design holds so far.
CREATE TABLE tally (step text, by_trigger bigint, by_app bigint);
CREATE PROCEDURE mark(step text) LANGUAGE sql AS $$
  INSERT INTO tally SELECT step, (SELECT count(*) FROM audit_log), (SELECT count(*) FROM change_events) $$;

-- 1. The service moves AUTH-RESET under AUTH-LOGIN.
SELECT set_config('tracelite.actor', 'mehmet', true);
UPDATE requirements SET parent_id = '01937b6e-0000-7000-8000-000000000002' WHERE code = 'AUTH-RESET';
INSERT INTO change_events (actor, kind, subject, data)
VALUES ('mehmet', 'requirement.moved', 'AUTH-RESET', '{"from": "AUTH", "to": "AUTH-LOGIN"}');
CALL mark('service: move');

-- 2. A data fix in psql (or in a migration): no service code runs, and no one is named.
SELECT set_config('tracelite.actor', '', true);
UPDATE requirements SET status = 'approved' WHERE code LIKE 'AUTH-%';
CALL mark('psql: bulk UPDATE');

-- 3. A migration adds a column with a default.
ALTER TABLE requirements ADD COLUMN priority int NOT NULL DEFAULT 3;
CALL mark('migration: ADD COLUMN');

-- 4. The service deletes a requirement and writes its event; then the transaction fails.
SAVEPOINT attempt;
DELETE FROM requirements WHERE code = 'AUTH-RESET';
INSERT INTO change_events (actor, kind, subject, data) VALUES ('ayşe', 'requirement.deleted', 'AUTH-RESET', '{}');
ROLLBACK TO SAVEPOINT attempt;
CALL mark('service: delete, rolled back');

-- 5. An administrator empties the table.
TRUNCATE requirements;
CALL mark('admin: TRUNCATE');

SELECT * FROM tally;
SELECT actor, code, op, changes FROM audit_log ORDER BY id;
SELECT actor, kind, subject, data FROM change_events;
