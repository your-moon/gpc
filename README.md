# gpc

[![Go Report Card](https://goreportcard.com/badge/github.com/your-moon/gpc)](https://goreportcard.com/report/github.com/your-moon/gpc)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Static analysis tool that validates GORM `Preload()` relation names using Go's type checker.

Catches typos, missing fields, and invalid nested paths at analysis time instead of runtime.

```
$ gpc ./...
repo/order.go:79: OrderItems.Product.Categor not found in db.Order
repo/user.go:45: Profil not found in db.User

2 error(s)
```

## How it works

GPC loads your packages with full type information via `go/packages`, then:

1. Finds every `.Preload()` call on a `*gorm.DB` receiver (verified via type checker)
2. Resolves the model type from the terminal call (`Find(&orders)` → `[]Order` → `Order` struct)
3. Walks each dotted relation path (`"User.Profile.Address"`) against actual struct fields
4. Reports any field that doesn't exist at any nesting level

No string heuristics. No guessing. The type checker knows exactly what fields exist.

## Install

```
go install github.com/your-moon/gpc@latest
```

## Usage

```
gpc ./...                      # check all packages
gpc ./internal/repo/           # check a directory
gpc ./internal/repo/order.go   # check a single file
```

### Flags

```
-o text|json    Output format (default: text)
-f <path>       Write JSON output to file (implies -o json)
-e              Show only errors
-V              Show only validated results (valid + errors, hide skipped)
```

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | All preloads valid |
| 1 | Tool error (bad arguments, package load failure) |
| 2 | Invalid preloads found |

### CI integration

```yaml
# GitHub Actions
- name: Check GORM preloads
  run: |
    go install github.com/your-moon/gpc@latest
    gpc ./...
```

```bash
# Pre-commit hook
#!/bin/sh
gpc ./... || exit 1
```

## What it catches

```go
type Profile struct { Bio string }
type User struct { Profile Profile }
type Order struct { User User }

func GetOrders(db *gorm.DB) {
    var orders []Order

    db.Preload("User").Find(&orders)                  // valid
    db.Preload("User.Profile").Find(&orders)           // valid
    db.Preload("User.Profil").Find(&orders)            // error: Profil not found in User
    db.Preload("Customer").Find(&orders)               // error: Customer not found in Order
    db.Preload("User.Profile.Address").Find(&orders)   // error: Address not found in Profile
}
```

### Supported patterns

| Pattern | Example | Supported |
|---------|---------|-----------|
| Direct chain | `db.Preload("User").Find(&x)` | Yes |
| Multiple preloads | `db.Preload("A").Preload("B").Find(&x)` | Yes |
| Nested relations | `db.Preload("User.Profile.Address")` | Yes |
| Cross-package models | `db.Preload("User").Find(&models.Order{})` | Yes |
| Embedded structs | `Preload("Creator")` on struct embedding `BaseModel` | Yes |
| Constants | `const Rel = "User"; db.Preload(Rel)` | Yes |
| `clause.Associations` | `db.Preload(clause.Associations)` | Yes |
| Variable-assigned db | `q := db.Preload("User"); q.Find(&x)` | Yes |
| Wrapper types | `type QB struct { *gorm.DB }; qb.Find(&x)` | Yes |
| Struct literal init | `&QB{DB: db.Preload("User")}` | Yes |
| Dynamic arguments | `db.Preload(someVar)` | Skipped (reported) |
| Preload conditions | `db.Preload("Posts", "active = ?", true)` | Yes (first arg validated) |

### What it skips

- Dynamic (non-constant) relation names — reported as "skipped"
- `Preload()` calls on types that are not `*gorm.DB` (or don't embed it)
- Preload chains with no terminal call (`Find`, `First`, `Take`, `Last`, `Scan`, `FirstOrCreate`)

## JSON output

```
gpc -f results.json ./...
```

```json
{
  "total": 5,
  "valid": 3,
  "errors": 2,
  "skipped": 0,
  "results": [
    {
      "file": "repo/order.go",
      "line": 79,
      "relation": "User",
      "model": "db.Order",
      "status": "valid"
    },
    {
      "file": "repo/order.go",
      "line": 82,
      "relation": "Profil",
      "model": "db.Order",
      "status": "error"
    }
  ]
}
```

## Architecture

```
main.go               CLI (cobra)
internal/
  engine/              Pipeline orchestrator
  loader/              go/packages.Load with full type info
  collector/           AST walk → Preload chain extraction
  resolver/            Type-based model resolution from Find() args
  validator/           Recursive relation path validation
  models/              Shared types
  output/              Text and JSON formatters
```

## Development

```
go build -o gpc .
go test ./internal/...
go vet ./...
```

## License

MIT
