CREATE EXTENSION btree_gist;

-- Bitemporal: valid_at is when a price applies in the world,
-- recorded_at is when the database believed it. Rows are never
-- overwritten; a change of mind closes recorded_at and adds new rows.
CREATE TABLE prices (
  sku         text,
  price       numeric(10, 2) NOT NULL,
  valid_at    daterange NOT NULL,
  recorded_at daterange NOT NULL,
  -- WITHOUT OVERLAPS handles one period; two periods need EXCLUDE.
  EXCLUDE USING gist (sku WITH =, valid_at WITH &&, recorded_at WITH &&)
);

-- 1 January: the cable costs 129.00, until further notice.
INSERT INTO prices VALUES ('usb-c', 129.00, '[2026-01-01,)', '[2026-01-01,)');

-- 1 March: from today it costs 149.00. The old belief is closed, and
-- the new belief is two rows: 129.00 until today, 149.00 from today.
UPDATE prices SET recorded_at = '[2026-01-01,2026-03-01)'
WHERE sku = 'usb-c' AND upper_inf(recorded_at);
INSERT INTO prices VALUES
  ('usb-c', 129.00, '[2026-01-01,2026-03-01)', '[2026-03-01,)'),
  ('usb-c', 149.00, '[2026-03-01,)',            '[2026-03-01,)');

-- 20 March: 149.00 was a typo, it should have been 139.00 all along.
UPDATE prices SET recorded_at = '[2026-03-01,2026-03-20)'
WHERE sku = 'usb-c' AND price = 149.00 AND upper_inf(recorded_at);
INSERT INTO prices VALUES ('usb-c', 139.00, '[2026-03-01,)', '[2026-03-20,)');

-- The price on 10 March, as the database knew it on three dates.
SELECT k.known_on, p.price
FROM (VALUES (date '2026-02-15'), (date '2026-03-15'), (date '2026-11-13')) AS k(known_on)
JOIN prices p
  ON p.sku = 'usb-c' AND p.valid_at @> date '2026-03-10' AND p.recorded_at @> k.known_on
ORDER BY k.known_on;

-- A second current belief about March would contradict the first.
INSERT INTO prices VALUES ('usb-c', 145.00, '[2026-03-15,)', '[2026-03-20,)');
