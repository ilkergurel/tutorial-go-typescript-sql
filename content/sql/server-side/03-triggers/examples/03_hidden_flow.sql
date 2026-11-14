CREATE TABLE requirements (
  id         int  PRIMARY KEY,
  code       text NOT NULL,
  status     text NOT NULL,
  deleted_at timestamptz
);
CREATE TABLE test_cases (
  id             int  PRIMARY KEY,
  requirement_id int  NOT NULL REFERENCES requirements,
  status         text NOT NULL
);
INSERT INTO requirements VALUES (1, 'REQ-LOGIN', 'failing', NULL), (2, 'REQ-OLD', 'passing', NULL);
INSERT INTO test_cases VALUES (10, 1, 'failed'), (11, 1, 'failed'), (12, 1, 'passed');

-- Trigger 1: turn every DELETE into a soft delete.
CREATE FUNCTION soft_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  UPDATE requirements SET deleted_at = now() WHERE id = OLD.id;
  RETURN NULL;  -- skip the real DELETE
END
$$;
CREATE TRIGGER requirements_soft_delete BEFORE DELETE ON requirements
  FOR EACH ROW EXECUTE FUNCTION soft_delete();

-- Trigger 2: roll test results up into the requirement's status.
CREATE FUNCTION roll_up() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE NOTICE 'roll_up for test case %', NEW.id;
  UPDATE requirements
  SET status = CASE WHEN (SELECT bool_and(status = 'passed') FROM test_cases
                          WHERE requirement_id = NEW.requirement_id)
                    THEN 'passing' ELSE 'failing' END
  WHERE id = NEW.requirement_id;
  RETURN NULL;
END
$$;
CREATE TRIGGER test_cases_roll_up AFTER UPDATE ON test_cases
  FOR EACH ROW EXECUTE FUNCTION roll_up();

-- The DELETE reports 0 rows, yet the row changed.
DELETE FROM requirements WHERE code = 'REQ-OLD';
SELECT code, status, deleted_at IS NOT NULL AS deleted FROM requirements ORDER BY id;

-- One UPDATE statement; three trigger calls and three hidden UPDATEs.
UPDATE test_cases SET status = 'passed' WHERE requirement_id = 1;
SELECT code, status FROM requirements ORDER BY id;

-- Nothing in either statement mentions this. Only the catalog does.
SELECT event_object_table AS on_table, trigger_name, action_timing, event_manipulation AS event
FROM information_schema.triggers
WHERE event_object_schema = current_schema()
ORDER BY on_table, trigger_name;
