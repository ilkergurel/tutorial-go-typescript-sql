CREATE TABLE scores (
  id     int PRIMARY KEY,
  player text NOT NULL,
  points int            -- NULL: hasn't played yet
);
INSERT INTO scores VALUES
  (1, 'deniz', 70), (2, 'emre', NULL), (3, 'selin', 90),
  (4, 'umut', 85),  (5, 'zeynep', 85), (6, 'ali', 60);

-- No ORDER BY means no promised order. Updating deniz (id 1) wrote a
-- new row version at the end of the table, so he's no longer "first".
UPDATE scores SET points = 72 WHERE id = 1;
SELECT id, player FROM scores LIMIT 3;

-- NULL sorts as larger than any value, so DESC puts it first.
SELECT player, points FROM scores ORDER BY points DESC LIMIT 2;

-- A leaderboard: NULLs last, and a unique tie-breaker (umut and zeynep
-- both have 85) so the cut at 2 is always the same rows.
SELECT player, points FROM scores
ORDER BY points DESC NULLS LAST, id
LIMIT 2;

-- Page 2 of the same ordering. OFFSET still computes the skipped rows.
SELECT player, points FROM scores
ORDER BY points DESC NULLS LAST, id
LIMIT 2 OFFSET 2;

-- "Top 2, including anyone tied with the last one" (PG 13+). The outer
-- ORDER BY fixes the display order, which ties leave open.
SELECT player, points
FROM (SELECT * FROM scores
      ORDER BY points DESC NULLS LAST
      FETCH FIRST 2 ROWS WITH TIES) AS top
ORDER BY points DESC, player;
