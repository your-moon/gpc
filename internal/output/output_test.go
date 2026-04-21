package output

import (
	"os"
	"testing"

	"github.com/your-moon/gpc/internal/models"
)

func TestWriteStructuredOutput(t *testing.T) {
	results := []models.PreloadResult{
		{File: "test.go", Line: 10, Relation: "User", Model: "Order", Status: "valid"},
		{File: "test.go", Line: 15, Relation: "Invalid", Model: "Order", Status: "error"},
		{File: "test.go", Line: 20, Relation: "(dynamic)", Model: "Order", Status: "skipped"},
	}

	testFile := "test_output.json"
	err := WriteStructuredOutput(results, testFile, false, false)
	if err != nil {
		t.Fatalf("WriteStructuredOutput: %v", err)
	}
	defer os.Remove(testFile)

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	for _, field := range []string{"total", "valid", "errors", "skipped", "results"} {
		if !contains(string(content), field) {
			t.Errorf("output missing field %q", field)
		}
	}
}

func TestWriteStructuredOutput_Empty(t *testing.T) {
	testFile := "test_empty.json"
	err := WriteStructuredOutput(nil, testFile, false, false)
	if err != nil {
		t.Fatalf("WriteStructuredOutput: %v", err)
	}
	defer os.Remove(testFile)
}

func TestWriteStructuredOutput_ErrorsOnly(t *testing.T) {
	results := []models.PreloadResult{
		{File: "test.go", Line: 10, Relation: "User", Model: "Order", Status: "valid"},
		{File: "test.go", Line: 15, Relation: "Bad", Model: "Order", Status: "error"},
	}

	testFile := "test_errors_only.json"
	err := WriteStructuredOutput(results, testFile, false, true)
	if err != nil {
		t.Fatalf("WriteStructuredOutput: %v", err)
	}
	defer os.Remove(testFile)

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if contains(string(content), `"status": "valid"`) {
		t.Error("errors-only output should not contain valid results")
	}
}

func TestFilterResults(t *testing.T) {
	results := []models.PreloadResult{
		{Status: "valid"},
		{Status: "error"},
		{Status: "skipped"},
	}

	errOnly := filterResults(results, false, true)
	if len(errOnly) != 1 || errOnly[0].Status != "error" {
		t.Errorf("errors-only: expected 1 error, got %d", len(errOnly))
	}

	validOnly := filterResults(results, true, false)
	if len(validOnly) != 2 {
		t.Errorf("validation-only: expected 2 (valid+error), got %d", len(validOnly))
	}

	all := filterResults(results, false, false)
	if len(all) != 3 {
		t.Errorf("unfiltered: expected 3, got %d", len(all))
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
