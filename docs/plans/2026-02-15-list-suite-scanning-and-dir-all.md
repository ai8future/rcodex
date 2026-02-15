# --list _suite Fallback & --dir-all Flag Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `--list` find repos inside `*_suite/` subdirectories, and add `--dir-all` to run all git repos in a directory.

**Architecture:** Extend the existing `--list` resolution in `parseArgs` with a fallback scan of `*_suite/` folders. Add a new `--dir-all` flag that reuses `discoverDirectories` to expand directories into their git-repo children.

**Tech Stack:** Go, standard library only (`os`, `path/filepath`, `strings`)

---

### Task 1: Add DirAll field to Config

**Files:**
- Modify: `pkg/runner/config.go:52` (add field after DirList)

**Step 1: Add the field**

In `pkg/runner/config.go`, add `DirAll` right after the `DirList` field (line 52):

```go
	DirList       string // Comma-separated subdirectory names (--list)
	DirAll        string // Comma-separated directories: run all git repos within (--dir-all)
```

**Step 2: Compile to verify**

Run: `cd /Users/cliff/Desktop/_code/codegen_suite/rcodegen && go build ./...`
Expected: Clean build, no errors

**Step 3: Commit**

```bash
git add pkg/runner/config.go
git commit -m "feat: add DirAll config field for --dir-all flag"
```

---

### Task 2: Add --dir-all flag definition and help text

**Files:**
- Modify: `pkg/runner/runner.go:858` (flag definition)
- Modify: `pkg/runner/runner.go:1178-1182` (help text)

**Step 1: Add flag definition**

After line 858 (`flag.StringVar(&cfg.DirList, "list", ...)`), add:

```go
	flag.StringVar(&cfg.DirAll, "A", "", "Run all git repos in directory (comma-separated paths)")
	flag.StringVar(&cfg.DirAll, "dir-all", "", "Run all git repos in directory (comma-separated paths)")
```

**Step 2: Add help text**

In `printUsage()`, after the `--list` help lines (after line 1179), add:

```go
	fmt.Printf("  %s-A%s, %s--dir-all%s %s<path>%s  Run all git repos in directory\n", Green, Reset, Green, Reset, Yellow, Reset)
	fmt.Printf("                        %s(comma-separated for multiple: --dir-all /a,/b)%s\n", Dim, Reset)
```

**Step 3: Compile to verify**

Run: `cd /Users/cliff/Desktop/_code/codegen_suite/rcodegen && go build ./...`
Expected: Clean build

**Step 4: Commit**

```bash
git add pkg/runner/runner.go
git commit -m "feat: add --dir-all / -A flag definition and help text"
```

---

### Task 3: Add --dir-all handling logic

**Files:**
- Modify: `pkg/runner/runner.go:928-931` (mutual exclusivity)
- Modify: `pkg/runner/runner.go:969` (add new block after --list handling)

**Step 1: Update mutual exclusivity checks**

Replace the existing check at lines 928-931:

```go
	// Check mutual exclusivity of --list and --recursive
	if cfg.DirList != "" && cfg.Recursive {
		return nil, fmt.Errorf("--list and --recursive cannot be used together")
	}
```

With:

```go
	// Check mutual exclusivity of directory scanning modes
	if cfg.DirList != "" && cfg.Recursive {
		return nil, fmt.Errorf("--list and --recursive cannot be used together")
	}
	if cfg.DirAll != "" && cfg.DirList != "" {
		return nil, fmt.Errorf("--dir-all and --list cannot be used together")
	}
	if cfg.DirAll != "" && cfg.Recursive {
		return nil, fmt.Errorf("--dir-all and --recursive cannot be used together")
	}
```

**Step 2: Add --dir-all handling block**

After the closing `}` of the `--list` block (line 969) and before the `// Handle recursive directory scanning` comment, add:

```go
	// Handle --dir-all: run all git repos in specified directories
	if cfg.DirAll != "" {
		var allDirs []string
		paths := strings.Split(cfg.DirAll, ",")
		for _, p := range paths {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			found, err := discoverDirectories(p, 1)
			if err != nil {
				return nil, fmt.Errorf("--dir-all: %v", err)
			}
			allDirs = append(allDirs, found...)
		}
		if len(allDirs) == 0 {
			return nil, fmt.Errorf("--dir-all: no git repositories found")
		}
		cfg.WorkDirs = allDirs
		cfg.Codebase = filepath.Base(allDirs[0])
	}
```

**Step 3: Compile to verify**

Run: `cd /Users/cliff/Desktop/_code/codegen_suite/rcodegen && go build ./...`
Expected: Clean build

**Step 4: Commit**

```bash
git add pkg/runner/runner.go
git commit -m "feat: implement --dir-all directory expansion logic"
```

---

### Task 4: Add _suite fallback to --list resolution

**Files:**
- Modify: `pkg/runner/runner.go:950-963` (the --list name resolution loop)

**Step 1: Extract a helper function**

Add this function before `discoverDirectories` (around line 638):

```go
// findSuiteDirs returns all *_suite subdirectory paths under baseDir.
// Results are cached: pass a non-nil pointer to reuse across calls.
func findSuiteDirs(baseDir string, cached *[]string) []string {
	if cached != nil && *cached != nil {
		return *cached
	}
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil
	}
	var suiteDirs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), "_suite") {
			suiteDirs = append(suiteDirs, filepath.Join(baseDir, entry.Name()))
		}
	}
	if cached != nil {
		*cached = suiteDirs
	}
	return suiteDirs
}
```

**Step 2: Update the --list resolution loop**

Replace the loop body at lines 950-963:

```go
		names := strings.Split(cfg.DirList, ",")
		var filtered []string
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			dirPath := filepath.Join(baseDir, name)
			if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
				filtered = append(filtered, dirPath)
			} else {
				return nil, fmt.Errorf("directory not found: %s", dirPath)
			}
		}
```

With:

```go
		names := strings.Split(cfg.DirList, ",")
		var filtered []string
		var suiteDirs []string // lazily populated cache
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			dirPath := filepath.Join(baseDir, name)
			if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
				// Found directly in base directory
				filtered = append(filtered, dirPath)
			} else {
				// Fallback: scan *_suite subdirectories
				found := false
				for _, suiteDir := range findSuiteDirs(baseDir, &suiteDirs) {
					candidate := filepath.Join(suiteDir, name)
					if info, err := os.Stat(candidate); err == nil && info.IsDir() {
						filtered = append(filtered, candidate)
						found = true
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("directory not found: %s (also checked *_suite subdirectories)", dirPath)
				}
			}
		}
```

**Step 3: Compile to verify**

Run: `cd /Users/cliff/Desktop/_code/codegen_suite/rcodegen && go build ./...`
Expected: Clean build

**Step 4: Commit**

```bash
git add pkg/runner/runner.go
git commit -m "feat: add _suite fallback scanning to --list resolution"
```

---

### Task 5: Add tests for _suite fallback and --dir-all

**Files:**
- Modify: `pkg/runner/runner_test.go`

**Step 1: Write tests for findSuiteDirs**

Add to `pkg/runner/runner_test.go`:

```go
func TestFindSuiteDirs(t *testing.T) {
	// Create temp directory structure
	base := t.TempDir()
	os.MkdirAll(filepath.Join(base, "serp_suite", "serp_svc", ".git"), 0755)
	os.MkdirAll(filepath.Join(base, "ai_suite", "infra_ai8", ".git"), 0755)
	os.MkdirAll(filepath.Join(base, "regular_dir"), 0755)
	os.MkdirAll(filepath.Join(base, "solstice", ".git"), 0755)

	dirs := findSuiteDirs(base, nil)
	if len(dirs) != 2 {
		t.Fatalf("expected 2 suite dirs, got %d: %v", len(dirs), dirs)
	}

	// Verify caching
	var cached []string
	_ = findSuiteDirs(base, &cached)
	if cached == nil {
		t.Fatal("expected cache to be populated")
	}
	dirs2 := findSuiteDirs(base, &cached)
	if len(dirs2) != 2 {
		t.Fatalf("cached call: expected 2 suite dirs, got %d", len(dirs2))
	}
}

func TestFindSuiteDirs_Empty(t *testing.T) {
	base := t.TempDir()
	dirs := findSuiteDirs(base, nil)
	if len(dirs) != 0 {
		t.Fatalf("expected 0 suite dirs, got %d", len(dirs))
	}
}
```

**Step 2: Write tests for discoverDirectories (used by --dir-all)**

```go
func TestDiscoverDirectories_SingleLevel(t *testing.T) {
	base := t.TempDir()
	os.MkdirAll(filepath.Join(base, "repo1", ".git"), 0755)
	os.MkdirAll(filepath.Join(base, "repo2", ".git"), 0755)
	os.MkdirAll(filepath.Join(base, "not_a_repo"), 0755)

	dirs, err := discoverDirectories(base, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dirs) != 2 {
		t.Fatalf("expected 2 repos, got %d: %v", len(dirs), dirs)
	}
}
```

**Step 3: Run tests**

Run: `cd /Users/cliff/Desktop/_code/codegen_suite/rcodegen && go test ./pkg/runner/ -v -run "TestFindSuiteDirs|TestDiscoverDirectories_SingleLevel"`
Expected: All PASS

**Step 4: Commit**

```bash
git add pkg/runner/runner_test.go
git commit -m "test: add tests for _suite fallback and directory discovery"
```

---

### Task 6: Build binaries and final verification

**Files:**
- Build: all binaries

**Step 1: Run full test suite**

Run: `cd /Users/cliff/Desktop/_code/codegen_suite/rcodegen && go test ./...`
Expected: All PASS

**Step 2: Build all binaries**

Run: `cd /Users/cliff/Desktop/_code/codegen_suite/rcodegen && go build -o bin/rclaude ./cmd/rclaude && go build -o bin/rcodex ./cmd/rcodex && go build -o bin/rgemini ./cmd/rgemini && go build -o bin/rcodegen ./cmd/rcodegen`
Expected: All binaries built successfully

**Step 3: Quick smoke test**

Run: `./bin/rclaude --help` and verify `--dir-all` appears in output.
Run: `./bin/rclaude -d ~/Desktop/_code --list serp_svc -n suite` to verify _suite fallback resolves `serp_svc` via `serp_suite/serp_svc`.

**Step 4: Increment VERSION, update CHANGELOG, commit and push**
