# GPC - GORM Preload Checker

Static analysis tool that validates GORM `Preload()` relation names at development time instead of runtime. Uses `go/types` for type-checked analysis.

## What It Does

Scans Go source files for `db.Preload("...")` calls, verifies the receiver is `*gorm.DB` via type checking, resolves the model type from nearby `Find`/`First`/`FirstOrCreate` calls using `go/types`, then recursively verifies relation paths against actual struct field definitions including embedded structs and cross-package types.

Domain glossary (Chain, Relation Path, Model, Verification): see `CONTEXT.md`.

## Architecture

```
main.go                          CLI entry (cobra), flags, delegates to engine
internal/
  engine/engine.go               Orchestrator: loader → collector → relations → results
  loader/loader.go               go/packages.Load wrapper, returns typed package info
  collector/collector.go         Single AST walk: extracts Preload chains, pre-resolves source lines
  relations/                     Model resolution + relation-path verification
    relations.go                 Verify entry point + result mapping
    resolve.go                   Model extraction (pointer/slice/named unwrap), field lookup
    walk.go                      Dotted relation-path traversal with diagnostic walkResult
  models/types.go                Shared data types (PreloadResult, AnalysisResult)
  output/output.go               Console and JSON output formatters
  testutil/testutil.go           Test helper: creates temp Go modules for go/packages
```

## Pipeline Flow

1. **Package Loading** — `loader.Load`: `go/packages.Load` with full type info
2. **Chain Collection** — `collector.Collect`: single AST walk finds `.Preload().Find()` chains, verifies `*gorm.DB` receiver, resolves constants, pre-resolves each Preload's source line (no `token.Pos` leaks downstream)
3. **Verification** — `relations.Verify`: resolves each chain's model and walks every dotted relation path through `types.Struct`. Walk returns a `walkResult{ok, failedAt, parent}` so future diagnostics can name the failing segment and the type it was looked up in.
4. **Output** — console or JSON

## Build & Test

```bash
go build -o gpc .
go test ./internal/...          # all tests pass
```

## CLI Flags

- `-o console|json` output format
- `-f <file>` JSON output path (default: `gpc_results.json`)
- `-V` validation-only (skip unknowns)
- `-e` errors-only

## Capabilities

- Type-checked `*gorm.DB` receiver verification (ignores non-GORM `.Preload()`)
- Recursive nested relation validation (`User.Profile.Address` — validates every level)
- Cross-package type resolution (models in different packages)
- Embedded struct field lookup (promoted fields)
- Constant folding (`const RelUser = "User"` resolved at analysis time)
- `clause.Associations` support
- Variable-assigned chains (`query := db.Preload("User"); query.Find(&orders)`)
- Embedded `*gorm.DB` wrappers (e.g. `QueryBuilder{*gorm.DB}` — Find/Preload via promotion)
- Struct literal initialization (`&QueryBuilder{DB: db.Preload("X")}`)
- Dynamic argument detection (non-literal args marked as skipped)

## Conventions

- Go 1.25, module `github.com/your-moon/gpc`
- Uses `go/types` + `golang.org/x/tools/go/packages` for type-checked static analysis
- Table-driven tests with `testing` stdlib
- `testutil.CreateTestModule` creates temp Go modules for tests
