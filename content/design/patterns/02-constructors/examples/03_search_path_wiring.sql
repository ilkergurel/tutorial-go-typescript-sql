-- LEARN(design.patterns.constructors): search_path is the database's wiring, and binding time matters.

-- Two implementations of one name, in two schemas.
CREATE SCHEMA catalog_live;
CREATE SCHEMA catalog_fixture;
CREATE TABLE catalog_live.lessons (id text PRIMARY KEY, minutes int NOT NULL);
CREATE TABLE catalog_fixture.lessons (LIKE catalog_live.lessons INCLUDING ALL);
INSERT INTO catalog_live.lessons VALUES ('go.basics.defer', 9), ('sql.basics.select', 12);
INSERT INTO catalog_fixture.lessons VALUES ('fixture', 1);

SET LOCAL search_path = catalog_live;

-- A string body is parsed at each call: "lessons" is looked up late.
CREATE FUNCTION minutes_late() RETURNS bigint LANGUAGE sql
    AS $$ SELECT sum(minutes) FROM lessons $$;
-- A SQL-standard body (PG 14) is parsed now: "lessons" is bound at creation.
CREATE FUNCTION minutes_bound() RETURNS bigint LANGUAGE sql
    RETURN (SELECT sum(minutes) FROM lessons);
-- A pinned search_path: late lookup, but in a fixed place.
CREATE FUNCTION minutes_pinned() RETURNS bigint LANGUAGE sql
    SET search_path = catalog_live
    AS $$ SELECT sum(minutes) FROM lessons $$;

SELECT minutes_late(), minutes_bound(), minutes_pinned();

-- The caller changes the path, as a test, a tenant switch or an attacker might.
SET LOCAL search_path = catalog_fixture, catalog_live;
SELECT minutes_late(), minutes_bound(), minutes_pinned();

-- The early-bound function is a recorded dependency of the table it uses.
DROP TABLE catalog_live.lessons;
