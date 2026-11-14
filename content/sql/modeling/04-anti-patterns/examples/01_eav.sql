-- LEARN(sql.modeling.anti-patterns): designs that hide the schema from the database.
-- EAV: one row per (entity, attribute, value), so any attribute "fits".
CREATE TABLE test_cases (id int PRIMARY KEY, code text NOT NULL UNIQUE);
CREATE TABLE test_case_attrs (
  test_case_id int  REFERENCES test_cases,
  attr         text,
  value        text,                -- every type becomes text
  PRIMARY KEY (test_case_id, attr)
);
INSERT INTO test_cases VALUES (1, 'TC-1'), (2, 'TC-2'), (3, 'TC-3');
INSERT INTO test_case_attrs VALUES
  (1, 'priority', '1'), (1, 'automated', 'true'),
  (2, 'priority', 'high'),          -- not a number: accepted
  (3, 'prority',  '2');             -- misspelled attribute: accepted

-- Reading it back as a row takes one aggregate per attribute (a pivot).
SELECT c.code,
       max(a.value) FILTER (WHERE a.attr = 'priority')  AS priority,
       max(a.value) FILTER (WHERE a.attr = 'automated') AS automated
FROM test_cases c
LEFT JOIN test_case_attrs a ON a.test_case_id = c.id
GROUP BY c.code
ORDER BY c.code;

-- The attributes everyone has, as columns: types, NOT NULL, CHECK and
-- DEFAULT work again. jsonb keeps the per-customer extras.
CREATE TABLE test_cases_typed (
  id        int PRIMARY KEY,
  code      text NOT NULL UNIQUE,
  priority  smallint NOT NULL CHECK (priority BETWEEN 1 AND 4),
  automated boolean NOT NULL DEFAULT false,
  extra     jsonb NOT NULL DEFAULT '{}'
);
INSERT INTO test_cases_typed (id, code, priority, automated, extra)
VALUES (1, 'TC-1', 1, true, '{"browser": "Firefox"}');
SELECT code, priority, automated, extra ->> 'browser' AS browser FROM test_cases_typed;

INSERT INTO test_cases_typed (id, code, priority) VALUES (2, 'TC-2', 'high');
