alter table kanban_tasks add column created_by uuid references users(id);
