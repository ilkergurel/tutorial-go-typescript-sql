CREATE TABLE requirements (id int PRIMARY KEY, code text NOT NULL);
CREATE TABLE test_cases (
  id             int PRIMARY KEY,
  requirement_id int NOT NULL REFERENCES requirements,
  title          text NOT NULL
);
INSERT INTO requirements VALUES (1, 'REQ-LOGIN'), (2, 'REQ-RESET'), (3, 'REQ-EXPORT');
INSERT INTO test_cases VALUES
  (10, 1, 'Valid password'), (11, 1, 'Locked account'), (12, 2, 'Reset e-mail sent');

-- Tests per requirement, including requirements with none.
-- GROUP BY r.id lets us select r.code: the primary key determines it.
SELECT r.code,
       count(*)    AS rows_in_group,
       count(t.id) AS tests
FROM requirements r
LEFT JOIN test_cases t ON t.requirement_id = r.id
GROUP BY r.id
ORDER BY r.code;

-- Grouping text case-insensitively, with Turkish city names typed three ways.
CREATE TABLE customers (id int PRIMARY KEY, city text NOT NULL);
INSERT INTO customers VALUES
  (1, 'Diyarbakır'), (2, 'DİYARBAKIR'), (3, 'diyarbakır'),
  (4, 'Iğdır'), (5, 'IĞDIR');

-- The database default (C.UTF-8) lowercases I to i, never to ı.
SELECT lower(city) AS city_key, count(*) AS customers
FROM customers
GROUP BY lower(city)
ORDER BY city_key;

-- Turkish rules: I -> ı and İ -> i. Each city is now one group.
SELECT lower(city COLLATE "tr-x-icu") AS city_key, count(*) AS customers
FROM customers
GROUP BY lower(city COLLATE "tr-x-icu")
ORDER BY city_key;
