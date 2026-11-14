-- integer: 4 bytes, up to 2,147,483,647. bigint: 8 bytes, up to about 9.2e18.
SELECT pg_column_size(1::integer) AS integer_bytes,
       pg_column_size(1::bigint)  AS bigint_bytes;

-- Division of two integers truncates toward zero, as in Go, C and Java.
-- Python's -7 // 2 is -4; here -7 / 2 is -3. One numeric operand changes the type.
SELECT 7 / 2 AS int_div, -7 / 2 AS neg_div, 7 % 2 AS remainder, 7 / 2.0 AS numeric_div;

CREATE TABLE counters (
  page text    PRIMARY KEY,
  hits integer NOT NULL
);
INSERT INTO counters VALUES ('home', 2147483647), ('about', 1);

-- Aggregates widen their result: sum(integer) returns bigint, so this works.
SELECT sum(hits) AS total, pg_typeof(sum(hits)) AS total_type FROM counters;

-- Row arithmetic does not widen. One more hit on 'home' leaves integer's
-- range, and Postgres raises an error instead of wrapping around.
UPDATE counters SET hits = hits + 1 WHERE page = 'home';
