# Troubleshooting Guide

## 🚨 Common CI/CD Issues

### Issue: "directory not found" Error

**Error:**

```
stat /home/runner/work/gpc/gpc/cmd/gpc: directory not found
Error: Process completed with exit code 1.
```

**Causes:**

1. Wrong working directory
2. Missing files in repository
3. Incorrect build command

**Solutions:**

#### 1. Check Directory Structure

```bash
# Verify you're in the right directory
pwd
ls -la

# Check if cmd directory exists
ls -la cmd/
ls -la cmd/gpc/
```

#### 2. Use Correct Build Command

```bash
# ❌ Wrong - missing output name
go build -v ./cmd/gpc/

# ✅ Correct - specify output name
go build -o gpc ./cmd/gpc/
```

#### 3. Test Locally First

```bash
# Run the test script
make test-build

# Or manually
./test-build.sh
```

#### 4. Check File Permissions

```bash
# Make sure files are executable
chmod +x test-build.sh
chmod +x gpc
```

### Issue: Binary Not Found After Build

**Error:**

```
./gpc: No such file or directory
```

**Solution:**

```bash
# Build with explicit output name
go build -o gpc ./cmd/gpc/

# Verify binary exists
ls -la gpc
file gpc
```

### Issue: Go Module Issues

**Error:**

```
go: cannot find main module
```

**Solution:**

```bash
# Make sure go.mod exists
ls -la go.mod

# Initialize if missing
go mod init github.com/your-moon/gorm-gpc

# Download dependencies
go mod download
go mod tidy
```

## 🔧 CI/CD Best Practices

### 1. Use Explicit Paths

```yaml
# ✅ Good - explicit paths
- name: Build
  run: go build -o gpc ./cmd/gpc/

# ❌ Avoid - relative paths can be unclear
- name: Build
  run: go build ./cmd/gpc/
```

### 2. Add Debug Steps

```yaml
- name: Debug
  run: |
    echo "Working directory: $(pwd)"
    echo "Directory contents:"
    ls -la
    echo "Go version:"
    go version
```

### 3. Test Binary After Build

```yaml
- name: Verify build
  run: |
    ls -la gpc
    file gpc
    ./gpc --help || echo "Binary built but no help flag"
```

## 🐛 Debugging Commands

### Check Project Structure

```bash
# Verify all files exist
find . -name "*.go" | head -10
ls -la cmd/gpc/main.go
cat cmd/gpc/main.go
```

### Test Build Process

```bash
# Step-by-step build test
go mod download
go build -v ./cmd/gpc/
go build -o gpc ./cmd/gpc/
./gpc ./testdata/correct.go
```

### Check Go Environment

```bash
# Verify Go setup
go version
go env GOPATH
go env GOROOT
go list -m all
```

## 📋 Pre-commit Checklist

Before pushing to CI/CD:

1. ✅ **Test locally:**

   ```bash
   make test
   make build
   ./gpc ./testdata/correct.go
   ```

2. ✅ **Check file structure:**

   ```bash
   ls -la cmd/gpc/main.go
   cat go.mod
   ```

3. ✅ **Verify build:**

   ```bash
   go build -o gpc ./cmd/gpc/
   ls -la gpc
   ```

4. ✅ **Test binary:**
   ```bash
   ./gpc ./testdata/correct.go
   ./gpc ./testdata/testdata.go || true
   ```

## 🚀 Quick Fixes

### If CI is failing:

1. **Add debug output:**

   ```yaml
   - name: Debug
     run: |
       pwd
       ls -la
       ls -la cmd/
   ```

2. **Use absolute paths:**
   ```bash
   go build -o /tmp/gpc ./cmd/gpc/
   /tmp/gpc ./testdata/correct.go
   ```

## 📞 Getting Help

If you're still having issues:

1. **Check the logs** - Look for the exact error message
2. **Run locally** - Test the same commands on your machine
3. **Use debug mode** - Add `-v` flags to see verbose output
4. **Check file permissions** - Make sure files are readable
5. **Verify Go version** - Ensure compatible Go version

## 🔍 Common File Issues

### Missing main.go

```bash
# Check if main.go exists
ls -la cmd/gpc/main.go

# If missing, create it:
mkdir -p cmd/gpc
cat > cmd/gpc/main.go << 'EOF'
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"
	gpc "github.com/your-moon/gorm-gpc"
)

func main() {
	singlechecker.Main(gpc.Analyzer)
}
EOF
```

### Wrong module name

```bash
# Check go.mod
cat go.mod

# Should be:
# module github.com/your-moon/gorm-gpc
```

### Missing dependencies

```bash
# Download dependencies
go mod download
go mod tidy

# Check if all dependencies are available
go list -m all
```

## ✅ Success Indicators

Your CI/CD should show:

- ✅ Tests pass
- ✅ Binary builds successfully
- ✅ Binary runs without errors
- ✅ Correct files are analyzed
- ✅ Errors are detected in test files

If all these pass, your setup is working correctly! 🎉
