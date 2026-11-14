-- LEARN(design.architecture.business-rules): a counter on the parent row plus a CHECK serializes the rule.
-- courses.booked is kept with each booking; the CHECK states the rule itself.

A: BEGIN;
B: BEGIN;
A: UPDATE courses SET booked = booked + 1 WHERE id = 1;
A: INSERT INTO bookings VALUES (1, 'Ayşe');
B: UPDATE courses SET booked = booked + 1 WHERE id = 1;   -- waits for A's row lock
A: COMMIT;
B: ROLLBACK;

A: SELECT c.id, c.seats, c.booked, count(b.student) AS bookings
   FROM courses AS c LEFT JOIN bookings AS b ON b.course_id = c.id GROUP BY c.id;
