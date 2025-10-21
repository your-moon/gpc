# Features & Limitations

## ✅ What It Does

### 1. Validates String Literal Preload Paths

The analyzer checks GORM `Preload()` calls with string literal arguments:

```go
db.Preload("User").Find(&orders)              // ✅ Validated
db.Preload("User.Profile").Find(&orders)      // ✅ Validated
db.Preload("User.Profile.Address").Find(&orders) // ✅ Validated
```

### 2. Handles Additional Arguments

The analyzer correctly handles `Preload()` calls with conditions and additional arguments:

```go
// ✅ Validates "Posts" relation, ignores the condition arguments
db.Preload("Posts", "published = ?", true).Find(&authors)

// ✅ Validates "Posts" relation, ignores the function argument
db.Preload("Posts", func(db *gorm.DB) *gorm.DB {
    return db.Where("published = ?", true)
}).Find(&authors)

// ✅ Validates "Comments.Post" relation, ignores conditions
db.Preload("Comments.Post", "published = ? AND views > ?", true, 100).Find(&authors)
```

**How it works:** The analyzer only validates the first argument (the relation path string) and ignores all subsequent arguments (`args ...interface{}`).

### 3. Detects Typos and Invalid Relations

```go
db.Preload("Profil").Find(&users)           // ❌ Error: Profil not found
db.Preload("User.Profil").Find(&orders)     // ❌ Error: User.Profil not found
db.Preload("NonExistent").Find(&orders)     // ❌ Error: NonExistent not found
```

### 4. Supports Nested Relations

```go
// Deep nesting is fully supported
db.Preload("Order.User.Profile.Address.Country").Find(&items)
```

### 5. Works with Pointer Types

```go
type User struct {
    Profile *Profile  // ✅ Pointer types are handled
}

db.Preload("Profile").Find(&users)  // ✅ Works correctly
```

### 6. Supports Slice Relations

```go
type Author struct {
    Posts []Post  // ✅ Slice relations are supported
}

db.Preload("Posts").Find(&authors)  // ✅ Works correctly
```

## ⚠️ Limitations

### 1. Variable Relation Names

The analyzer **cannot validate** dynamic or variable relation names:

```go
relationName := "User"
db.Preload(relationName).Find(&orders)  // ⚠️ Skipped (not a string literal)

for _, rel := range relations {
    db.Preload(rel).Find(&data)  // ⚠️ Skipped
}
```

**Why:** Static analysis cannot determine runtime values.

### 2. Model Type Inference

The analyzer uses a simplified heuristic to find the model type from nearby `.Find()` calls:

```go
// ✅ Works - Find call is nearby
db.Preload("User").Find(&orders)

// ⚠️ May not work - complex call chains
query := db.Preload("User")
// ... many lines later ...
query.Find(&orders)
```

**Workaround:** Keep `Preload()` and `Find()` calls close together.

### 3. Custom Struct Tags

The analyzer does not currently support custom GORM struct tags:

```go
type User struct {
    UserProfile Profile `gorm:"foreignKey:ProfileID;references:ID"`
}

// The analyzer looks for "UserProfile", not custom tag names
db.Preload("UserProfile").Find(&users)  // ✅ Validated as "UserProfile"
```

### 4. Interface Types

The analyzer may not work correctly with interface types:

```go
type User struct {
    Data interface{}  // ⚠️ May not be validated correctly
}
```

### 5. Embedded Structs

Currently has limited support for embedded/anonymous structs:

```go
type User struct {
    Profile  // ⚠️ Anonymous embedding may not work
}
```

## 🔮 Future Enhancements

Planned features for future versions:

- [ ] Better call chain tracking for model type inference
- [ ] Support for `Joins()` validation
- [ ] Custom struct tag support
- [ ] Embedded struct support
- [ ] Configuration file for custom rules
- [ ] IDE quick-fix suggestions
- [ ] Support for `Association()` validation

## 📊 Comparison with Runtime Validation

| Feature                | Static Analysis (This Tool) | Runtime        |
| ---------------------- | --------------------------- | -------------- |
| **Detection Time**     | Compile time                | Runtime        |
| **Performance Impact** | None                        | Minimal        |
| **Coverage**           | String literals only        | All cases      |
| **False Positives**    | Rare                        | None           |
| **IDE Integration**    | Yes                         | No             |
| **CI/CD Integration**  | Yes                         | Requires tests |

## 🎯 Best Practices

### 1. Use String Literals

```go
// ✅ Good - can be validated
db.Preload("User.Profile").Find(&orders)

// ❌ Avoid - cannot be validated
relation := "User.Profile"
db.Preload(relation).Find(&orders)
```

### 2. Keep Preload and Find Close

```go
// ✅ Good - easy to infer model type
db.Preload("User").
    Preload("User.Profile").
    Find(&orders)

// ⚠️ Less ideal - harder to infer
query := db.Preload("User")
// ... many lines ...
query.Find(&orders)
```

### 3. Use Consistent Naming

```go
// ✅ Good - clear and consistent
type Order struct {
    User User
}

// ⚠️ Confusing - field name doesn't match type
type Order struct {
    Customer User  // Preload("Customer"), not Preload("User")
}
```

### 4. Combine with Tests

Static analysis catches most errors, but runtime tests are still valuable:

```go
func TestPreloads(t *testing.T) {
    // Test that preloads actually work
    var orders []Order
    result := db.Preload("User").Find(&orders)
    assert.NoError(t, result.Error)
}
```

## 🤝 Contributing

Found a limitation that affects your use case? Please:

1. Open an issue describing the scenario
2. Provide a minimal code example
3. Explain the expected behavior

We're always looking to improve the analyzer!
