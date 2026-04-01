# ⏰ Scheduler

Run browser tasks on a schedule using cron expressions. Requires `scheduler.enabled: true`.

## Enable

```yaml
scheduler:
  enabled: true
```

## CRUD

```bash
# Create
curl -X POST http://localhost:3000/api/schedules \
  -H "Content-Type: application/json" \
  -d '{"cron":"0 * * * *","plugin":"screenshot","method":"capture","params":{"url":"https://dashboard.example.com"}}'

# List
curl http://localhost:3000/api/schedules

# Delete
curl -X DELETE http://localhost:3000/api/schedules/{id}
```

## Cron Syntax

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, Sun=0)
│ │ │ │ │
* * * * *
```

Examples:
- `*/5 * * * *` — every 5 minutes
- `0 * * * *` — every hour
- `0 9 * * 1-5` — weekdays at 9am
- `0 0 * * *` — daily at midnight

## Use with AI Agent

Schedule an AI agent to run periodically:

```bash
curl -X POST http://localhost:3000/api/schedules \
  -H "Content-Type: application/json" \
  -d '{
    "cron": "0 9 * * *",
    "plugin": "ai",
    "method": "execute",
    "params": {
      "task": "Check competitor pricing on example.com",
      "openai_api_key": "sk-...",
      "max_steps": 10
    }
  }'
```

## Persistence

With `storage.driver: sqlite`, scheduled tasks persist across restarts.
