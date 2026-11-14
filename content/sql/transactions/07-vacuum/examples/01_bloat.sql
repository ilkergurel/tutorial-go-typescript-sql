-- Runtime sql-nontx: every statement commits on its own, as in psql.
-- TEMP: only this session sees the table, so no other session's open
-- transaction can hold VACUUM back here (example 2 shows that happening).
CREATE TEMP TABLE counters (id int PRIMARY KEY, hits int NOT NULL, note text NOT NULL);
INSERT INTO counters SELECT g, 0, repeat('x', 100) FROM generate_series(1, 10000) AS g;

CREATE TABLE sizes (n int GENERATED ALWAYS AS IDENTITY, step text, bytes bigint);
INSERT INTO sizes (step, bytes) VALUES ('10000 rows loaded', pg_relation_size('counters'));

UPDATE counters SET hits = hits + 1;
INSERT INTO sizes (step, bytes) VALUES ('UPDATE every row', pg_relation_size('counters'));
VACUUM counters;
INSERT INTO sizes (step, bytes) VALUES ('VACUUM', pg_relation_size('counters'));
UPDATE counters SET hits = hits + 1;
INSERT INTO sizes (step, bytes) VALUES ('UPDATE every row again', pg_relation_size('counters'));
DELETE FROM counters WHERE id % 2 = 0;
INSERT INTO sizes (step, bytes) VALUES ('DELETE every other row', pg_relation_size('counters'));
VACUUM counters;
INSERT INTO sizes (step, bytes) VALUES ('VACUUM', pg_relation_size('counters'));
VACUUM FULL counters;
INSERT INTO sizes (step, bytes) VALUES ('VACUUM FULL', pg_relation_size('counters'));

-- Byte counts depend on the build; the ratio to the first one doesn't.
SELECT n, step,
       round(bytes::numeric / first_value(bytes) OVER (ORDER BY n), 1) AS size_vs_loaded
FROM sizes ORDER BY n;
