-- LEARN(inside.codebase.at-10x): the schema has no user; a second learner is a migration, not a config change.
-- progress as in server/cmd/stacklab/migrations/003_learning.sql.
CREATE TABLE progress (
    lesson_id        text PRIMARY KEY,
    status           text NOT NULL DEFAULT 'viewed' CHECK (status IN ('viewed', 'ran', 'completed')),
    first_viewed_at  timestamptz NOT NULL DEFAULT now(),
    completed_at     timestamptz
);

-- learningstore.mark: status only moves forward.
PREPARE mark(text, text) AS
INSERT INTO progress (lesson_id, status) VALUES ($1, $2)
ON CONFLICT (lesson_id) DO UPDATE SET
    status = CASE
        WHEN progress.status = 'completed' THEN 'completed'
        WHEN progress.status = 'ran' AND excluded.status = 'viewed' THEN 'ran'
        ELSE excluded.status
    END;

-- Ayşe finishes the lesson; then Mehmet opens it for the first time.
EXECUTE mark('go.basics.defer', 'completed');
EXECUTE mark('go.basics.defer', 'viewed');
SELECT lesson_id, status FROM progress;

-- What a second learner costs: a user in the key, and a home for existing rows.
ALTER TABLE progress ADD COLUMN user_id text NOT NULL DEFAULT 'ayse';
ALTER TABLE progress ALTER COLUMN user_id DROP DEFAULT;
ALTER TABLE progress DROP CONSTRAINT progress_pkey;
ALTER TABLE progress ADD PRIMARY KEY (user_id, lesson_id);

PREPARE mark_for(text, text, text) AS
INSERT INTO progress (user_id, lesson_id, status) VALUES ($1, $2, $3)
ON CONFLICT (user_id, lesson_id) DO UPDATE SET
    status = CASE
        WHEN progress.status = 'completed' THEN 'completed'
        WHEN progress.status = 'ran' AND excluded.status = 'viewed' THEN 'ran'
        ELSE excluded.status
    END;

EXECUTE mark_for('mehmet', 'go.basics.defer', 'viewed');
SELECT user_id, lesson_id, status FROM progress ORDER BY user_id;
