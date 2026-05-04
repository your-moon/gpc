# Examples

Sample code for trying gpc against. Each file is a standalone package
showing one shape of usage.

| File | Contents |
|------|----------|
| `basic.go` | Valid preloads on simple nested relations |
| `errors.go` | Typos and missing fields the checker should flag |
| `complex.go` | Deep nesting and slice relations |
| `with_conditions.go` | `Preload(name, args...)` form, plus dynamic args (skipped) |

Run gpc against any of them:

```bash
gpc ./examples/
gpc ./examples/errors.go
```
