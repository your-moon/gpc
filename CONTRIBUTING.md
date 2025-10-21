# Contributing to GORM Preload Checker

First off, thank you for considering contributing to GORM Preload Checker! 🎉

## Code of Conduct

This project and everyone participating in it is governed by our commitment to providing a welcoming and inclusive environment for all contributors.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

**Bug Report Template:**

- **Description**: Clear description of the bug
- **Steps to Reproduce**: Minimal code example
- **Expected Behavior**: What you expected to happen
- **Actual Behavior**: What actually happened
- **Environment**: Go version, OS, GORM version
- **Additional Context**: Any other relevant information

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, include:

- **Clear title and description**
- **Use case**: Why this enhancement would be useful
- **Possible implementation**: If you have ideas
- **Examples**: Code examples if applicable

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** with clear, descriptive commits
3. **Add tests** for any new functionality
4. **Ensure tests pass**: Run `make test`
5. **Update documentation** if needed
6. **Submit a pull request**

## Development Setup

### Prerequisites

```bash
# Install Go 1.24 or higher
go version

# Clone your fork
git clone https://github.com/YOUR_USERNAME/gpc.git
cd gpc

# Install dependencies
go mod download
```

### Building

```bash
# Build the binary
make build

# Or manually
go build -o gpc ./cmd/gpc/
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run tests with race detector
go test -race ./...

# Run specific test
go test -v -run TestAnalyzer
```

### Code Style

We follow standard Go conventions:

- Use `gofmt` to format your code
- Use `golint` for style checks
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Write clear, self-documenting code
- Add comments for exported functions

```bash
# Format code
make fmt

# Run linter
make lint
```

## Project Structure

```
.
├── cmd/
│   └── gpc/              # CLI entry point
│       └── main.go
├── testdata/             # Test files
│   ├── testdata.go       # Test with errors
│   └── correct.go        # Test with correct code
├── preloadcheck.go       # Main analyzer logic
├── preloadcheck_test.go  # Tests
├── go.mod                # Dependencies
└── README.md             # Documentation
```

## Writing Tests

### Unit Tests

```go
func TestAnalyzer(t *testing.T) {
    // Test analyzer properties
    if Analyzer.Name != "preloadcheck" {
        t.Errorf("Expected name 'preloadcheck', got '%s'", Analyzer.Name)
    }
}
```

### Integration Tests

Add test files to `testdata/` directory:

```go
// testdata/example.go
package testdata

import "gorm.io/gorm"

type User struct {
    Name string
}

func Example() {
    var db *gorm.DB
    var users []User

    db.Preload("InvalidRelation").Find(&users) // want "invalid preload"
}
```

## Commit Messages

We follow conventional commits:

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Adding or updating tests
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

Examples:

```
feat: add support for Joins validation
fix: handle pointer types in struct fields
docs: update installation instructions
test: add test cases for nested relations
```

## Review Process

1. **Automated checks**: CI runs tests and linters
2. **Code review**: Maintainers review your code
3. **Feedback**: Address any requested changes
4. **Merge**: Once approved, your PR will be merged

## Areas for Contribution

### Good First Issues

- Documentation improvements
- Adding more test cases
- Fixing typos
- Adding examples

### Advanced Issues

- Improving type inference
- Supporting dynamic relation names
- Adding support for `Joins()` validation
- Performance optimizations

## Getting Help

- 💬 [GitHub Discussions](https://github.com/your-moon/gpc/discussions)
- 🐛 [Issue Tracker](https://github.com/your-moon/gpc/issues)
- 📧 Reach out to maintainers

## Recognition

Contributors will be:

- Listed in the README
- Mentioned in release notes
- Part of the project's history

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing! 🚀
