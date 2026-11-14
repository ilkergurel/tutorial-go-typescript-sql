-- A case-insensitive unique index, built without blocking writes.
A: CREATE UNIQUE INDEX CONCURRENTLY users_email_lower ON users (lower(email));
-- It failed on the duplicate, and left something behind.
A: SELECT indexrelid::regclass AS index, indisvalid, indisready
   FROM pg_index WHERE indrelid = 'users'::regclass ORDER BY 1;
A: INSERT INTO users (email) VALUES ('AYSE@example.com');

-- Clean up: drop the leftover, fix the data, try again.
A: DROP INDEX CONCURRENTLY users_email_lower;
A: DELETE FROM users WHERE email <> lower(email);

-- Meanwhile B starts a read-only report that keeps one snapshot.
B: BEGIN ISOLATION LEVEL REPEATABLE READ;
B: SELECT count(*) FROM users;
A: CREATE UNIQUE INDEX CONCURRENTLY users_email_lower ON users (lower(email));
B: SELECT command, phase FROM pg_stat_progress_create_index;
B: COMMIT;
A: SELECT indexrelid::regclass AS index, indisvalid, indisready
   FROM pg_index WHERE indrelid = 'users'::regclass ORDER BY 1;
