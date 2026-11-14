-- Phase 3: run history. Every executed example lands here so the owner can
-- revisit what they ran (spec §9: run history is part of the learning loop).
-- result holds the full typed RunResult JSON (one shape for all runtimes).

-- +goose Up
CREATE TABLE runs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id   text NOT NULL,
    example_id  text NOT NULL,
    runtime     text NOT NULL,
    code        text NOT NULL,
    result      jsonb NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX runs_created_at_idx ON runs (created_at DESC);
