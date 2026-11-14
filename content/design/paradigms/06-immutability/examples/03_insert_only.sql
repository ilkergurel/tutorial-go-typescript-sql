-- LEARN(design.paradigms.immutability): an INSERT-only table: a change is a new row, and the past is a query.
CREATE TABLE prices (
  sku        text NOT NULL,
  kurus      int  NOT NULL CHECK (kurus >= 0),
  valid_from date NOT NULL,
  PRIMARY KEY (sku, valid_from)
);

-- Refuse UPDATE, DELETE and TRUNCATE; only the owner or a superuser can undo this.
CREATE FUNCTION refuse_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION '% is append-only: % refused', TG_TABLE_NAME, TG_OP;
END $$;
CREATE TRIGGER prices_append_only
  BEFORE UPDATE OR DELETE OR TRUNCATE ON prices
  FOR EACH STATEMENT EXECUTE FUNCTION refuse_change();

INSERT INTO prices VALUES ('ayran', 1000, '2026-10-01'), ('simit', 1250, '2026-10-01');
-- The November price change is a new row; October's stays as it was.
INSERT INTO prices VALUES ('simit', 1500, '2026-11-01');

-- What order S-1, placed on 20 October, was charged.
SELECT kurus FROM prices
WHERE sku = 'simit' AND valid_from <= '2026-10-20'
ORDER BY valid_from DESC
LIMIT 1;

-- The current list: the newest row per sku.
SELECT DISTINCT ON (sku) sku, kurus, valid_from
FROM prices
ORDER BY sku, valid_from DESC;

UPDATE prices SET kurus = 0 WHERE sku = 'ayran';
