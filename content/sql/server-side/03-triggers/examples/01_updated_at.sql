-- LEARN(sql.server-side.triggers): a BEFORE ROW trigger rewrites the row before it is stored.
CREATE TABLE requirements (
  id         int  PRIMARY KEY,
  title      text NOT NULL,
  status     text NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
-- Pretend these rows were last changed a week ago.
INSERT INTO requirements VALUES
  (1, 'Kullanıcı girişi', 'draft',    '2026-11-06 09:00+03'),
  (2, 'CSV dışa aktarım', 'approved', '2026-11-06 09:00+03'),
  (3, 'Denetim izi',      'draft',    '2026-11-06 09:00+03');

CREATE FUNCTION set_updated_at() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;  -- the row to store; RETURN NULL would skip this row silently
END
$$;

CREATE TRIGGER requirements_updated_at
  BEFORE UPDATE ON requirements
  FOR EACH ROW
  WHEN (OLD.* IS DISTINCT FROM NEW.*)
  EXECUTE FUNCTION set_updated_at();

UPDATE requirements SET status = 'approved' WHERE id = 1;        -- a real change
UPDATE requirements SET status = 'approved' WHERE id = 2;        -- same value
UPDATE requirements SET updated_at = '2000-01-01' WHERE id = 3;  -- a backdating client

SELECT id, status,
       updated_at = now() AS stamped_now,
       updated_at = '2026-11-06 09:00+03' AS kept_old_stamp
FROM requirements
ORDER BY id;
