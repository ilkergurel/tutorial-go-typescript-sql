-- Phase 1 skeleton schema. Tables for lessons/progress/notes/bookmarks/runs land in
-- Phase 2 with the lessons API; this migration only proves the pipeline end to end.
-- Every later migration is append-only: never edit an applied migration.
-- goose requires the annotation below in every file; statements before it are
-- a parse error ("unexpected state 0"). No Down section: down is a no-op.

-- +goose Up
CREATE TABLE schema_info (
    key         text PRIMARY KEY,
    value       text NOT NULL,
    updated_at  timestamptz NOT NULL DEFAULT now()
);

INSERT INTO schema_info (key, value) VALUES ('phase', '1');
