-- LEARN(sql.indexes.stats-pagination): plans are only as good as the row estimates behind them.
CREATE TABLE addresses (id int PRIMARY KEY, city text NOT NULL, district text NOT NULL);
-- Ten districts in five cities: each district belongs to exactly one city.
INSERT INTO addresses
SELECT g, d.city, d.district
FROM generate_series(1, 20000) AS g
JOIN (VALUES (0, 'İstanbul', 'Kadıköy'), (1, 'İstanbul', 'Beşiktaş'),
             (2, 'Ankara', 'Çankaya'),   (3, 'Ankara', 'Keçiören'),
             (4, 'İzmir', 'Konak'),      (5, 'İzmir', 'Bornova'),
             (6, 'Bursa', 'Nilüfer'),    (7, 'Bursa', 'Osmangazi'),
             (8, 'Antalya', 'Muratpaşa'), (9, 'Antalya', 'Konyaaltı'))
  AS d (k, city, district) ON d.k = g % 10;
ANALYZE addresses;

-- The planner's row estimate next to the real count. The condition is pasted
-- into the SQL text: fine for a lesson helper, never for user input.
CREATE FUNCTION estimate_vs_actual(cond text, OUT estimated bigint, OUT actual bigint)
LANGUAGE plpgsql AS $$
DECLARE plan json;
BEGIN
  EXECUTE 'EXPLAIN (FORMAT JSON) SELECT * FROM addresses WHERE ' || cond INTO plan;
  estimated := (plan -> 0 -> 'Plan' ->> 'Plan Rows')::bigint;
  EXECUTE 'SELECT count(*) FROM addresses WHERE ' || cond INTO actual;
END $$;

-- Each condition alone is estimated exactly; together they are multiplied.
SELECT c AS condition, e.estimated, e.actual
FROM unnest(ARRAY[$$city = 'İstanbul'$$, $$district = 'Kadıköy'$$,
                  $$city = 'İstanbul' AND district = 'Kadıköy'$$]) AS c,
     LATERAL estimate_vs_actual(c) AS e;

-- Extended statistics record that district determines city.
CREATE STATISTICS addresses_city_district (dependencies) ON city, district FROM addresses;
ANALYZE addresses;
SELECT * FROM estimate_vs_actual($$city = 'İstanbul' AND district = 'Kadıköy'$$);

-- A bulk import after the last ANALYZE: 5,000 addresses in a city it never saw.
INSERT INTO addresses
SELECT g, 'Eskişehir', 'Tepebaşı' FROM generate_series(20001, 25000) AS g;
SELECT * FROM estimate_vs_actual($$city = 'Eskişehir'$$);
ANALYZE addresses;
SELECT * FROM estimate_vs_actual($$city = 'Eskişehir'$$);
