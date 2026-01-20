# Plan: Persistent Grade Storage

## Overview
Store grades independently of report files in `_rcodegen/.grades.json` so historical data survives report file deletion.

## Architecture

```
Report Generated → Extract Grade → Append to .grades.json
                                          ↓
                            Dashboard reads .grades.json
                            (falls back to file scan)
```

## File Format: `_rcodegen/.grades.json`

```json
{
  "grades": [
    {
      "date": "2026-01-16T23:36:00Z",
      "tool": "claude",
      "task": "audit",
      "grade": 82,
      "reportFile": "claude-rcodegen-audit-2026-01-16_2336.md"
    },
    {
      "date": "2026-01-16T23:27:00Z",
      "tool": "gemini",
      "task": "audit",
      "grade": 84,
      "reportFile": "gemini-rcodegen-audit-2026-01-16_2327.md"
    }
  ]
}
```

---

## Implementation Steps

### Part 1: CLI Grade Persistence (Go)

#### 1.1 Create `pkg/runner/grades.go`

New file with:
- `GradeEntry` struct
- `GradesFile` struct
- `ExtractGradeFromReport(reportPath string) (float64, error)` - regex extraction
- `AppendGrade(reportDir, reportFile, tool, task string, grade float64) error` - append to .grades.json
- `LoadGrades(reportDir string) (*GradesFile, error)` - read existing
- `SaveGrades(reportDir string, grades *GradesFile) error` - write with locking

#### 1.2 Modify `pkg/runner/runner.go`

After task execution completes successfully (~line 312):
```go
// After runSingleTask returns
if exitCode == 0 {
    r.persistGrade(cfg, workDir)
}
```

Add `persistGrade` method:
1. Get report directory
2. Find the newest `.md` file matching `{tool}-*-{task}-*.md`
3. Extract grade using regex
4. Call `AppendGrade()`

#### 1.3 Grade Extraction Regex

Use same patterns as dashboard:
```go
patterns := []string{
    `TOTAL_SCORE:\s*(\d+(?:\.\d+)?)\s*/\s*100`,
    `Overall Grade[^:]*:\s*(\d+(?:\.\d+)?)\s*/\s*100`,
    `Grade[^:]*:\s*(\d+(?:\.\d+)?)\s*/\s*100`,
    `(\d+(?:\.\d+)?)\s*/\s*100\s*points`,
}
```

---

### Part 2: Migration Command

#### 2.1 Add `--migrate-grades` flag to rcodegen

```bash
rcodegen --migrate-grades [directory]
```

Behavior:
1. Scan `_rcodegen/*.md` files in specified directory (or current)
2. Parse filename: `{tool}-{codebase}-{task}-{YYYY-MM-DD_HHMM}.md`
3. Extract grade from each file
4. Build `.grades.json` (deduplicate by reportFile)
5. Write to `_rcodegen/.grades.json`

#### 2.2 Implementation in `pkg/runner/migrate.go`

```go
func MigrateGrades(baseDir string) error {
    reportDir := filepath.Join(baseDir, "_rcodegen")
    files, _ := filepath.Glob(filepath.Join(reportDir, "*.md"))

    grades := &GradesFile{Grades: []GradeEntry{}}

    for _, file := range files {
        tool, task, date := parseFilename(filepath.Base(file))
        grade := ExtractGradeFromReport(file)
        if grade > 0 {
            grades.Grades = append(grades.Grades, GradeEntry{
                Date:       date,
                Tool:       tool,
                Task:       task,
                Grade:      grade,
                ReportFile: filepath.Base(file),
            })
        }
    }

    return SaveGrades(reportDir, grades)
}
```

---

### Part 3: Dashboard Updates

#### 3.1 Modify `src/app/api/repos/route.ts`

Update grade history building:

```typescript
async function loadGradeHistory(reportDir: string): Promise<GradeHistoryPoint[]> {
  const gradesFile = path.join(reportDir, '.grades.json')

  // Try to read from .grades.json first
  try {
    const data = await fs.readFile(gradesFile, 'utf-8')
    const { grades } = JSON.parse(data)
    return grades.map(g => ({
      date: g.date,
      grade: g.grade,
      tool: g.tool,
      task: g.task
    }))
  } catch {
    // Fall back to file scanning (backwards compatibility)
    return scanReportFiles(reportDir)
  }
}
```

#### 3.2 Hybrid Approach

For robustness, merge both sources:
1. Load from `.grades.json`
2. Scan for any new `.md` files not in `.grades.json`
3. Return combined, deduplicated list

---

## File Changes Summary

### New Files (Go)
- `pkg/runner/grades.go` - Grade persistence functions
- `pkg/runner/migrate.go` - Migration command

### Modified Files (Go)
- `pkg/runner/runner.go` - Hook grade persistence after task
- `pkg/runner/flags.go` - Add `--migrate-grades` flag
- `cmd/rcodegen/main.go` - Handle migrate flag

### Modified Files (TypeScript)
- `dashboard/src/app/api/repos/route.ts` - Read from .grades.json

---

## Execution Order

1. **CLI: grades.go** - Core grade persistence functions
2. **CLI: runner.go** - Hook after task completion
3. **CLI: migrate.go** - Migration command
4. **CLI: flags.go** - Add flag
5. **Run migration** - Backfill existing grades
6. **Dashboard: route.ts** - Read from .grades.json

---

## Testing

1. Run `rcodegen claude audit` on a repo
2. Check `_rcodegen/.grades.json` was created/updated
3. Delete a `.md` report file
4. Verify dashboard still shows the grade from `.grades.json`
5. Run migration on repo with existing reports
6. Verify all grades captured
