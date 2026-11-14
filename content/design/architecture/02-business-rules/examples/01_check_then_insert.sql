-- LEARN(design.architecture.business-rules): a read-then-write rule races, in the app and in a trigger alike.
-- Rule: a course never has more bookings than seats. Both courses have one seat.

-- 1. The rule in the application: count, compare, insert.
A: BEGIN;
B: BEGIN;
A: SELECT count(*) FROM bookings WHERE course_id = 1;   -- 0 < 1, go ahead
B: SELECT count(*) FROM bookings WHERE course_id = 1;   -- 0 < 1, go ahead
A: INSERT INTO bookings VALUES (1, 'Ayşe');
B: INSERT INTO bookings VALUES (1, 'Işıl');
A: COMMIT;
B: COMMIT;

-- 2. The same rule "in the database", as a BEFORE INSERT trigger.
A: CREATE FUNCTION check_seats() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF (SELECT count(*) FROM bookings WHERE course_id = NEW.course_id)
       >= (SELECT seats FROM courses WHERE id = NEW.course_id) THEN
        RAISE EXCEPTION 'course % is full', NEW.course_id;
    END IF;
    RETURN NEW;
END $$;
A: CREATE TRIGGER bookings_check_seats BEFORE INSERT ON bookings
    FOR EACH ROW EXECUTE FUNCTION check_seats();
A: BEGIN;
B: BEGIN;
A: INSERT INTO bookings VALUES (2, 'Ayşe');
B: INSERT INTO bookings VALUES (2, 'Işıl');   -- A's row is invisible to B's count
A: COMMIT;
B: COMMIT;

-- One at a time, the trigger works, which is why tests pass.
A: INSERT INTO bookings VALUES (2, 'Can');

A: SELECT course_id, count(*) AS booked FROM bookings GROUP BY course_id ORDER BY course_id;
