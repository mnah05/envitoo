-- Remove position — ordering now handled by created_at in queries
alter table kanban_tasks drop column if exists position;

-- Track latest assignment (who assigned, not full history)
alter table kanban_tasks add column assigned_by uuid references users(id);

-- Unique position constraint discussion is now moot — no position column
alter table kanban_tasks
  add constraint kanban_tasks_event_id_status_created_at_idx
  check (true); -- placeholder, remove if not needed; see note below

-- Status change audit log
create table kanban_task_status_log (
  id uuid primary key default gen_random_uuid(),
  task_id uuid not null references kanban_tasks(id) on delete cascade,
  changed_by uuid references users(id),
  old_status text,
  new_status text not null,
  changed_at timestamptz not null default now()
);
