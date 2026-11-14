CREATE TABLE customers (
  id    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  phone text NOT NULL
);
INSERT INTO customers (phone)
SELECT CASE WHEN g % 4 = 0 THEN '+49 30 ' ELSE '+90 212 ' END || (5550000 + g)
FROM generate_series(1, 1000) AS g;

-- Goal: a required country column, derived from each phone number.
-- Step 1: add it nullable and without a default: a catalog-only change.
ALTER TABLE customers ADD COLUMN country text;

-- Step 2: the rule, NOT VALID. No scan, so the exclusive lock is brief, and
-- from now on every inserted or updated row must have a country...
ALTER TABLE customers
  ADD CONSTRAINT customers_country_not_null NOT NULL country NOT VALID;

-- ...while the existing rows are still NULL, and the catalog admits it.
SELECT count(*) AS rows_without_country,
       (SELECT convalidated FROM pg_constraint
        WHERE conrelid = 'customers'::regclass
          AND conname = 'customers_country_not_null') AS validated
FROM customers
WHERE country IS NULL;

-- Step 3: backfill in batches; in production, one short transaction each.
UPDATE customers SET country = CASE WHEN phone LIKE '+90%' THEN 'TR' ELSE 'DE' END
WHERE id BETWEEN 1 AND 500;
UPDATE customers SET country = CASE WHEN phone LIKE '+90%' THEN 'TR' ELSE 'DE' END
WHERE id BETWEEN 501 AND 1000;

-- Step 4: VALIDATE scans under SHARE UPDATE EXCLUSIVE: reads and writes go on.
ALTER TABLE customers VALIDATE CONSTRAINT customers_country_not_null;

SELECT count(*) AS rows_without_country,
       (SELECT convalidated FROM pg_constraint
        WHERE conrelid = 'customers'::regclass
          AND conname = 'customers_country_not_null') AS validated
FROM customers
WHERE country IS NULL;

-- The constraint has guarded every write since step 2.
INSERT INTO customers (phone) VALUES ('+90 312 5550000');
