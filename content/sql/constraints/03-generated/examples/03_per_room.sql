-- = on a plain column inside a GiST index needs btree_gist. It is a trusted
-- extension: the database owner can install it without being a superuser.
CREATE EXTENSION btree_gist;

CREATE TABLE bookings (
  id        int  PRIMARY KEY,
  room      text NOT NULL,
  during    tsrange NOT NULL,
  cancelled boolean NOT NULL DEFAULT false,
  -- Same room AND overlapping time is forbidden, among live bookings only.
  CONSTRAINT bookings_room_no_overlap
    EXCLUDE USING gist (room WITH =, during WITH &&) WHERE (NOT cancelled)
);

INSERT INTO bookings (id, room, during) VALUES
  (1, 'Galata',    '[2026-11-13 09:00, 2026-11-13 10:00)'),
  (2, 'Bosphorus', '[2026-11-13 09:00, 2026-11-13 10:00)');  -- other room
UPDATE bookings SET cancelled = true WHERE id = 1;
INSERT INTO bookings (id, room, during) VALUES
  (3, 'Galata',    '[2026-11-13 09:30, 2026-11-13 10:30)');  -- slot was freed

SELECT id, room, during, cancelled
FROM bookings
ORDER BY room, lower(during), id;

INSERT INTO bookings (id, room, during) VALUES
  (4, 'Galata',    '[2026-11-13 10:00, 2026-11-13 11:00)');
