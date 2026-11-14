CREATE TABLE requirements (code text PRIMARY KEY);
INSERT INTO requirements VALUES ('REQ-1'), ('REQ-2');
CREATE TABLE test_runs (
  case_code text         NOT NULL,
  req_code  text         NOT NULL REFERENCES requirements,
  run_on    date         NOT NULL,
  seconds   numeric(6,2) NOT NULL CHECK (seconds >= 0),
  passed    boolean      NOT NULL,
  PRIMARY KEY (case_code, run_on)
);
INSERT INTO test_runs VALUES ('TC-1', 'REQ-1', '2026-11-02', 4, true);

-- All text, no constraints: loading the file can't fail. In psql:
-- \copy staging (case_code, req_code, run_on, seconds, result) FROM 'kosular.csv' WITH (FORMAT csv, HEADER, DELIMITER ';')
CREATE TABLE staging (
  row_no int GENERATED ALWAYS AS IDENTITY,
  case_code text, req_code text, run_on text, seconds text, result text
);
INSERT INTO staging (case_code, req_code, run_on, seconds, result) VALUES
  ('TC-2', 'REQ-1', '09.11.2026', '1.250,5', 'Geçti'), ('TC-3', 'REQ-2', '10.11.2026', '0,75', 'KALDI'),
  ('TC-4', 'REQ-9', '10.11.2026', '2', 'geçti'),       ('TC-5', 'REQ-1', '31.11.2026', '2,5', 'Gecti'),
  ('TC-6', 'REQ-2', '11.11.2026', '-1', 'Kaldı'),      ('TC-1', 'REQ-1', '02.11.2026', '4', 'Geçti'),
  ('TC-7', 'REQ-2', '12.11.2026', '3,1', 'Geçti'),     ('TC-7', 'REQ-2', '12.11.2026', '3,2', 'Geçti');

-- Every rule in one place: typed values, or NULL plus a reason.
CREATE VIEW checked AS
SELECT s.row_no, s.case_code, s.req_code, t.run_on, t.seconds, t.passed,
  array_remove(ARRAY[
    CASE WHEN NOT EXISTS (SELECT FROM requirements r WHERE r.code = s.req_code) THEN 'unknown requirement' END,
    CASE WHEN t.run_on IS NULL THEN 'date is not DD.MM.YYYY' END,
    CASE WHEN t.seconds IS NULL THEN 'bad number' WHEN t.seconds < 0 THEN 'negative seconds' END,
    CASE WHEN t.passed IS NULL THEN 'result is not Geçti/Kaldı' END,
    CASE WHEN count(*) OVER (PARTITION BY s.case_code, t.run_on) > 1 THEN 'twice in file' END,
    CASE WHEN EXISTS (SELECT FROM test_runs x WHERE (x.case_code, x.run_on) = (s.case_code, t.run_on))
         THEN 'already imported' END], NULL) AS problems
FROM staging s
CROSS JOIN LATERAL (SELECT regexp_match(s.run_on, '^(\d\d)\.(\d\d)\.(\d{4})$') AS d,
                           replace(replace(s.seconds, '.', ''), ',', '.') AS n) p
CROSS JOIN LATERAL (SELECT
  CASE WHEN pg_input_is_valid(p.d[3] || '-' || p.d[2] || '-' || p.d[1], 'date')
       THEN (p.d[3] || '-' || p.d[2] || '-' || p.d[1])::date END AS run_on,
  CASE WHEN s.seconds ~ '^-?(\d{1,3}(\.\d{3})+|\d+)(,\d+)?$' AND pg_input_is_valid(p.n, 'numeric(6,2)')
       THEN p.n::numeric(6,2) END AS seconds,
  CASE lower(s.result COLLATE "tr-x-icu") WHEN 'geçti' THEN true WHEN 'kaldı' THEN false END AS passed) t;

-- Report first, move second (see the note on why the order matters).
SELECT row_no, case_code, array_to_string(problems, '; ') AS problems
FROM checked WHERE problems <> '{}' ORDER BY row_no;

INSERT INTO test_runs (case_code, req_code, run_on, seconds, passed)
SELECT case_code, req_code, run_on, seconds, passed FROM checked WHERE problems = '{}';

SELECT * FROM test_runs ORDER BY run_on, case_code;
