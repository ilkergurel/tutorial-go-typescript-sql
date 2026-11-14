CREATE TABLE test_cases (id int PRIMARY KEY, title text NOT NULL, status text NOT NULL);
CREATE TABLE audit_log (
  id      bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  tbl     text  NOT NULL,
  op      text  NOT NULL,
  row_id  int   NOT NULL,
  changes jsonb NOT NULL,
  at      timestamptz NOT NULL DEFAULT now(),
  by_role text  NOT NULL DEFAULT current_user
);

-- One generic function for every audited table: TG_* says where it fired.
CREATE FUNCTION audit_row() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  old_row jsonb := CASE WHEN TG_OP <> 'INSERT' THEN to_jsonb(OLD) END;
  new_row jsonb := CASE WHEN TG_OP <> 'DELETE' THEN to_jsonb(NEW) END;
BEGIN
  INSERT INTO audit_log (tbl, op, row_id, changes)
  VALUES (TG_TABLE_NAME, TG_OP, (coalesce(new_row, old_row) ->> 'id')::int,
          CASE TG_OP
            WHEN 'UPDATE' THEN (SELECT jsonb_object_agg(k, jsonb_build_array(old_row -> k, v))
                                FROM jsonb_each(new_row) AS e(k, v)
                                WHERE old_row -> k IS DISTINCT FROM v)
            ELSE coalesce(new_row, old_row)
          END);
  RETURN NULL;  -- AFTER trigger: the return value is ignored
END
$$;

CREATE TRIGGER test_cases_audit
  AFTER INSERT OR UPDATE OR DELETE ON test_cases
  FOR EACH ROW EXECUTE FUNCTION audit_row();

INSERT INTO test_cases VALUES (1, 'Giriş', 'failed'), (2, 'Çıkış', 'passed');
UPDATE test_cases SET status = 'passed' WHERE id = 1;
DELETE FROM test_cases WHERE id = 2;

-- A change that is rolled back takes its audit row with it.
SAVEPOINT before_mistake;
UPDATE test_cases SET title = 'Yanlış' WHERE id = 1;
ROLLBACK TO before_mistake;

SELECT id, tbl, op, row_id, changes FROM audit_log ORDER BY id;
