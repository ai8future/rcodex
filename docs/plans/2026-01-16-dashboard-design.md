# rcodegen Dashboard Design

## Overview

A local web dashboard for viewing rcodegen reports across repos, with scheduling capabilities.

**Components:**
- **Dashboard** (Next.js) - Web UI at `rcodegen/dashboard`
- **Scheduler Daemon** (Node.js) - Standalone at `rcodegen/scheduler`

## Architecture

### Dashboard (Next.js 14 + App Router + Tailwind)

Scans `~/Desktop/_code` for repos with `_rcodegen` directories on each page load.

**Pages:**
- `/` - Repo list with grades, last run, report counts, schedule status
- `/repo/[name]` - Report list for a repo, grouped by type
- `/repo/[name]/report/[file]` - Rendered markdown report
- `/schedules` - Manage all schedules

**API Routes:**
- `GET /api/repos` - Scan filesystem, return repo summaries
- `GET /api/repos/[name]/reports` - List reports for a repo
- `GET /api/repos/[name]/reports/[file]` - Return report content
- `GET /api/schedules` - Read schedules.json
- `POST /api/schedules` - Add schedule
- `DELETE /api/schedules/[id]` - Remove schedule
- `PATCH /api/schedules/[id]` - Update schedule (toggle enabled, edit cron)
- `GET /api/daemon/status` - Read scheduler-status.json

### Scheduler Daemon (Node.js)

Standalone process that reads schedules and spawns rcodex commands.

**Behavior:**
- Polls `~/.rcodegen/schedules.json` every 60 seconds
- Uses node-cron or similar for cron parsing
- Spawns `rcodex <task>` in repo directory
- Writes heartbeat + recent runs to `~/.rcodegen/scheduler-status.json`

## Data Model

### `~/.rcodegen/schedules.json`

```json
{
  "schedules": [
    {
      "id": "abc123",
      "repo": "/Users/cliff/Desktop/_code/dispatch",
      "task": "suite",
      "cron": "0 9 * * 1",
      "enabled": true,
      "created": "2026-01-16T14:00:00Z"
    }
  ]
}
```

### `~/.rcodegen/scheduler-status.json`

```json
{
  "pid": 12345,
  "started": "2026-01-16T08:00:00Z",
  "lastHeartbeat": "2026-01-16T14:30:00Z",
  "recentRuns": [
    {
      "id": "abc123",
      "repo": "dispatch",
      "task": "suite",
      "startedAt": "2026-01-16T09:00:00Z",
      "finishedAt": "2026-01-16T09:12:00Z",
      "exitCode": 0
    }
  ]
}
```

## Report Parsing

**File naming convention:** `{tool}-{codebase}-{task}-{date}.md`

**Grade extraction:** Search for patterns like:
- `TOTAL_SCORE: XX/100`
- `Overall Grade: XX/100`
- `Grade: XX/100`

**Metadata extraction:**
- Date from filename or "Date Created:" line in report
- Tool from filename prefix (codex, gemini, etc.)
- Task type from filename (audit, test, fix, refactor, quick, grade)

## UI Components

### Repo Table (main page)
| Repo | Last Run | Grade | Reports | Scheduled |
|------|----------|-------|---------|-----------|
| dispatch | 2h ago | 77/100 | 14 | Mon 9am |

### Daemon Status Indicator
- Green dot + "Running" when heartbeat < 2 minutes old
- Red dot + "Stopped" when heartbeat stale or file missing

## Running Locally

```bash
# Dashboard
cd dashboard
npm install
npm run dev
# Open http://localhost:3000

# Scheduler (separate terminal)
cd scheduler
npm install
node index.js
```

## Future Considerations (not in v1)
- Grade trend charts over time
- Email/notification on schedule completion
- Compare reports side-by-side
- Filter/search reports
