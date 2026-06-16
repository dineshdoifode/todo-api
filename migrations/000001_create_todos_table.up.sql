-- Enable pgcrypto for gen_random_uuid() (available in Postgres 13+ as a built-in,
-- but this guard keeps older versions compatible).
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS todos (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    task       TEXT        NOT NULL,
    due_date   TIMESTAMPTZ NOT NULL,
    completed  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for the most common list query: pending todos ordered by due date.
CREATE INDEX IF NOT EXISTS idx_todos_due_date    ON todos (due_date ASC);
CREATE INDEX IF NOT EXISTS idx_todos_completed   ON todos (completed);
CREATE INDEX IF NOT EXISTS idx_todos_comp_due    ON todos (completed, due_date ASC);
