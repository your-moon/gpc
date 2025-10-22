package output

import (
	"os"
	"testing"

	"github.com/your-moon/gpc/internal/models"
)

func TestWriteStructuredOutput(t *testing.T) {
	// Create test results
	results := []models.PreloadResult{
		{
			File:     "test.go",
			Line:     10,
			Relation: "User",
			Model:    "Order",
			Variable: "orders",
			FindLine: 10,
			Status:   "correct",
		},
		{
			File:     "test.go",
			Line:     15,
			Relation: "Profile",
			Model:    "Unknown",
			Status:   "unknown",
		},
	}

	// Test JSON output
	testFile := "test_output.json"
	err := WriteStructuredOutput(results, testFile)
	if err != nil {
		t.Fatalf("Failed to write structured output: %v", err)
	}
	defer os.Remove(testFile)

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("Output file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Basic content verification
	contentStr := string(content)
	if !contains(contentStr, "total_preloads") {
		t.Errorf("Output missing total_preloads field")
	}
	if !contains(contentStr, "correct") {
		t.Errorf("Output missing correct field")
	}
	if !contains(contentStr, "unknown") {
		t.Errorf("Output missing unknown field")
	}
	if !contains(contentStr, "accuracy") {
		t.Errorf("Output missing accuracy field")
	}
	if !contains(contentStr, "results") {
		t.Errorf("Output missing results field")
	}
}

func TestWriteConsoleOutput(t *testing.T) {
	// Create test results
	results := []models.PreloadResult{
		{
			File:     "test.go",
			Line:     10,
			Relation: "User",
			Model:    "Order",
			Variable: "orders",
			FindLine: 10,
			Status:   "correct",
		},
		{
			File:     "test.go",
			Line:     15,
			Relation: "Profile",
			Model:    "Unknown",
			Status:   "unknown",
		},
	}

	// Test console output (we can't easily test the actual output, but we can test that it doesn't panic)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Console output panicked: %v", r)
		}
	}()

	WriteConsoleOutput(results)
}

func TestGetStatusEmoji(t *testing.T) {
	testCases := []struct {
		status   string
		expected string
	}{
		{"correct", "✅"},
		{"unknown", "❓"},
		{"error", "❌"},
		{"invalid", "❓"},
		{"", "❓"},
	}

	for _, tc := range testCases {
		result := getStatusEmoji(tc.status)
		if result != tc.expected {
			t.Errorf("For status '%s', expected '%s', got '%s'", tc.status, tc.expected, result)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
