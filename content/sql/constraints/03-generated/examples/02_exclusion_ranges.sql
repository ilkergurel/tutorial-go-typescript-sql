-- One meeting room: no two bookings may overlap. && means "overlaps".
CREATE TABLE bookings (
  id     int  PRIMARY KEY,
  who    text NOT NULL,
  during tsrange NOT NULL,
  CONSTRAINT bookings_no_overlap EXCLUDE USING gist (during WITH &&)
);

INSERT INTO bookings VALUES
  (1, 'Ayşe',   '[2026-11-13 09:00, 2026-11-13 10:00)'),
  -- Half-open ranges: one ends at 10:00, the next starts at 10:00. No overlap.
  (2, 'Mehmet', '[2026-11-13 10:00, 2026-11-13 11:30)');

SELECT id, who, during FROM bookings ORDER BY lower(during);

-- UNIQUE (during) would only reject the exact same range twice.
INSERT INTO bookings VALUES
  (3, 'Zeynep', '[2026-11-13 11:00, 2026-11-13 12:00)');
