# Examples

This directory contains example code demonstrating the usage of GORM Preload Checker.

## Files

### `basic.go`

Basic examples showing correct usage of GORM Preload with simple nested relations.

**Run the linter:**

```bash
preloadcheck ./examples/basic.go
```

Expected: No errors (all preloads are correct)

### `errors.go`

Examples of common errors that the linter will catch, including typos and non-existent relations.

**Run the linter:**

```bash
preloadcheck ./examples/errors.go
```

Expected: Multiple errors detected

### `complex.go`

Complex examples with deep nesting, multiple relations, and slice relations.

**Run the linter:**

```bash
preloadcheck ./examples/complex.go
```

Expected: Errors in the `ComplexErrors` function

## Running Examples

### Check all examples

```bash
make run-examples
```

Or manually:

```bash
preloadcheck ./examples/
```

### Check specific example

```bash
preloadcheck ./examples/basic.go
```

## Learning Path

1. **Start with `basic.go`** - Understand correct usage
2. **Review `errors.go`** - Learn common mistakes
3. **Study `complex.go`** - See advanced patterns

## Integration in Your Project

Copy these patterns into your own codebase and run the linter:

```bash
preloadcheck ./...
```

This will catch any typos or invalid relation names before they cause runtime errors!
