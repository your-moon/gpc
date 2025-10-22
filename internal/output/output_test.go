package output

import (
	"os"
	"testing"

	"github.com/your-moon/gpc/internal/models"
)

func TestWriteStructuredOutput(t *testing.T) {
	tests := []struct {
		name    string
		results []models.PreloadResult
	}{
		{
			name: "Basic results",
			results: []models.PreloadResult{
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
			},
		},
		{
			name:    "Empty results",
			results: []models.PreloadResult{},
		},
		{
			name: "Mixed status results",
			results: []models.PreloadResult{
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
				{
					File:     "test.go",
					Line:     20,
					Relation: "Invalid",
					Model:    "Error",
					Status:   "error",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON output
			testFile := "test_output_" + tt.name + ".json"
			err := WriteStructuredOutput(tt.results, testFile)
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
			requiredFields := []string{"total_preloads", "correct", "unknown", "accuracy", "results"}
			for _, field := range requiredFields {
				if !contains(contentStr, field) {
					t.Errorf("Output missing %s field", field)
				}
			}
		})
	}
}

func TestWriteConsoleOutput(t *testing.T) {
	tests := []struct {
		name    string
		results []models.PreloadResult
	}{
		{
			name: "Basic results",
			results: []models.PreloadResult{
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
			},
		},
		{
			name:    "Empty results",
			results: []models.PreloadResult{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test console output (we can't easily test the actual output, but we can test that it doesn't panic)
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Console output panicked: %v", r)
				}
			}()

			WriteConsoleOutput(tt.results)
		})
	}
}

func TestGetStatusEmoji(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "Correct status",
			status:   "correct",
			expected: "✅",
		},
		{
			name:     "Unknown status",
			status:   "unknown",
			expected: "❓",
		},
		{
			name:     "Error status",
			status:   "error",
			expected: "❌",
		},
		{
			name:     "Invalid status",
			status:   "invalid",
			expected: "❓",
		},
		{
			name:     "Empty status",
			status:   "",
			expected: "❓",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStatusEmoji(tt.status)
			if result != tt.expected {
				t.Errorf("For status '%s', expected '%s', got '%s'", tt.status, tt.expected, result)
			}
		})
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
