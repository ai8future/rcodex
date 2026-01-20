# rcodegen Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix validated bugs and security issues identified across multiple audit reports.

**Architecture:** Direct fixes to existing code - no new abstractions needed. Each fix is isolated and testable independently.

**Tech Stack:** Go, TypeScript (Next.js dashboard)

---

## Consolidated Analysis

From 8 reports analyzed, I identified these **valid and actionable** fixes (excluding test suggestions):

### HIGH PRIORITY
1. **Path Traversal in Dashboard APIs** - All reports agree this is critical
2. **Orchestrator Codex/Gemini Label Bug** - Multiple reports confirm line 481 mislabels Codex as Gemini
3. **VoteExecutor Race Condition** - Accesses `ctx.StepResults` without lock

### MEDIUM PRIORITY
4. **GenerateJobID rand.Read Error Ignored** - Silent failure if entropy unavailable
5. **MergeExecutor Silent Failures** - File read errors are swallowed
6. **Deprecated strings.Title Usage** - 5 occurrences across 2 files
7. **Condition Evaluation Operator Precedence** - AND evaluated before OR (wrong)

### NOT FIXING (Assessed as Low Value or Risk)
- Lock file race condition in reading holder info (cosmetic only, not security)
- Workspace cleanup mechanism (feature, not bug)
- File IO under RLock in Context.Resolve (documented, safe for correctness)

---

### Task 1: Fix Path Traversal in Dashboard Report API

**Files:**
- Modify: `dashboard/src/app/api/repos/[name]/reports/[file]/route.ts`

**Step 1: Add path safety helper and use it**

```typescript
import { NextResponse } from 'next/server'
import fs from 'fs'
import path from 'path'
import os from 'os'

const CODE_DIR = path.resolve(os.homedir(), 'Desktop/_code')

function resolveWithin(baseDir: string, ...segments: string[]): string | null {
  const resolvedBase = path.resolve(baseDir)
  const resolvedTarget = path.resolve(resolvedBase, ...segments)
  // Ensure target is within base (prefix check with separator)
  const prefix = resolvedBase.endsWith(path.sep) ? resolvedBase : resolvedBase + path.sep
  if (!resolvedTarget.startsWith(prefix) && resolvedTarget !== resolvedBase) {
    return null
  }
  return resolvedTarget
}

export async function GET(
  request: Request,
  { params }: { params: Promise<{ name: string; file: string }> }
) {
  const { name, file } = await params

  // Validate path components
  const repoPath = resolveWithin(CODE_DIR, name)
  if (!repoPath) {
    return NextResponse.json({ error: 'Invalid repo path' }, { status: 400 })
  }

  const filePath = resolveWithin(repoPath, '_rcodegen', file)
  if (!filePath) {
    return NextResponse.json({ error: 'Invalid file path' }, { status: 400 })
  }

  // Require .md extension
  if (!filePath.endsWith('.md')) {
    return NextResponse.json({ error: 'Invalid file type' }, { status: 400 })
  }

  if (!fs.existsSync(filePath)) {
    return NextResponse.json({ error: 'Report not found' }, { status: 404 })
  }

  try {
    const content = fs.readFileSync(filePath, 'utf-8')
    return NextResponse.json({ content })
  } catch (error) {
    console.error('Error reading report:', error)
    return NextResponse.json({ error: 'Failed to read report' }, { status: 500 })
  }
}
```

**Step 2: Commit**

```bash
git add dashboard/src/app/api/repos/[name]/reports/[file]/route.ts
git commit -m "fix(dashboard): prevent path traversal in report API"
```

---

### Task 2: Fix Path Traversal in Dashboard Repo API

**Files:**
- Modify: `dashboard/src/app/api/repos/[name]/route.ts`

**Step 1: Add path safety helper and validate inputs**

Add `resolveWithin` helper (same as Task 1) and update the GET handler:

```typescript
const CODE_DIR = path.resolve(os.homedir(), 'Desktop/_code')

function resolveWithin(baseDir: string, ...segments: string[]): string | null {
  const resolvedBase = path.resolve(baseDir)
  const resolvedTarget = path.resolve(resolvedBase, ...segments)
  const prefix = resolvedBase.endsWith(path.sep) ? resolvedBase : resolvedBase + path.sep
  if (!resolvedTarget.startsWith(prefix) && resolvedTarget !== resolvedBase) {
    return null
  }
  return resolvedTarget
}
```

Then in the GET function, change:
```typescript
const { name } = await params
const repoPath = path.join(CODE_DIR, name)
```

To:
```typescript
const { name } = await params
const repoPath = resolveWithin(CODE_DIR, name)
if (!repoPath) {
  return NextResponse.json({ error: 'Invalid repo path' }, { status: 400 })
}
```

And update `rcodgenDir`:
```typescript
const rcodgenDir = path.join(repoPath, '_rcodegen')
```

**Step 2: Commit**

```bash
git add dashboard/src/app/api/repos/[name]/route.ts
git commit -m "fix(dashboard): prevent path traversal in repo API"
```

---

### Task 3: Fix Codex/Gemini Label Bug in Orchestrator

**Files:**
- Modify: `pkg/orchestrator/orchestrator.go:481`

**Step 1: Fix the label from "Gemini" to "Codex"**

Change line 481 from:
```go
expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Gemini", "✓", codexEditCost, codexOut})
```

To:
```go
expanded = append(expanded, ExpandedStep{stepNum, "Edit", "Codex", "✓", codexEditCost, codexOut})
```

**Step 2: Run build to verify**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add pkg/orchestrator/orchestrator.go
git commit -m "fix(orchestrator): correct Codex label in run report"
```

---

### Task 4: Fix VoteExecutor Race Condition

**Files:**
- Modify: `pkg/orchestrator/context.go` (add GetResult method)
- Modify: `pkg/executor/vote.go` (use GetResult instead of direct access)

**Step 1: Add GetResult method to Context**

Add after line 110 in `context.go`:
```go
// GetResult safely retrieves a step result with proper locking.
func (c *Context) GetResult(name string) (*envelope.Envelope, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	env, ok := c.StepResults[name]
	return env, ok
}
```

**Step 2: Update VoteExecutor to use GetResult**

In `vote.go`, change line 20 from:
```go
if env, ok := ctx.StepResults[stepName]; ok {
```

To:
```go
if env, ok := ctx.GetResult(stepName); ok && env != nil {
```

**Step 3: Run build to verify**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add pkg/orchestrator/context.go pkg/executor/vote.go
git commit -m "fix(executor): add thread-safe GetResult for vote executor"
```

---

### Task 5: Fix GenerateJobID Error Handling

**Files:**
- Modify: `pkg/workspace/workspace.go:24-26`

**Step 1: Handle rand.Read error**

Change:
```go
func GenerateJobID() string {
	now := time.Now()
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s-%s", now.Format("20060102-150405"), hex.EncodeToString(b))
}
```

To:
```go
func GenerateJobID() string {
	now := time.Now()
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use nanoseconds if crypto/rand fails
		return fmt.Sprintf("%s-%08x", now.Format("20060102-150405"), now.UnixNano()&0xFFFFFFFF)
	}
	return fmt.Sprintf("%s-%s", now.Format("20060102-150405"), hex.EncodeToString(b))
}
```

**Step 2: Run build and tests**

```bash
go build ./... && go test ./pkg/workspace/...
```

**Step 3: Commit**

```bash
git add pkg/workspace/workspace.go
git commit -m "fix(workspace): handle rand.Read failure in GenerateJobID"
```

---

### Task 6: Fix MergeExecutor Silent Failures

**Files:**
- Modify: `pkg/executor/merge.go`

**Step 1: Add import for fmt**

Add `"fmt"` to imports.

**Step 2: Track and report failed inputs**

Change:
```go
func (e *MergeExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context, ws *workspace.Workspace) (*envelope.Envelope, error) {
	// Collect inputs
	var contents []string
	for _, inputRef := range step.Merge.Inputs {
		path := ctx.Resolve(inputRef)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		contents = append(contents, string(data))
	}
```

To:
```go
func (e *MergeExecutor) Execute(step *bundle.Step, ctx *orchestrator.Context, ws *workspace.Workspace) (*envelope.Envelope, error) {
	// Collect inputs
	var contents []string
	var failedInputs []string
	for _, inputRef := range step.Merge.Inputs {
		path := ctx.Resolve(inputRef)
		data, err := os.ReadFile(path)
		if err != nil {
			failedInputs = append(failedInputs, fmt.Sprintf("%s: %v", inputRef, err))
			continue
		}
		contents = append(contents, string(data))
	}
```

And update the return envelope to include failed_inputs:
```go
return envelope.New().
	Success().
	WithOutputRef(outputPath).
	WithResult("input_count", len(contents)).
	WithResult("failed_inputs", failedInputs).
	Build(), nil
```

**Step 3: Run build**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add pkg/executor/merge.go
git commit -m "fix(executor): track failed inputs in merge executor"
```

---

### Task 7: Replace Deprecated strings.Title

**Files:**
- Modify: `pkg/orchestrator/progress.go` (3 occurrences)
- Modify: `pkg/orchestrator/live_display.go` (2 occurrences)

**Step 1: Add capitalize helper to progress.go**

Add after the imports:
```go
// capitalize returns s with first letter uppercased (replaces deprecated strings.Title)
func capitalizeWord(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
```

**Step 2: Replace strings.Title in progress.go**

Replace all 3 occurrences of `strings.Title(...)` with `capitalizeWord(...)`:
- Line 265: `toolName := capitalizeWord(step.Tool)`
- Line 327: `toolClr, capitalizeWord(step.Tool), colorReset,`
- Line 431: `toolClr, capitalizeWord(step.Tool), colorReset)`

**Step 3: Add capitalize helper to live_display.go**

Add the same helper function after imports.

**Step 4: Replace strings.Title in live_display.go**

Replace both occurrences:
- Line 385: `toolName := capitalizeWord(step.Tool)`
- Line 390: `modelName := capitalizeWord(step.Model)`

**Step 5: Run build**

```bash
go build ./...
```

**Step 6: Commit**

```bash
git add pkg/orchestrator/progress.go pkg/orchestrator/live_display.go
git commit -m "fix(orchestrator): replace deprecated strings.Title"
```

---

### Task 8: Fix Condition Evaluation Operator Precedence

**Files:**
- Modify: `pkg/orchestrator/condition.go:21-26`

**Step 1: Swap AND/OR evaluation order**

Change:
```go
// Handle AND/OR
if idx := strings.Index(expr, " AND "); idx != -1 {
	return evaluate(expr[:idx]) && evaluate(expr[idx+5:])
}
if idx := strings.Index(expr, " OR "); idx != -1 {
	return evaluate(expr[:idx]) || evaluate(expr[idx+4:])
}
```

To:
```go
// Handle OR first (lower precedence - evaluated at top level)
if idx := strings.Index(expr, " OR "); idx != -1 {
	return evaluate(expr[:idx]) || evaluate(expr[idx+4:])
}
// Handle AND (higher precedence - evaluated deeper in recursion)
if idx := strings.Index(expr, " AND "); idx != -1 {
	return evaluate(expr[:idx]) && evaluate(expr[idx+5:])
}
```

**Step 2: Run build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add pkg/orchestrator/condition.go
git commit -m "fix(orchestrator): correct AND/OR operator precedence"
```

---

## Verification

After all tasks:

```bash
# Full build
go build ./...

# Run all tests
go test ./...

# Smoke test
./rcodegen -h
```

---

## Summary

| Task | Priority | Risk | Files Changed |
|------|----------|------|---------------|
| 1. Dashboard report path traversal | HIGH | LOW | 1 |
| 2. Dashboard repo path traversal | HIGH | LOW | 1 |
| 3. Codex/Gemini label bug | HIGH | LOW | 1 |
| 4. VoteExecutor race condition | HIGH | MEDIUM | 2 |
| 5. GenerateJobID error handling | MEDIUM | LOW | 1 |
| 6. MergeExecutor silent failures | MEDIUM | LOW | 1 |
| 7. Deprecated strings.Title | MEDIUM | LOW | 2 |
| 8. Condition operator precedence | MEDIUM | MEDIUM | 1 |
