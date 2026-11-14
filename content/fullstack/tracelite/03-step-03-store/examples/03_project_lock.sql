-- LEARN(fullstack.tracelite.step-03-store): MoveRequirement's statements, in two sessions at once.
-- A moves AUTH under EXPORT, B moves EXPORT under AUTH. Each move is legal on its own.

-- Step 2's walk up from the new parent: does it reach the moved requirement?
A: CREATE FUNCTION would_cycle(moved text, new_parent text) RETURNS boolean LANGUAGE sql AS $$
     WITH RECURSIVE up AS (
       SELECT id, parent_id FROM requirements WHERE code = new_parent
       UNION ALL SELECT r.id, r.parent_id FROM requirements r JOIN up ON r.id = up.parent_id
     ) CYCLE id SET is_cycle USING path
     SELECT EXISTS (SELECT FROM up JOIN requirements m USING (id) WHERE m.code = moved) $$;

-- 1. Check, then write, without a lock.
A: BEGIN;
B: BEGIN;
A: SELECT would_cycle('AUTH', 'EXPORT');
B: SELECT would_cycle('EXPORT', 'AUTH');
A: UPDATE requirements SET parent_id = (SELECT id FROM requirements WHERE code = 'EXPORT') WHERE code = 'AUTH';
B: UPDATE requirements SET parent_id = (SELECT id FROM requirements WHERE code = 'AUTH') WHERE code = 'EXPORT';
A: COMMIT;
B: COMMIT;
A: SELECT r.code, p.code AS parent FROM requirements r JOIN requirements p ON p.id = r.parent_id ORDER BY r.code;
A: UPDATE requirements SET parent_id = NULL;

-- 2. What MoveRequirement does: lock the project row, then check, then write.
A: BEGIN;
B: BEGIN;
A: SELECT p.name FROM projects p JOIN requirements r ON r.project_id = p.id WHERE r.code = 'AUTH' FOR UPDATE OF p;
B: SELECT p.name FROM projects p JOIN requirements r ON r.project_id = p.id WHERE r.code = 'EXPORT' FOR UPDATE OF p;
A: SELECT would_cycle('AUTH', 'EXPORT');
A: UPDATE requirements SET parent_id = (SELECT id FROM requirements WHERE code = 'EXPORT') WHERE code = 'AUTH';
A: COMMIT;
B: SELECT would_cycle('EXPORT', 'AUTH');
B: ROLLBACK;
