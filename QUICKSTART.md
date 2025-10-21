# Quick Start Guide

Get started with GORM Preload Checker in 5 minutes!

## 1. Installation

```bash
go install github.com/your-moon/gpc@latest
```

## 2. Run on Your Project

```bash
# Check your entire project
gpc ./...

# Check specific package
gpc ./internal/models/

# Check single file
gpc ./main.go
```

## 3. Example Output

### ✅ No Errors (All Good!)

```bash
$ gpc ./main.go
# (no output - everything is correct)
```

### ❌ Errors Found

```bash
$ gpc ./main.go
./main.go:26:2: invalid preload: User.Profil not found in Order
./main.go:31:2: invalid preload: Customer not found in Order
```

## 4. Common Use Cases

### Basic Usage

```go
type User struct {
    Name string
}

type Order struct {
    User User
}

func GetOrders(db *gorm.DB) {
    var orders []Order

    // ✅ Correct
    db.Preload("User").Find(&orders)

    // ❌ Typo - will be caught!
    db.Preload("Usr").Find(&orders)
}
```

### With Conditions

```go
// ✅ Works with conditions - validates "User" relation
db.Preload("User", "active = ?", true).Find(&orders)

// ❌ Still catches typos
db.Preload("Usr", "active = ?", true).Find(&orders)
```

### Nested Relations

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

    // ✅ Correct nested preload
    db.Preload("User.Profile.Address").Find(&orders)

    // ❌ Typo in nested path - caught!
    db.Preload("User.Profil.Address").Find(&orders)
}
```

## 5. Integration

### Makefile

```makefile
lint:
	go vet ./...
	gpc ./...
```

### GitHub Actions

```yaml
- name: Run GORM Preload Checker
  run: |
    go install github.com/your-moon/gpc/cmd/gpc@latest
    gpc ./...
```

### Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Running GORM Preload Checker..."
gpc ./...
if [ $? -ne 0 ]; then
    echo "❌ Preload validation failed!"
    exit 1
fi
```

## 6. What Gets Checked?

✅ **Validated:**

- String literal relation names
- Nested relations (any depth)
- Relations with conditions
- Pointer and slice types

⚠️ **Not Validated:**

- Variable relation names
- Dynamic relation names
- Relations from external packages (currently)

## 7. Tips

### Keep Preload and Find Together

```go
// ✅ Good - easy to validate
db.Preload("User").Find(&orders)

// ⚠️ May not validate - type inference harder
query := db.Preload("User")
// ... many lines later ...
query.Find(&orders)
```

### Use String Literals

```go
// ✅ Good - can be validated
db.Preload("User").Find(&orders)

// ⚠️ Cannot be validated
relation := "User"
db.Preload(relation).Find(&orders)
```

## 8. Getting Help

- 📖 [Full Documentation](README.md)
- 🔍 [Features & Limitations](docs/FEATURES.md)
- 💡 [Examples](examples/)
- 🐛 [Report Issues](https://github.com/your-moon/gpc/issues)

## 9. Next Steps

1. ✅ Install the tool
2. ✅ Run it on your project
3. ✅ Fix any errors found
4. ✅ Add to your CI/CD pipeline
5. ✅ Share with your team!

---

**Happy coding! 🚀**
