# GORM Preload Checker

[![Go Report Card](https://goreportcard.com/badge/github.com/your-moon/gorm-preloadcheck)](https://goreportcard.com/report/github.com/your-moon/gorm-preloadcheck)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A static analysis tool for [GORM](https://gorm.io/) that detects typos and invalid relation names in `Preload()` calls at compile time.

## 🎯 Problem

When using GORM's `Preload()` method, typos in relation names are only caught at runtime, leading to:

- Silent failures in production
- Missing data in queries
- Hard-to-debug issues

```go
// This typo won't be caught until runtime! 😱
db.Preload("User.Profil.Address").Find(&orders)  // "Profil" should be "Profile"
```

## ✨ Solution

This linter catches these errors during development:

```bash
$ preloadcheck ./...
./main.go:26:2: invalid preload: User.Profil not found in Order
```

## 📦 Installation

### As a linter

```bash
go install github.com/your-moon/gorm-preloadcheck/cmd/preloadcheck@latest
```

### As a library

```bash
go get github.com/your-moon/gorm-preloadcheck
```

## 🚀 Usage

### Command Line

Check a single file:

```bash
preloadcheck ./main.go
```

Check a package:

```bash
preloadcheck ./...
```

Check specific directory:

```bash
preloadcheck ./internal/models/
```

### IDE Integration

#### VS Code

Add to your `settings.json`:

```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--enable=preloadcheck"]
}
```

#### GoLand/IntelliJ

1. Go to `Settings` → `Tools` → `File Watchers`
2. Add new watcher with program: `preloadcheck`
3. Arguments: `$FilePath$`

### golangci-lint Integration

Add to your `.golangci.yml`:

```yaml
linters-settings:
  preloadcheck:
    enabled: true

linters:
  enable:
    - preloadcheck
```

## 📖 Examples

### Valid Usage ✅

```go
type Address struct {
    City string
}

type Profile struct {
    Address Address
}

type User struct {
    Profile Profile
}

type Order struct {
    User User
}

func GetOrders(db *gorm.DB) {
    var orders []Order

    // ✅ All relation names are correct
    db.Preload("User").Find(&orders)
    db.Preload("User.Profile").Find(&orders)
    db.Preload("User.Profile.Address").Find(&orders)
}
```

### Invalid Usage ❌

```go
func GetOrders(db *gorm.DB) {
    var orders []Order

    // ❌ Typo: "Profil" instead of "Profile"
    db.Preload("User.Profil.Address").Find(&orders)
    // Error: invalid preload: User.Profil not found in Order

    // ❌ Wrong relation name
    db.Preload("Customer").Find(&orders)
    // Error: invalid preload: Customer not found in Order

    // ❌ Nested typo
    db.Preload("User.Profile.Addres").Find(&orders)
    // Error: invalid preload: User.Profile.Addres not found in Order
}
```

## 🛠️ Development

### Prerequisites

- Go 1.24 or higher
- GORM v1.31.0 or higher

### Build from source

```bash
git clone https://github.com/your-moon/gorm-preloadcheck.git
cd gorm-preloadcheck
make build
```

### Run tests

```bash
make test
```

### Run linter on itself

```bash
make lint
```

## 🧪 Testing

The project includes comprehensive tests:

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### Quick Start for Contributors

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for your changes
5. Ensure all tests pass (`make test`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## 📝 How It Works

The analyzer:

1. **Finds Preload calls**: Scans your code for `db.Preload()` calls
2. **Extracts relation paths**: Gets the string argument (e.g., "User.Profile.Address")
3. **Validates each level**: Checks if each relation exists in the corresponding struct
4. **Reports errors**: Provides clear error messages with file location

### Current Limitations

- Only works with string literals (not variables)
- Requires the model type to be inferable from nearby `.Find()` calls
- Does not support dynamic relation names

### Roadmap

- [ ] Support for variable relation names
- [ ] Better call chain tracking
- [ ] Support for `Joins()` validation
- [ ] Custom struct tag support
- [ ] IDE quick-fix suggestions

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [GORM](https://gorm.io/) - The fantastic Go ORM
- [golang.org/x/tools/go/analysis](https://pkg.go.dev/golang.org/x/tools/go/analysis) - Go static analysis framework

## 📮 Support

- 🐛 [Report a bug](https://github.com/your-moon/gorm-preloadcheck/issues/new?template=bug_report.md)
- 💡 [Request a feature](https://github.com/your-moon/gorm-preloadcheck/issues/new?template=feature_request.md)
- 💬 [Ask a question](https://github.com/your-moon/gorm-preloadcheck/discussions)

## ⭐ Star History

If you find this project useful, please consider giving it a star! ⭐

---

Made with ❤️ by the Go community
