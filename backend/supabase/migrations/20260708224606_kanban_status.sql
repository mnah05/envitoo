-- Drop old flexible column-based structure
alter table kanban_tasks drop constraint if exists kanban_tasks_column_id_fkey;
alter table kanban_tasks drop column if exists column_id;
drop table if exists kanban_columns;

-- Add enforced status column
alter table kanban_tasks
  add column status text not null default 'not_started'
  check (status = any (array['not_started'::text, 'in_progress'::text, 'completed'::text]));
