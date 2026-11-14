-- Staged values are text. Which casts can you trust with Turkish input?

-- Dates: an ambiguous date is read according to DateStyle, a setting.
SET DateStyle = 'ISO, MDY';
SELECT '03.04.2026'::date AS read_as, pg_input_is_valid('31.12.2026', 'date') AS "31.12 ok";
SET DateStyle = 'ISO, DMY';
SELECT '03.04.2026'::date AS read_as, pg_input_is_valid('31.12.2026', 'date') AS "31.12 ok";

-- Numbers: test a value without raising an error.
SELECT v, pg_input_is_valid(v, 'numeric(6,2)') AS ok,
       (pg_input_error_info(v, 'numeric(6,2)')).message
FROM (VALUES ('12.5'), ('12,5'), ('1.250'), ('12500')) AS t(v);

-- Labels: lower() follows the collation's case rules.
SELECT v, lower(v) AS lower_default, lower(v COLLATE "tr-x-icu") AS lower_tr
FROM (VALUES ('GEÇTİ'), ('KALDI'), ('Kaldı')) AS t(v);

-- A plain cast has no soft mode: one bad value ends the statement.
SELECT v::numeric FROM (VALUES ('12.5'), ('12,5')) AS t(v);
