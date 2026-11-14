-- LEARN(design.paradigms.polymorphism): variants in one table: a kind column, per-variant columns, and a CHECK tying them together.
CREATE TABLE discounts (
  code         text PRIMARY KEY,
  kind         text NOT NULL CHECK (kind IN ('percent', 'fixed')),
  percent      int CHECK (percent BETWEEN 1 AND 100),
  amount_kurus bigint CHECK (amount_kurus > 0),
  -- Each kind has exactly its own columns: the tagged union, enforced.
  CONSTRAINT one_shape CHECK (
    (kind = 'percent' AND percent IS NOT NULL AND amount_kurus IS NULL) OR
    (kind = 'fixed' AND amount_kurus IS NOT NULL AND percent IS NULL))
);
INSERT INTO discounts VALUES
  ('DEV250', 'fixed', NULL, 25000),
  ('HOSGELDIN', 'fixed', NULL, 5000),
  ('KASIM15', 'percent', 15, NULL);

-- The switch over kind, written as a CASE in the query that needs it.
SELECT code,
       CASE kind
         WHEN 'percent' THEN 20000 - 20000 * percent / 100
         WHEN 'fixed' THEN greatest(20000 - amount_kurus, 0)
       END AS price_kurus,
       kind
FROM discounts
ORDER BY code;

-- A row with the wrong shape for its kind is refused, like the TS literal.
INSERT INTO discounts VALUES ('BOZUK', 'percent', 15, 5000);
