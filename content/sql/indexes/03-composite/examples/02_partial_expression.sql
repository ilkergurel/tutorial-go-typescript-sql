CREATE TABLE tickets (
  id        bigint PRIMARY KEY,
  status    text NOT NULL,
  opened_on date NOT NULL,
  city      text NOT NULL
);
-- 2% open, the rest closed; city names arrive in capitals from an import.
INSERT INTO tickets
SELECT g,
       CASE WHEN g % 50 = 0 THEN 'open' ELSE 'closed' END,
       date '2026-01-01' + g / 100,
       (ARRAY['ISPARTA', 'IĞDIR', 'İZMİR', 'ANKARA'])[g % 4 + 1] || '-' || g
FROM generate_series(1, 20000) AS g;
ANALYZE tickets;

-- Partial: only rows matching the WHERE get an entry.
CREATE INDEX tickets_opened ON tickets (opened_on);
CREATE INDEX tickets_open_opened ON tickets (opened_on) WHERE status = 'open';
SELECT pg_relation_size('tickets_opened') / 8192 AS full_pages,
       pg_relation_size('tickets_open_opened') / 8192 AS partial_pages;

-- The query states the index's condition, so the planner can use it.
EXPLAIN (COSTS OFF)
SELECT id FROM tickets WHERE status = 'open' ORDER BY opened_on LIMIT 10;

-- Expression: index what the query compares, here Turkish lower-casing.
SELECT lower('ISPARTA') AS default_rules,
       lower('ISPARTA' COLLATE "tr-x-icu") AS turkish_rules;
CREATE INDEX tickets_city_tr ON tickets (lower(city COLLATE "tr-x-icu"));
ANALYZE tickets;

-- Same expression as the index, collation included: index scan, one match.
EXPLAIN (COSTS OFF)
SELECT id, city FROM tickets WHERE lower(city COLLATE "tr-x-icu") = 'ısparta-8';
SELECT id, city FROM tickets WHERE lower(city COLLATE "tr-x-icu") = 'ısparta-8';

-- A different expression: no index, and no match under the default rules.
EXPLAIN (COSTS OFF)
SELECT id, city FROM tickets WHERE lower(city) = 'ısparta-8';
SELECT id, city FROM tickets WHERE lower(city) = 'ısparta-8';
