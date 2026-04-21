# GPC v2: Type-Checked GORM Preload Analysis

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace GPC's fragile string-heuristic analysis with type-checked static analysis using `go/types` + `golang.org/x/tools/go/packages`, enabling correct cross-package type resolution, recursive nested relation validation, and `*gorm.DB` receiver verification.

**Architecture:** v2 lives in `internal/v2/` alongside v1. New pipeline: `loader` (go/packages.Load) → `collector` (single AST walk finds Preload chains + terminal Find calls) → `resolver` (go/types resolves model from Find argument) → `validator` (recursive struct field walk for nested relations) → existing `output` package reused. `main.go` swapped to v2 at end.

**Tech Stack:** Go 1.25, `golang.org/x/tools/go/packages`, `go/types`, `go/ast`, cobra CLI (kept from v1)

---

## File Structure

```
internal/v2/
  testutil/          testutil.go        — CreateTestModule helper for go/packages-compatible temp dirs
  loader/            loader.go          — go/packages.Load wrapper, returns typed package info
  collector/         collector.go       — single AST walk: extracts PreloadChain structs
  resolver/          resolver.go        — type-based model resolution from Find/First args
  validator/         validator.go       — recursive relation path validation via types.Struct
  engine/            engine.go          — orchestrator: loader → collector → resolver → validator → results
main.go                                — swapped to use internal/v2/engine at end
internal/output/output.go              — reused as-is (already works with models.PreloadResult)
internal/models/types.go               — reused as-is
```

---

### Task 1: Add `golang.org/x/tools` dependency and v2 test helper

**Files:**
- Modify: `go.mod`
- Create: `internal/v2/testutil/testutil.go`
- Create: `internal/v2/testutil/testutil_test.go`

- [ ] **Step 1: Add golang.org/x/tools dependency**

Run:
```bash
go get golang.org/x/tools/go/packages
```

- [ ] **Step 2: Write the test for CreateTestModule**

```go
// internal/v2/testutil/testutil_test.go
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestCreateTestModule(t *testing.T) {
	files := map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Find(&orders)
}
`,
	}

	dir := CreateTestModule(t, files)

	// Verify go.mod exists
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); os.IsNotExist(err) {
		t.Fatal("go.mod not created")
	}

	// Verify go/packages can load it
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedName,
		Dir:  dir,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		t.Fatalf("packages.Load failed: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("no packages loaded")
	}
	if len(pkgs[0].Errors) > 0 {
		t.Fatalf("package errors: %v", pkgs[0].Errors)
	}
	if pkgs[0].TypesInfo == nil {
		t.Fatal("TypesInfo is nil — types not loaded")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/v2/testutil/ -run TestCreateTestModule -v`
Expected: FAIL — package/function not found

- [ ] **Step 4: Implement CreateTestModule**

```go
// internal/v2/testutil/testutil.go
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// CreateTestModule creates a temporary Go module with the given files.
// Returns the module directory path. Cleaned up automatically when test ends.
// Each key in files is a relative path, value is file content.
func CreateTestModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	goMod := `module testmod

go 1.25

require gorm.io/gorm v1.31.0

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/text v0.20.0 // indirect
)
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Run go mod tidy to download dependencies and create go.sum
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %s\n%v", out, err)
	}

	return dir
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/v2/testutil/ -run TestCreateTestModule -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/v2/testutil/ go.mod go.sum
git commit -m "feat(v2): add test module helper for go/packages-based tests"
```

---

### Task 2: Loader — load packages with type information

**Files:**
- Create: `internal/v2/loader/loader.go`
- Create: `internal/v2/loader/loader_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/v2/loader/loader_test.go
package loader

import (
	"testing"

	"github.com/your-moon/gpc/internal/v2/testutil"
)

func TestLoad(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Find(&orders)
}
`,
	})

	result, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(result.Packages) == 0 {
		t.Fatal("no packages loaded")
	}
	pkg := result.Packages[0]
	if pkg.TypesInfo == nil {
		t.Fatal("TypesInfo is nil")
	}
	if len(pkg.Syntax) == 0 {
		t.Fatal("no syntax trees loaded")
	}
}

func TestLoad_InvalidDir(t *testing.T) {
	_, err := Load("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for invalid directory")
	}
}

func TestLoad_MultiplePackages(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "testmod/models"

func main() {
	_ = models.User{}
}
`,
		"models/models.go": `package models

type User struct {
	ID   int64
	Name string
}
`,
	})

	result, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(result.Packages) < 2 {
		t.Fatalf("expected at least 2 packages, got %d", len(result.Packages))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/v2/loader/ -run TestLoad -v`
Expected: FAIL — package/function not found

- [ ] **Step 3: Implement loader**

```go
// internal/v2/loader/loader.go
package loader

import (
	"fmt"

	"golang.org/x/tools/go/packages"
)

// Result holds the loaded packages with type information.
type Result struct {
	Packages []*packages.Package
}

// Load loads all Go packages in the given directory with full type information.
func Load(dir string) (*Result, error) {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
			packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps,
		Dir: dir,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	// Check for package-level errors
	var errs []error
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			errs = append(errs, fmt.Errorf("%s: %s", pkg.PkgPath, e.Msg))
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("package errors: %v", errs[0])
	}

	return &Result{Packages: pkgs}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/v2/loader/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/v2/loader/
git commit -m "feat(v2): add package loader with go/packages type info"
```

---

### Task 3: Collector — extract Preload chains from typed AST

**Files:**
- Create: `internal/v2/collector/collector.go`
- Create: `internal/v2/collector/collector_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/v2/collector/collector_test.go
package collector

import (
	"testing"

	"github.com/your-moon/gpc/internal/v2/loader"
	"github.com/your-moon/gpc/internal/v2/testutil"
)

func TestCollect_BasicChain(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Find(&orders)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}

	chain := chains[0]
	if len(chain.Preloads) != 1 {
		t.Fatalf("expected 1 preload, got %d", len(chain.Preloads))
	}
	if chain.Preloads[0].Relation != "User" {
		t.Errorf("expected relation 'User', got '%s'", chain.Preloads[0].Relation)
	}
	if chain.Terminal == nil {
		t.Fatal("expected terminal call, got nil")
	}
	if chain.Terminal.Method != "Find" {
		t.Errorf("expected terminal method 'Find', got '%s'", chain.Terminal.Method)
	}
}

func TestCollect_MultiplePreloads(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type Profile struct {
	Bio string
}

type User struct {
	ID      int64
	Name    string
	Profile Profile
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Preload("User.Profile").Find(&orders)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	if len(chains[0].Preloads) != 2 {
		t.Fatalf("expected 2 preloads, got %d", len(chains[0].Preloads))
	}
}

func TestCollect_SeparateChains(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID   int64
	User User
}

type Trip struct {
	ID     int64
	Driver string
}

func GetData(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Find(&orders)

	var trips []Trip
	db.Preload("Driver").Find(&trips)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 2 {
		t.Fatalf("expected 2 chains, got %d", len(chains))
	}
}

func TestCollect_NonGormPreloadIgnored(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type Cache struct{}

func (c *Cache) Preload(key string) {}

type User struct {
	ID int64
}

func GetData(db *gorm.DB) {
	cache := &Cache{}
	cache.Preload("key")

	var users []User
	db.Preload("Name").Find(&users)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain (only gorm), got %d", len(chains))
	}
}

func TestCollect_ConstantPreloadArg(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

const RelUser = "User"

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload(RelUser).Find(&orders)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	if chains[0].Preloads[0].Relation != "User" {
		t.Errorf("expected constant-folded relation 'User', got '%s'", chains[0].Preloads[0].Relation)
	}
}

func TestCollect_DynamicPreloadArg(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID int64
}

func GetData(db *gorm.DB, field string) {
	var users []User
	db.Preload(field).Find(&users)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	if chains[0].Preloads[0].Relation != "" {
		t.Errorf("expected empty relation for dynamic arg, got '%s'", chains[0].Preloads[0].Relation)
	}
	if !chains[0].Preloads[0].Dynamic {
		t.Error("expected Dynamic=true for non-literal arg")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/v2/collector/ -v`
Expected: FAIL — package/types not found

- [ ] **Step 3: Implement collector**

```go
// internal/v2/collector/collector.go
package collector

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"

	"github.com/your-moon/gpc/internal/v2/loader"
)

// PreloadInfo holds info about a single .Preload("X") call.
type PreloadInfo struct {
	Relation string // resolved string value, empty if dynamic
	Dynamic  bool   // true if argument is not a resolvable constant
	Pos      token.Pos
}

// TerminalCall holds info about the terminal call (.Find, .First, etc.)
type TerminalCall struct {
	Method string    // "Find", "First", "FirstOrCreate", etc.
	Arg    ast.Expr  // the &variable argument
	Pos    token.Pos
}

// Chain represents a Preload chain ending in a terminal call.
type Chain struct {
	Preloads []PreloadInfo
	Terminal *TerminalCall
	File     string
	Pkg      *packages.Package
}

var terminalMethods = map[string]bool{
	"Find": true, "First": true, "FirstOrCreate": true,
	"Take": true, "Last": true, "Scan": true,
}

const gormPkgPath = "gorm.io/gorm"

// Collect walks all packages and extracts Preload chains.
func Collect(result *loader.Result) []Chain {
	var chains []Chain

	for _, pkg := range result.Packages {
		for _, file := range pkg.Syntax {
			fileName := pkg.Fset.Position(file.Pos()).Filename
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Look for terminal calls (Find, First, etc.)
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if !terminalMethods[sel.Sel.Name] {
					return true
				}

				// Check the receiver chain is gorm.DB
				if !isGormDBExpr(sel.X, pkg.TypesInfo) {
					return true
				}

				// Extract the terminal call
				var terminal *TerminalCall
				if len(call.Args) > 0 {
					terminal = &TerminalCall{
						Method: sel.Sel.Name,
						Arg:    call.Args[0],
						Pos:    call.Pos(),
					}
				} else {
					return true
				}

				// Walk backward through the chain to collect Preload calls
				preloads := collectPreloads(sel.X, pkg)

				if len(preloads) > 0 {
					chains = append(chains, Chain{
						Preloads: preloads,
						Terminal: terminal,
						File:     fileName,
						Pkg:      pkg,
					})
				}

				return true
			})
		}
	}

	return chains
}

// collectPreloads walks the method chain backward collecting all .Preload() calls.
func collectPreloads(expr ast.Expr, pkg *packages.Package) []PreloadInfo {
	var preloads []PreloadInfo
	cur := expr

	for {
		call, ok := cur.(*ast.CallExpr)
		if !ok {
			break
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}

		if sel.Sel.Name == "Preload" && len(call.Args) > 0 {
			pi := PreloadInfo{Pos: call.Pos()}
			relation, ok := resolveStringArg(call.Args[0], pkg.TypesInfo)
			if ok {
				pi.Relation = relation
			} else {
				pi.Dynamic = true
			}
			preloads = append(preloads, pi)
		}

		cur = sel.X
	}

	// Reverse so order matches source order (outermost first)
	for i, j := 0, len(preloads)-1; i < j; i, j = i+1, j-1 {
		preloads[i], preloads[j] = preloads[j], preloads[i]
	}

	return preloads
}

// resolveStringArg resolves a call argument to a string value.
// Handles string literals and constants.
func resolveStringArg(expr ast.Expr, info *types.Info) (string, bool) {
	// Try constant evaluation (handles both literals and const refs)
	tv, ok := info.Types[expr]
	if ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		return constant.StringVal(tv.Value), true
	}
	return "", false
}

// isGormDBExpr checks if an expression has type *gorm.DB.
func isGormDBExpr(expr ast.Expr, info *types.Info) bool {
	typ := info.TypeOf(expr)
	if typ == nil {
		return false
	}
	// Unwrap pointer
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}
	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Name() == "DB" && obj.Pkg() != nil && obj.Pkg().Path() == gormPkgPath
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/v2/collector/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/v2/collector/
git commit -m "feat(v2): add type-aware Preload chain collector with constant folding"
```

---

### Task 4: Resolver — resolve model type from Find argument

**Files:**
- Create: `internal/v2/resolver/resolver.go`
- Create: `internal/v2/resolver/resolver_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/v2/resolver/resolver_test.go
package resolver

import (
	"testing"

	"github.com/your-moon/gpc/internal/v2/collector"
	"github.com/your-moon/gpc/internal/v2/loader"
	"github.com/your-moon/gpc/internal/v2/testutil"
)

func TestResolve_BasicModel(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Find(&orders)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	chains := collector.Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}

	model := Resolve(chains[0])
	if model == nil {
		t.Fatal("expected resolved model, got nil")
	}
	if model.Name != "Order" {
		t.Errorf("expected model name 'Order', got '%s'", model.Name)
	}
}

func TestResolve_PointerModel(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID int64
}

func GetUser(db *gorm.DB) {
	var user User
	db.Preload("Profile").First(&user)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	chains := collector.Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}

	model := Resolve(chains[0])
	if model == nil {
		t.Fatal("expected resolved model, got nil")
	}
	if model.Name != "User" {
		t.Errorf("expected 'User', got '%s'", model.Name)
	}
}

func TestResolve_CrossPackageModel(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import (
	"gorm.io/gorm"
	"testmod/models"
)

func GetOrders(db *gorm.DB) {
	var orders []models.Order
	db.Preload("User").Find(&orders)
}
`,
		"models/models.go": `package models

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	chains := collector.Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}

	model := Resolve(chains[0])
	if model == nil {
		t.Fatal("expected resolved model, got nil")
	}
	if model.Name != "Order" {
		t.Errorf("expected 'Order', got '%s'", model.Name)
	}
	if model.Pkg == nil || model.Pkg.Name() != "models" {
		t.Errorf("expected package 'models', got %v", model.Pkg)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/v2/resolver/ -v`
Expected: FAIL

- [ ] **Step 3: Implement resolver**

```go
// internal/v2/resolver/resolver.go
package resolver

import (
	"go/ast"
	"go/types"

	"github.com/your-moon/gpc/internal/v2/collector"
)

// Model holds the resolved model type information.
type Model struct {
	Name       string         // struct name (e.g., "Order")
	Pkg        *types.Package // package the struct belongs to
	StructType *types.Struct  // the underlying struct type
	Named      *types.Named   // the named type
}

// Resolve determines the model type from a chain's terminal call argument.
func Resolve(chain collector.Chain) *Model {
	if chain.Terminal == nil || chain.Terminal.Arg == nil {
		return nil
	}

	info := chain.Pkg.TypesInfo
	argType := info.TypeOf(chain.Terminal.Arg)
	if argType == nil {
		return nil
	}

	return extractModel(argType)
}

// extractModel unwraps pointer/slice/array types to find the underlying named struct.
func extractModel(typ types.Type) *Model {
	typ = deref(typ)

	switch t := typ.(type) {
	case *types.Named:
		underlying := t.Underlying()
		if st, ok := underlying.(*types.Struct); ok {
			return &Model{
				Name:       t.Obj().Name(),
				Pkg:        t.Obj().Pkg(),
				StructType: st,
				Named:      t,
			}
		}
		// Could be a named slice/pointer, unwrap further
		return extractModel(underlying)
	case *types.Slice:
		return extractModel(t.Elem())
	case *types.Array:
		return extractModel(t.Elem())
	case *types.Pointer:
		return extractModel(t.Elem())
	}

	return nil
}

// deref removes one layer of pointer indirection (for &variable).
func deref(typ types.Type) types.Type {
	if ptr, ok := typ.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return typ
}

// FieldInfo holds resolved information about a struct field.
type FieldInfo struct {
	Name       string
	Type       types.Type
	StructType *types.Struct // non-nil if the field type is a struct
	Named      *types.Named  // non-nil if the field has a named type
}

// LookupField finds a field by name in a struct, including promoted (embedded) fields.
func LookupField(st *types.Struct, name string) *FieldInfo {
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if field.Name() == name {
			fi := &FieldInfo{
				Name: field.Name(),
				Type: field.Type(),
			}
			// Unwrap to find underlying struct
			underlying := unwrapToStruct(field.Type())
			if underlying != nil {
				fi.StructType = underlying.st
				fi.Named = underlying.named
			}
			return fi
		}
	}

	// Check embedded (promoted) fields
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if !field.Embedded() {
			continue
		}
		embedded := unwrapToStruct(field.Type())
		if embedded != nil {
			if result := LookupField(embedded.st, name); result != nil {
				return result
			}
		}
	}

	return nil
}

type structInfo struct {
	st    *types.Struct
	named *types.Named
}

func unwrapToStruct(typ types.Type) *structInfo {
	typ = derefAll(typ)

	// Handle slices/arrays
	switch t := typ.(type) {
	case *types.Slice:
		typ = derefAll(t.Elem())
	case *types.Array:
		typ = derefAll(t.Elem())
	}

	if named, ok := typ.(*types.Named); ok {
		if st, ok := named.Underlying().(*types.Struct); ok {
			return &structInfo{st: st, named: named}
		}
	}
	if st, ok := typ.(*types.Struct); ok {
		return &structInfo{st: st}
	}
	return nil
}

func derefAll(typ types.Type) types.Type {
	for {
		ptr, ok := typ.(*types.Pointer)
		if !ok {
			return typ
		}
		typ = ptr.Elem()
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/v2/resolver/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/v2/resolver/
git commit -m "feat(v2): add type-based model resolver with cross-package support"
```

---

### Task 5: Validator — recursive relation path validation

**Files:**
- Create: `internal/v2/validator/validator.go`
- Create: `internal/v2/validator/validator_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/v2/validator/validator_test.go
package validator

import (
	"testing"

	"github.com/your-moon/gpc/internal/v2/collector"
	"github.com/your-moon/gpc/internal/v2/loader"
	"github.com/your-moon/gpc/internal/v2/resolver"
	"github.com/your-moon/gpc/internal/v2/testutil"
)

func loadAndCollect(t *testing.T, files map[string]string) ([]collector.Chain, *loader.Result) {
	t.Helper()
	dir := testutil.CreateTestModule(t, files)
	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	chains := collector.Collect(result)
	return chains, result
}

func TestValidate_SimpleValid(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "correct" {
		t.Errorf("expected 'correct', got '%s'", results[0].Status)
	}
}

func TestValidate_SimpleInvalid(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("Customer").Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "error" {
		t.Errorf("expected 'error', got '%s'", results[0].Status)
	}
}

func TestValidate_NestedValid(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type Address struct {
	City string
}

type Profile struct {
	Bio     string
	Address Address
}

type User struct {
	ID      int64
	Profile Profile
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User.Profile.Address").Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "correct" {
		t.Errorf("expected 'correct', got '%s'", results[0].Status)
	}
}

func TestValidate_NestedInvalid_DeepTypo(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type Address struct {
	City string
}

type Profile struct {
	Bio     string
	Address Address
}

type User struct {
	ID      int64
	Profile Profile
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User.Profil.Address").Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "error" {
		t.Errorf("expected 'error', got '%s'", results[0].Status)
	}
	if results[0].Relation != "User.Profil.Address" {
		t.Errorf("expected relation 'User.Profil.Address', got '%s'", results[0].Relation)
	}
}

func TestValidate_DynamicSkipped(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID int64
}

func GetData(db *gorm.DB, field string) {
	var users []User
	db.Preload(field).Find(&users)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "skipped" {
		t.Errorf("expected 'skipped' for dynamic arg, got '%s'", results[0].Status)
	}
}

func TestValidate_CrossPackageNested(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import (
	"gorm.io/gorm"
	"testmod/models"
)

func GetOrders(db *gorm.DB) {
	var orders []models.Order
	db.Preload("User.Profile").Find(&orders)
}
`,
		"models/models.go": `package models

type Profile struct {
	Bio string
}

type User struct {
	ID      int64
	Profile Profile
}

type Order struct {
	ID   int64
	User User
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "correct" {
		t.Errorf("expected 'correct', got '%s'", results[0].Status)
	}
}

func TestValidate_EmbeddedStruct(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type BaseModel struct {
	Creator User
}

type Order struct {
	BaseModel
	ID int64
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("Creator").Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "correct" {
		t.Errorf("expected 'correct' for embedded field, got '%s'", results[0].Status)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/v2/validator/ -v`
Expected: FAIL

- [ ] **Step 3: Implement validator**

```go
// internal/v2/validator/validator.go
package validator

import (
	"strings"

	"github.com/your-moon/gpc/internal/models"
	"github.com/your-moon/gpc/internal/v2/collector"
	"github.com/your-moon/gpc/internal/v2/resolver"
)

// Validate checks all preload chains and returns results.
func Validate(chains []collector.Chain) []models.PreloadResult {
	var results []models.PreloadResult

	for _, chain := range chains {
		model := resolver.Resolve(chain)
		file := chain.File
		pkg := chain.Pkg

		for _, preload := range chain.Preloads {
			line := 0
			if pkg != nil {
				line = pkg.Fset.Position(preload.Pos).Line
			}

			modelName := "Unknown"
			if model != nil {
				if model.Pkg != nil {
					modelName = model.Pkg.Name() + "." + model.Name
				} else {
					modelName = model.Name
				}
			}

			result := models.PreloadResult{
				File:     file,
				Line:     line,
				Relation: preload.Relation,
				Model:    modelName,
			}

			if preload.Dynamic {
				result.Status = "skipped"
				result.Relation = "(dynamic)"
				results = append(results, result)
				continue
			}

			if preload.Relation == "" {
				result.Status = "skipped"
				results = append(results, result)
				continue
			}

			if model == nil {
				result.Status = "unknown"
				results = append(results, result)
				continue
			}

			// Validate the relation path recursively
			if validatePath(preload.Relation, model) {
				result.Status = "correct"
			} else {
				result.Status = "error"
			}

			results = append(results, result)
		}
	}

	return results
}

// validatePath recursively validates a dotted relation path against a model's struct type.
func validatePath(relation string, model *resolver.Model) bool {
	parts := strings.SplitN(relation, ".", 2)
	fieldName := parts[0]

	fi := resolver.LookupField(model.StructType, fieldName)
	if fi == nil {
		return false
	}

	// If there are more segments, recurse into the field's type
	if len(parts) == 2 {
		if fi.StructType == nil {
			return false
		}
		nextModel := &resolver.Model{
			Name:       fi.Name,
			StructType: fi.StructType,
			Named:      fi.Named,
		}
		if fi.Named != nil && fi.Named.Obj() != nil {
			nextModel.Pkg = fi.Named.Obj().Pkg()
			nextModel.Name = fi.Named.Obj().Name()
		}
		return validatePath(parts[1], nextModel)
	}

	return true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/v2/validator/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/v2/validator/
git commit -m "feat(v2): add recursive relation validator with embedded struct support"
```

---

### Task 6: Engine — orchestrate full pipeline + wire into main.go

**Files:**
- Create: `internal/v2/engine/engine.go`
- Create: `internal/v2/engine/engine_test.go`
- Modify: `main.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/v2/engine/engine_test.go
package engine

import (
	"testing"

	"github.com/your-moon/gpc/internal/v2/testutil"
)

func TestAnalyze_EndToEnd(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type Address struct {
	City string
}

type Profile struct {
	Bio     string
	Address Address
}

type User struct {
	ID      int64
	Name    string
	Profile Profile
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Find(&orders)
	db.Preload("User.Profile").Find(&orders)
	db.Preload("User.Profile.Address").Find(&orders)
	db.Preload("User.Profil").Find(&orders)
	db.Preload("Customer").Find(&orders)
}
`,
	})

	results, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}

	// Count statuses
	counts := map[string]int{}
	for _, r := range results {
		counts[r.Status]++
	}

	if counts["correct"] != 3 {
		t.Errorf("expected 3 correct, got %d", counts["correct"])
	}
	if counts["error"] != 2 {
		t.Errorf("expected 2 errors, got %d", counts["error"])
	}
}

func TestAnalyze_CrossPackage(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import (
	"gorm.io/gorm"
	"testmod/db"
)

func GetOrders(dbConn *gorm.DB) {
	var orders []db.Order
	dbConn.Preload("User.Profile").Find(&orders)
	dbConn.Preload("User.Profil").Find(&orders)
}
`,
		"db/models.go": `package db

type Profile struct {
	Bio string
}

type User struct {
	ID      int64
	Profile Profile
}

type Order struct {
	ID   int64
	User User
}
`,
	})

	results, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Status != "correct" {
		t.Errorf("expected first result 'correct', got '%s'", results[0].Status)
	}
	if results[1].Status != "error" {
		t.Errorf("expected second result 'error', got '%s'", results[1].Status)
	}
}

func TestAnalyze_ConstantFolding(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

const RelUser = "User"

type User struct {
	ID int64
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload(RelUser).Find(&orders)
}
`,
	})

	results, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "correct" {
		t.Errorf("expected 'correct', got '%s'", results[0].Status)
	}
	if results[0].Relation != "User" {
		t.Errorf("expected relation 'User', got '%s'", results[0].Relation)
	}
}

func TestAnalyze_NoPreloads(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID int64
}

func GetUsers(db *gorm.DB) {
	var users []User
	db.Find(&users)
}
`,
	})

	results, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/v2/engine/ -v`
Expected: FAIL

- [ ] **Step 3: Implement engine**

```go
// internal/v2/engine/engine.go
package engine

import (
	"github.com/your-moon/gpc/internal/models"
	"github.com/your-moon/gpc/internal/v2/collector"
	"github.com/your-moon/gpc/internal/v2/loader"
	"github.com/your-moon/gpc/internal/v2/validator"
)

// Analyze runs the full v2 analysis pipeline on the given directory.
func Analyze(dir string) ([]models.PreloadResult, error) {
	result, err := loader.Load(dir)
	if err != nil {
		return nil, err
	}

	chains := collector.Collect(result)

	results := validator.Validate(chains)

	return results, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/v2/engine/ -v`
Expected: PASS

- [ ] **Step 5: Wire v2 engine into main.go**

Replace the `runChecker` function in `main.go` to use v2 engine. The v2 engine needs to handle both file and directory targets, but `go/packages` always works on directories. For file targets, use the parent directory and filter results to only the target file.

Update `main.go` — replace the `runChecker` function:

```go
func runChecker(cmd *cobra.Command, args []string) {
	target := args[0]

	info, err := os.Stat(target)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	var dir string
	var filterFile string
	if info.IsDir() {
		dir = target
	} else {
		dir = filepath.Dir(target)
		absPath, _ := filepath.Abs(target)
		filterFile = absPath
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	results, err := engine.Analyze(absDir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Filter to target file if specified
	if filterFile != "" {
		var filtered []models.PreloadResult
		for _, r := range results {
			if r.File == filterFile {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	if outputFormat == "json" {
		err = output.WriteStructuredOutput(results, outputFile, validationOnly, errorsOnly)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Analysis complete! Results written to %s\n", outputFile)
	} else {
		output.WriteConsoleOutput(results, validationOnly, errorsOnly)
	}
}
```

Update imports in `main.go`:

```go
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/your-moon/gpc/internal/models"
	"github.com/your-moon/gpc/internal/output"
	"github.com/your-moon/gpc/internal/v2/engine"
)
```

Remove unused flags (`debugMode`, `verboseMode`) and their `init()` registrations. Remove the `debug` import.

- [ ] **Step 6: Run full test suite**

Run: `go test ./internal/v2/... -v`
Expected: ALL PASS

- [ ] **Step 7: Build and test CLI manually**

Run:
```bash
go build -o gpc .
./gpc ./examples/
```

Expected: output showing correct/error results for the example files.

- [ ] **Step 8: Commit**

```bash
git add internal/v2/engine/ main.go
git commit -m "feat(v2): add engine orchestrator and wire v2 into main.go"
```

---

### Task 7: Handle `clause.Associations` and edge cases

**Files:**
- Modify: `internal/v2/collector/collector.go`
- Modify: `internal/v2/collector/collector_test.go`
- Modify: `internal/v2/validator/validator.go`
- Modify: `internal/v2/validator/validator_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/v2/collector/collector_test.go`:

```go
func TestCollect_ClauseAssociations(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	ID int64
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload(clause.Associations).Find(&orders)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	if chains[0].Preloads[0].Relation != "clause.Associations" {
		t.Errorf("expected 'clause.Associations', got '%s'", chains[0].Preloads[0].Relation)
	}
}
```

Add to `internal/v2/validator/validator_test.go`:

```go
func TestValidate_ClauseAssociations(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	ID int64
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload(clause.Associations).Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "correct" {
		t.Errorf("expected 'correct' for clause.Associations, got '%s'", results[0].Status)
	}
}

func TestValidate_EmptyRelation(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type Order struct {
	ID int64
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("").Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "error" {
		t.Errorf("expected 'error' for empty relation, got '%s'", results[0].Status)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/v2/collector/ ./internal/v2/validator/ -v -run "ClauseAssociations|EmptyRelation"`
Expected: FAIL

- [ ] **Step 3: Update collector to handle clause.Associations**

In `internal/v2/collector/collector.go`, update `resolveStringArg`:

```go
// resolveStringArg resolves a call argument to a string value.
// Handles string literals, constants, and clause.Associations.
func resolveStringArg(expr ast.Expr, info *types.Info) (string, bool) {
	// Check for clause.Associations (selector expression)
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "Associations" {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "clause" {
				return "clause.Associations", true
			}
		}
	}

	// Try constant evaluation (handles both literals and const refs)
	tv, ok := info.Types[expr]
	if ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		return constant.StringVal(tv.Value), true
	}
	return "", false
}
```

- [ ] **Step 4: Update validator to handle clause.Associations and empty relations**

In `internal/v2/validator/validator.go`, in the `Validate` function, add handling before the `validatePath` call:

```go
			// clause.Associations is always valid in GORM
			if preload.Relation == "clause.Associations" {
				result.Relation = "clause.Associations"
				result.Status = "correct"
				results = append(results, result)
				continue
			}

			// Empty relation is always an error
			if preload.Relation == "" && !preload.Dynamic {
				result.Status = "error"
				results = append(results, result)
				continue
			}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/v2/... -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/v2/collector/ internal/v2/validator/
git commit -m "feat(v2): handle clause.Associations and empty relation edge cases"
```

---

### Task 8: Handle variable-assigned db chains (non-inline Find)

**Files:**
- Modify: `internal/v2/collector/collector.go`
- Modify: `internal/v2/collector/collector_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/v2/collector/collector_test.go`:

```go
func TestCollect_AssignedVariable(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	query := db.Preload("User")
	query.Find(&orders)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	if len(chains[0].Preloads) != 1 {
		t.Fatalf("expected 1 preload, got %d", len(chains[0].Preloads))
	}
	if chains[0].Preloads[0].Relation != "User" {
		t.Errorf("expected 'User', got '%s'", chains[0].Preloads[0].Relation)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/v2/collector/ -run TestCollect_AssignedVariable -v`
Expected: FAIL — assigned variable chain not detected (the collector currently only walks inline method chains)

- [ ] **Step 3: Update collector to track variable assignments**

The approach: when we find a terminal call like `query.Find(&orders)`, check if `query` is a local variable. If so, find the assignment and collect preloads from it. In `internal/v2/collector/collector.go`, add a second pass that resolves variable-assigned chains.

Add this function and update `Collect`:

```go
// Collect walks all packages and extracts Preload chains.
func Collect(result *loader.Result) []Chain {
	var chains []Chain

	for _, pkg := range result.Packages {
		for _, file := range pkg.Syntax {
			fileName := pkg.Fset.Position(file.Pos()).Filename

			// First pass: collect inline chains (Preload().Find())
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if !terminalMethods[sel.Sel.Name] {
					return true
				}

				if !isGormDBExpr(sel.X, pkg.TypesInfo) {
					return true
				}

				var terminal *TerminalCall
				if len(call.Args) > 0 {
					terminal = &TerminalCall{
						Method: sel.Sel.Name,
						Arg:    call.Args[0],
						Pos:    call.Pos(),
					}
				} else {
					return true
				}

				// Collect preloads from the inline chain
				preloads := collectPreloads(sel.X, pkg)

				// If no preloads found inline, check if the receiver is a variable
				// that was assigned from a chain containing Preload calls
				if len(preloads) == 0 {
					preloads = collectPreloadsFromVariable(sel.X, file, pkg)
				}

				if len(preloads) > 0 {
					chains = append(chains, Chain{
						Preloads: preloads,
						Terminal: terminal,
						File:     fileName,
						Pkg:      pkg,
					})
				}

				return true
			})
		}
	}

	return chains
}

// collectPreloadsFromVariable resolves preloads when the receiver is a variable
// e.g., query := db.Preload("User"); query.Find(&orders)
func collectPreloadsFromVariable(expr ast.Expr, file *ast.File, pkg *packages.Package) []PreloadInfo {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return nil
	}

	// Find the definition of this variable
	obj := pkg.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return nil
	}

	// Walk the file to find the assignment
	var preloads []PreloadInfo
	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, lhs := range assign.Lhs {
			lhsIdent, ok := lhs.(*ast.Ident)
			if !ok {
				continue
			}
			lhsObj := pkg.TypesInfo.ObjectOf(lhsIdent)
			if lhsObj != obj {
				continue
			}
			if i < len(assign.Rhs) {
				// Found the assignment, collect preloads from the RHS
				if call, ok := assign.Rhs[i].(*ast.CallExpr); ok {
					preloads = collectPreloadsFromCall(call, pkg)
				}
			}
		}
		return true
	})

	return preloads
}

// collectPreloadsFromCall extracts preloads from a call expression tree.
func collectPreloadsFromCall(call *ast.CallExpr, pkg *packages.Package) []PreloadInfo {
	var preloads []PreloadInfo

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	if sel.Sel.Name == "Preload" && len(call.Args) > 0 {
		pi := PreloadInfo{Pos: call.Pos()}
		relation, ok := resolveStringArg(call.Args[0], pkg.TypesInfo)
		if ok {
			pi.Relation = relation
		} else {
			pi.Dynamic = true
		}
		preloads = append(preloads, pi)
	}

	// Recurse into the receiver
	if innerCall, ok := sel.X.(*ast.CallExpr); ok {
		inner := collectPreloadsFromCall(innerCall, pkg)
		preloads = append(inner, preloads...)
	}

	return preloads
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/v2/collector/ -v`
Expected: ALL PASS

- [ ] **Step 5: Run full v2 test suite**

Run: `go test ./internal/v2/... -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/v2/collector/
git commit -m "feat(v2): support variable-assigned Preload chains"
```

---

### Task 9: Clean up — remove unused v1 code, update CLAUDE.md

**Files:**
- Modify: `main.go` (final cleanup of unused imports)
- Modify: `CLAUDE.md`

- [ ] **Step 1: Verify the build is clean**

Run:
```bash
go build -o gpc .
go vet ./...
```

Expected: no errors, no warnings

- [ ] **Step 2: Run all v2 tests one final time**

Run: `go test ./internal/v2/... -v -count=1`
Expected: ALL PASS

- [ ] **Step 3: Test CLI with examples directory**

Run:
```bash
./gpc ./examples/
./gpc ./examples/basic.go
./gpc -o json -f /tmp/test_v2.json ./examples/
cat /tmp/test_v2.json
```

Verify: output shows correct/error results. JSON output has proper structure.

- [ ] **Step 4: Update CLAUDE.md**

Replace the Architecture section to reflect v2 pipeline. Update Known Issues to remove v1-specific bugs. Keep CLI flags section.

- [ ] **Step 5: Commit**

```bash
git add main.go CLAUDE.md
git commit -m "feat(v2): finalize v2 integration, update project docs"
```

---

## Summary

| Task | Component | Subagent Model | Why |
|------|-----------|----------------|-----|
| 1 | Test helper + dependency | haiku | Boilerplate go.mod writing, no design decisions |
| 2 | Loader | haiku | Thin wrapper around go/packages.Load |
| 3 | Collector | sonnet/opus | Core logic: AST walk + type checking + constant folding |
| 4 | Resolver | haiku | Mechanical type unwrapping (pointer/slice/named) |
| 5 | Validator | sonnet/opus | Recursive validation + embedded struct handling |
| 6 | Engine + main.go | sonnet/opus | Integration wiring, CLI changes |
| 7 | Edge cases | haiku | Small additions to existing code |
| 8 | Variable chains | sonnet/opus | Requires understanding of go/types ObjectOf |
| 9 | Cleanup | haiku | Remove dead code, update docs |
