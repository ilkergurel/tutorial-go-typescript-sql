-- LEARN(sql.basics.psql): \d and friends are catalog queries; this is one in plain SQL.
CREATE TABLE customers (
  id    int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  email text NOT NULL UNIQUE
);
CREATE TABLE orders (
  id          int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  customer_id int NOT NULL REFERENCES customers,
  total       numeric(10,2) NOT NULL
);
CREATE VIEW big_orders AS SELECT * FROM orders WHERE total >= 1000;
COMMENT ON TABLE orders IS 'One row per checkout';

-- \d with no pattern (= \dtvmsE): the relations visible on the search
-- path, minus the system schemas. \d+ adds size and the COMMENT text.
SELECT c.relname AS "Name",
       CASE c.relkind
         WHEN 'r' THEN 'table'    WHEN 'p' THEN 'partitioned table'
         WHEN 'v' THEN 'view'     WHEN 'm' THEN 'materialized view'
         WHEN 'S' THEN 'sequence' WHEN 'f' THEN 'foreign table'
       END AS "Type",
       pg_get_userbyid(c.relowner) AS "Owner",
       obj_description(c.oid, 'pg_class') AS "Description"
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r', 'p', 'v', 'm', 'S', 'f')
  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
  AND n.nspname !~ '^pg_toast'
  AND pg_table_is_visible(c.oid)
ORDER BY "Name";
