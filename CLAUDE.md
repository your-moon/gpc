# GPC - GORM Preload Checker

Static analysis tool that validates GORM `Preload()` relation names at development time instead of runtime. Uses `go/types` for type-checked analysis.

## What It Does

Scans Go source files for `db.Preload("...")` calls, verifies the receiver is `*gorm.DB` via type checking, resolves the model type from nearby `Find`/`First`/`FirstOrCreate` calls using `go/types`, then recursively validates relation paths against actual struct field definitions including embedded structs and cross-package types.

## Architecture (v2)

```
main.go                          CLI entry (cobra), flags, delegates to v2 engine
internal/v2/
  engine/engine.go               Orchestrator: loader → collector → validator → results
  loader/loader.go               go/packages.Load wrapper, returns typed package info
  collector/collector.go         Single AST walk: extracts Preload chains with type verification
  resolver/resolver.go           Type-based model resolution from Find/First args
  validator/validator.go         Recursive relation path validation via types.Struct
  testutil/testutil.go           Test helper: creates temp Go modules for go/packages
internal/
  models/types.go                Shared data types (PreloadResult, AnalysisResult)
  output/output.go               Console and JSON output formatters
```

## Pipeline Flow

1. **Package Loading** — `loader.Load`: `go/packages.Load` with full type info
2. **Chain Collection** — `collector.Collect`: single AST walk finds `.Preload().Find()` chains, verifies `*gorm.DB` receiver, resolves constants
3. **Model Resolution** — `resolver.Resolve`: unwraps `&variable` type (pointer/slice/named) to find concrete struct
4. **Validation** — `validator.Validate`: recursively walks dotted relation paths against struct fields via `types.Struct`
5. **Output** — console or JSON

## Build & Test

```bash
go build -o gpc .
go test ./internal/v2/...       # v2 tests (all pass)
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
- Dynamic argument detection (non-literal args marked as skipped)

## Conventions

- Go 1.25, module `github.com/your-moon/gpc`
- Uses `go/types` + `golang.org/x/tools/go/packages` for type-checked static analysis
- Table-driven tests with `testing` stdlib
- `testutil.CreateTestModule` creates temp Go modules for tests
- v1 code still exists in `internal/` (parser, analyzer, validator, service, debug) but is unused by main.go
