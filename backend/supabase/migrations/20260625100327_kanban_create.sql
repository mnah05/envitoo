CREATE TABLE kanban_columns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    event_id UUID NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    position INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, name),
    UNIQUE (event_id, position)
);

CREATE TABLE kanban_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    event_id UUID NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    column_id UUID NOT NULL REFERENCES kanban_columns (id) ON DELETE CASCADE,
    assigned_to UUID REFERENCES users (id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    description TEXT,
    deadline TIMESTAMPTZ,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tasks_event_id ON kanban_tasks (event_id);

CREATE INDEX idx_tasks_column_id ON kanban_tasks (column_id);

CREATE INDEX idx_tasks_assigned_to ON kanban_tasks (assigned_to);