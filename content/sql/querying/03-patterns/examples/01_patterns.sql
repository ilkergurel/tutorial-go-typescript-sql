CREATE TABLE events (
  id        serial PRIMARY KEY,
  title     text NOT NULL,
  kind      text NOT NULL,
  starts_at timestamp NOT NULL
);

INSERT INTO events (title, kind, starts_at) VALUES
  ('Deploy window', 'ops',    '2026-11-12 09:00:00'),
  ('Istanbul meetup', 'talk', '2026-11-12 18:30:00'),
  ('istanbul study group', 'talk', '2026-11-13 19:00:00'),
  ('Quiet day', 'other', '2026-11-14 12:00:00');

-- Case-insensitive search: ILIKE lowercases both sides, then matches.
SELECT title FROM events WHERE title ILIKE '%istanbul%';

-- Conditional labels: searched CASE, NULL when nothing matches (no ELSE).
SELECT title,
       CASE WHEN kind = 'ops'  THEN 'maintenance'
            WHEN kind = 'talk' THEN 'community'
       END AS label
FROM events
ORDER BY id;

-- to_char renders timestamps the way reports want them.
SELECT title, to_char(starts_at, 'Dy DD Mon HH24:MI') AS when
FROM events
ORDER BY starts_at;
