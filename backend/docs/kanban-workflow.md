# Kanban API — Endpoints & Pseudo Logic

Shared middleware (runs on all `/events/:eventId/kanban/*` routes):

```
func WithEventRole(next):
    userID := auth.CurrentUser(r)
    eventID := urlParam("eventId")

    isOwner := query("SELECT organizer_id = $1 FROM events WHERE id = $2", userID, eventID)

    isAdmin := false
    if !isOwner:
        isAdmin := query(`
            SELECT EXISTS (
              SELECT 1 FROM volunteer_applications va
              JOIN volunteer_roles vr ON vr.id = va.role_id
              WHERE va.user_id = $1 AND va.event_id = $2
                AND va.status = 'approved' AND vr.is_admin = true
            )`, userID, eventID)

    ctx.set("userID", userID)
    ctx.set("isAdminOrOwner", isOwner || isAdmin)
    next(ctx)
```

---

## 1. POST /events/:eventId/kanban/tasks — Create task
**Auth:** admin/owner only

```
if !ctx.isAdminOrOwner: return 403

body := parse(title, description, deadline, assigned_to?)

BEGIN TX
    task := INSERT INTO kanban_tasks
        (event_id, title, description, deadline, created_by, assigned_to, assigned_by, status)
        VALUES ($eventID, body.title, body.description, body.deadline,
                ctx.userID,
                body.assigned_to,                      -- nullable
                body.assigned_to != nil ? ctx.userID : nil,  -- assigned_by only set if assigning now
                'not_started')
        RETURNING *
COMMIT

return 201, task
```

---

## 2. GET /events/:eventId/kanban/tasks — List tasks
**Auth:** anyone associated with the event (admin/owner/assignee/team) — read-only

```
if !ctx.isAdminOrOwner && !isEventParticipant(ctx.userID, eventID): return 403

tasks := SELECT * FROM kanban_tasks
         WHERE event_id = $eventID
         ORDER BY status, created_at ASC   -- or updated_at, your call

return 200, group_by_status(tasks)   -- or return flat list, group client-side
```

---

## 3. GET /events/:eventId/kanban/tasks/:taskId — Get single task
**Auth:** same as list (read-only, broad access)

```
task := SELECT * FROM kanban_tasks WHERE id = $taskId AND event_id = $eventID
if task == nil: return 404

if !ctx.isAdminOrOwner && !isEventParticipant(ctx.userID, eventID): return 403

return 200, task
```

---

## 4. PATCH /events/:eventId/kanban/tasks/:taskId — Edit task details
**Auth:** admin/owner only (title, description, deadline — NOT status, NOT assignment)

```
if !ctx.isAdminOrOwner: return 403

task := SELECT * FROM kanban_tasks WHERE id = $taskId AND event_id = $eventID
if task == nil: return 404

body := parse(title?, description?, deadline?)

UPDATE kanban_tasks
SET title = COALESCE(body.title, title),
    description = COALESCE(body.description, description),
    deadline = COALESCE(body.deadline, deadline),
    updated_at = now()
WHERE id = $taskId

return 200, updatedTask
```

---

## 5. DELETE /events/:eventId/kanban/tasks/:taskId — Delete task
**Auth:** admin/owner only

```
if !ctx.isAdminOrOwner: return 403

task := SELECT * FROM kanban_tasks WHERE id = $taskId AND event_id = $eventID
if task == nil: return 404

DELETE FROM kanban_tasks WHERE id = $taskId
-- kanban_task_status_log rows cascade-delete via FK (on delete cascade)

return 204
```

---

## 6. PATCH /events/:eventId/kanban/tasks/:taskId/assign — Assign/reassign
**Auth:** admin/owner only

```
if !ctx.isAdminOrOwner: return 403

task := SELECT * FROM kanban_tasks WHERE id = $taskId AND event_id = $eventID
if task == nil: return 404

body := parse(assigned_to)   -- target user id (can be ctx.userID for self-assign)

-- optional: verify assigned_to is actually a participant/volunteer of this event
if !isEventParticipant(body.assigned_to, eventID): return 400 "invalid assignee"

UPDATE kanban_tasks
SET assigned_to = body.assigned_to,
    assigned_by = ctx.userID,
    updated_at = now()
WHERE id = $taskId

return 200, updatedTask
```

---

## 7. PATCH /events/:eventId/kanban/tasks/:taskId/status — Update status
**Auth:** admin/owner (any task) OR assignee (own task only)

```
task := SELECT * FROM kanban_tasks WHERE id = $taskId AND event_id = $eventID
if task == nil: return 404

isOwnTask := task.assigned_to != nil && task.assigned_to == ctx.userID
if !ctx.isAdminOrOwner && !isOwnTask: return 403

body := parse(new_status)   -- must be one of: not_started, in_progress, completed
if !isValidStatus(body.new_status): return 400

if body.new_status == task.status: return 200, task   -- no-op, skip logging

BEGIN TX
    UPDATE kanban_tasks
    SET status = body.new_status, updated_at = now()
    WHERE id = $taskId
    -- CHECK constraint enforces valid enum value at DB level too

    INSERT INTO kanban_task_status_log
        (task_id, changed_by, old_status, new_status)
        VALUES ($taskId, ctx.userID, task.status, body.new_status)
COMMIT

return 200, updatedTask
```

---

## 8. GET /events/:eventId/kanban/tasks/:taskId/status-history — Audit trail
**Auth:** admin/owner (any task) OR assignee (own task) — your call on broader team visibility

```
task := SELECT * FROM kanban_tasks WHERE id = $taskId AND event_id = $eventID
if task == nil: return 404

isOwnTask := task.assigned_to != nil && task.assigned_to == ctx.userID
if !ctx.isAdminOrOwner && !isOwnTask: return 403

history := SELECT * FROM kanban_task_status_log
           WHERE task_id = $taskId
           ORDER BY changed_at ASC

return 200, history
```

---

## Notes
- Every handler re-derives `isAdminOrOwner` from context (set once in middleware) — no repeated admin-check queries per handler.
- Status update and assign are deliberately separate from the generic PATCH — different auth rules, different side effects (log insert vs plain field update).
- `isEventParticipant()` is a helper you'll need to define — likely checks `volunteer_applications.status = 'approved'` for that event, or organizer_id match. Worth deciding once and reusing everywhere read-access is checked.
- Cascade delete on `kanban_task_status_log.task_id` means deleting a task wipes its history — if you want to preserve history even after task deletion (e.g. for compliance/audit), you'd need `ON DELETE SET NULL` or a soft-delete pattern on tasks instead. Worth considering if audit trail matters beyond the task's lifetime.
