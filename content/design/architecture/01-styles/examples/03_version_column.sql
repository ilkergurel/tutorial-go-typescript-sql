-- LEARN(design.architecture.styles): optimistic locking in one UPDATE: check the version and move it forward.
-- Tabs A and B both read version 1 and save at the same moment.

A: BEGIN;
B: BEGIN;
A: UPDATE notes SET body = 'defer runs LIFO, at function exit', version = version + 1
   WHERE lesson_id = 'go.basics.defer' AND version = 1 RETURNING version;
B: UPDATE notes SET body = 'defer evaluates arguments early', version = version + 1
   WHERE lesson_id = 'go.basics.defer' AND version = 1 RETURNING version;
A: COMMIT;
B: COMMIT;

A: SELECT body, version FROM notes;
