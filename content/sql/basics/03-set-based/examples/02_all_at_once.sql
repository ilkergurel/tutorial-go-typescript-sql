-- A statement acts on the whole set at once, against the table as it
-- was when the statement started.
CREATE TABLE shifts (nurse text PRIMARY KEY, day_ward text, night_ward text);
INSERT INTO shifts VALUES ('elif', 'cardiology', 'icu'), ('kerem', 'er', 'pediatrics');

-- Every right-hand side reads the old row: a swap needs no temp variable.
UPDATE shifts SET day_ward = night_ward, night_ward = day_ward;
SELECT nurse, day_ward, night_ward FROM shifts ORDER BY nurse;

-- Reading and writing the same table: the SELECT sees the 3 rows that
-- existed when the INSERT began, never the rows it is adding.
CREATE TABLE nums (n int);
INSERT INTO nums VALUES (1), (2), (3);
INSERT INTO nums SELECT n + 3 FROM nums;
SELECT n FROM nums ORDER BY n;

-- The crack in the model: a plain UNIQUE is checked row by row, so
-- shifting every position by one collides with a row not yet moved.
CREATE TABLE queue (pos int UNIQUE, job text);
INSERT INTO queue VALUES (1, 'backup'), (2, 'report'), (3, 'email');
UPDATE queue SET pos = pos + 1;
