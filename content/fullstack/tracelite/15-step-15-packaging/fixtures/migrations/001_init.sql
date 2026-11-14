-- +goose Up
-- Abridged: step 2 has the whole file.
CREATE TABLE projects (
  id         uuid PRIMARY KEY DEFAULT uuidv7(),
  name       text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
