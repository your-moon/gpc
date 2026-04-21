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
