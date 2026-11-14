-- Phase 4: learning features. Single-user app, so lesson_id is the natural key
-- everywhere — no user table, no joins (docs/ARCHITECTURE.md ERD).

-- +goose Up
CREATE TABLE progress (
    lesson_id        text PRIMARY KEY,
    status           text NOT NULL DEFAULT 'viewed' CHECK (status IN ('viewed', 'ran', 'completed')),
    first_viewed_at  timestamptz NOT NULL DEFAULT now(),
    completed_at     timestamptz
);

CREATE TABLE notes (
    lesson_id   text PRIMARY KEY,
    body        text NOT NULL DEFAULT '',
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE bookmarks (
    lesson_id   text PRIMARY KEY,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- answers maps quiz question index (as text) to the chosen option index.
CREATE TABLE quiz_answers (
    lesson_id   text PRIMARY KEY,
    answers     jsonb NOT NULL DEFAULT '{}',
    updated_at  timestamptz NOT NULL DEFAULT now()
);
