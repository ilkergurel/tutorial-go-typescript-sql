CREATE TABLE requirements (
  id    int  PRIMARY KEY,
  code  text NOT NULL UNIQUE,
  title text NOT NULL
);
CREATE TABLE test_runs (
  id         bigint       PRIMARY KEY,
  req_id     int          NOT NULL REFERENCES requirements,
  run_at     timestamptz  NOT NULL,
  seconds    numeric(6,2) NOT NULL,
  passed     boolean      NOT NULL,
  commit_sha text         NOT NULL,
  tester     text         -- NULL: an automated run
);

INSERT INTO requirements
SELECT i, 'REQ-' || lpad(i::text, 3, '0'),
       (ARRAY['Giriş', 'Arama', 'Rapor', 'Yetki', 'Dışa aktarım'])[1 + i % 5] || ' ' || i
FROM generate_series(1, 200) AS i;

-- 100,000 runs, one statement. md5 of the row number stands in for
-- random(): it looks random but is the same on every run.
INSERT INTO test_runs
SELECT i,
       1 + floor(200 * r.u ^ 2)::int,
       timestamptz '2026-01-01 09:00+03' + i * interval '257 seconds',
       round(0.2 + 30 * r.v ^ 4, 2),
       r.v < 0.9,
       left(md5((i / 25)::text), 10),
       CASE WHEN i % 4 = 0 THEN (ARRAY['ayşe', 'mehmet', 'zeynep'])[1 + i % 3] END
FROM generate_series(1, 100000) AS i,
     LATERAL decode(md5(i::text), 'hex') AS h(b),
     LATERAL (SELECT (get_byte(b, 0) * 256 + get_byte(b, 1)) / 65536.0 AS u,
                     (get_byte(b, 2) * 256 + get_byte(b, 3)) / 65536.0 AS v) AS r;

SELECT count(*) AS runs, count(DISTINCT req_id) AS reqs,
       count(DISTINCT commit_sha) AS commits,
       round(100.0 * count(*) FILTER (WHERE passed) / count(*), 1) AS pass_pct,
       round(100.0 * count(tester) / count(*), 1) AS manual_pct,
       min(run_at AT TIME ZONE 'Europe/Istanbul')::date AS first_day,
       max(run_at AT TIME ZONE 'Europe/Istanbul')::date AS last_day
FROM test_runs;

SELECT r.code, count(*) AS runs
FROM test_runs t JOIN requirements r ON r.id = t.req_id
GROUP BY r.code ORDER BY runs DESC, r.code LIMIT 3;

SELECT passed, round(avg(seconds), 2) AS avg_s, min(seconds), max(seconds)
FROM test_runs GROUP BY passed ORDER BY passed;
