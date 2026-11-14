SET TimeZone = 'Europe/Istanbul';

CREATE TABLE requirements (
  id         bigint      PRIMARY KEY,
  code       text        NOT NULL,
  title      text        NOT NULL,
  owner      text,
  created_at timestamptz NOT NULL
);
CREATE TABLE test_cases (
  id             bigint PRIMARY KEY,
  requirement_id bigint NOT NULL REFERENCES requirements,
  status         text   NOT NULL
);
INSERT INTO requirements VALUES
  (1, 'REQ-1', 'Giriş ekranı', 'Zeynep', '2026-11-13 09:00+03'),
  (2, 'REQ-2', 'Dışa aktarım', NULL,     '2026-11-13 10:30+03');
INSERT INTO test_cases VALUES
  (9007199254740993, 1, 'passed'), (11, 1, 'failed');

-- json keeps the keys in the order you wrote them; jsonb sorts them
-- (shorter first) and prints its own canonical spacing.
SELECT json_build_object('code', code, 'title', title, 'owner', owner, 'id', id)
         AS as_json,
       jsonb_build_object('code', code, 'title', title, 'owner', owner, 'id', id)
         AS as_jsonb
FROM requirements
WHERE id = 1;

-- A whole row: keys come from column names; - removes what the client
-- shouldn't get, jsonb_strip_nulls the fields that are null.
SELECT jsonb_strip_nulls(to_jsonb(r) - 'id') AS body
FROM requirements AS r
ORDER BY r.id;

-- One response document per requirement. timestamptz becomes ISO 8601 in
-- the session time zone; ids also go out as strings for JavaScript.
SELECT jsonb_build_object(
         'code', r.code,
         'created_at', r.created_at,
         'tests', (SELECT coalesce(jsonb_agg(jsonb_build_object(
                                     'id', t.id, 'id_str', t.id::text,
                                     'status', t.status) ORDER BY t.id), '[]')
                   FROM test_cases AS t
                   WHERE t.requirement_id = r.id)) AS body
FROM requirements AS r
ORDER BY r.id;
