package validator

import (
	"testing"

	"github.com/your-moon/gpc/internal/models"
	"github.com/your-moon/gpc/internal/testutils"
)

func TestValidatePreloadRelations(t *testing.T) {
	tests := []struct {
		name       string
		results    []models.PreloadResult
		allStructs map[string]models.StructInfo
		expected   []testutils.ExpectedAnalysisResult
	}{
		{
			name: "Valid relations",
			results: []models.PreloadResult{
				{
					File:     "test.go",
					Line:     10,
					Relation: "User",
					Model:    "Order",
					Status:   "correct",
				},
				{
					File:     "test.go",
					Line:     15,
					Relation: "Profile",
					Model:    "User",
					Status:   "correct",
				},
			},
			allStructs: map[string]models.StructInfo{
				"Order": {
					Name: "Order",
					Fields: map[string]string{
						"ID":     "int64",
						"UserID": "int64",
						"User":   "User",
					},
				},
				"User": {
					Name: "User",
					Fields: map[string]string{
						"ID":      "int64",
						"Name":    "string",
						"Profile": "Profile",
					},
				},
			},
			expected: []testutils.ExpectedAnalysisResult{
				{Relation: "User", Model: "Order", Status: "correct"},
				{Relation: "Profile", Model: "User", Status: "correct"},
			},
		},
		{
			name: "Invalid relations",
			results: []models.PreloadResult{
				{
					File:     "test.go",
					Line:     10,
					Relation: "InvalidField",
					Model:    "Order",
					Status:   "correct",
				},
				{
					File:     "test.go",
					Line:     15,
					Relation: "NonExistent",
					Model:    "User",
					Status:   "correct",
				},
			},
			allStructs: map[string]models.StructInfo{
				"Order": {
					Name: "Order",
					Fields: map[string]string{
						"ID":     "int64",
						"UserID": "int64",
						"User":   "User",
					},
				},
				"User": {
					Name: "User",
					Fields: map[string]string{
						"ID":   "int64",
						"Name": "string",
					},
				},
			},
			expected: []testutils.ExpectedAnalysisResult{
				{Relation: "InvalidField", Model: "Order", Status: "error"},
				{Relation: "NonExistent", Model: "User", Status: "error"},
			},
		},
		{
			name: "Unknown models",
			results: []models.PreloadResult{
				{
					File:     "test.go",
					Line:     10,
					Relation: "User",
					Model:    "UnknownModel",
					Status:   "correct",
				},
			},
			allStructs: map[string]models.StructInfo{
				"Order": {
					Name: "Order",
					Fields: map[string]string{
						"ID":     "int64",
						"UserID": "int64",
						"User":   "User",
					},
				},
			},
			expected: []testutils.ExpectedAnalysisResult{
				{Relation: "User", Model: "UnknownModel", Status: "unknown"},
			},
		},
		{
			name: "Package-qualified models",
			results: []models.PreloadResult{
				{
					File:     "test.go",
					Line:     10,
					Relation: "User",
					Model:    "databases.Order",
					Status:   "correct",
				},
			},
			allStructs: map[string]models.StructInfo{
				"Order": {
					Name: "Order",
					Fields: map[string]string{
						"ID":     "int64",
						"UserID": "int64",
						"User":   "User",
					},
				},
			},
			expected: []testutils.ExpectedAnalysisResult{
				{Relation: "User", Model: "databases.Order", Status: "correct"},
			},
		},
		{
			name: "Nested relations",
			results: []models.PreloadResult{
				{
					File:     "test.go",
					Line:     10,
					Relation: "User.Profile",
					Model:    "Order",
					Status:   "correct",
				},
			},
			allStructs: map[string]models.StructInfo{
				"Order": {
					Name: "Order",
					Fields: map[string]string{
						"ID":     "int64",
						"UserID": "int64",
						"User":   "User",
					},
				},
			},
			expected: []testutils.ExpectedAnalysisResult{
				{Relation: "User.Profile", Model: "Order", Status: "correct"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation
			validatedResults := ValidatePreloadRelations(tt.results, tt.allStructs)

			// Verify results
			testutils.AssertAnalysisResults(t, validatedResults, tt.expected)
		})
	}
}

func TestExtractBaseModelName(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		expected  string
	}{
		{
			name:      "Simple model name",
			modelName: "User",
			expected:  "User",
		},
		{
			name:      "Package-qualified model",
			modelName: "databases.User",
			expected:  "User",
		},
		{
			name:      "Nested package model",
			modelName: "models.database.User",
			expected:  "User",
		},
		{
			name:      "Empty string",
			modelName: "",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBaseModelName(tt.modelName)
			if result != tt.expected {
				t.Errorf("For model '%s', expected '%s', got '%s'", tt.modelName, tt.expected, result)
			}
		})
	}
}

func TestValidateRelationInStruct(t *testing.T) {
	structInfo := models.StructInfo{
		Name: "Order",
		Fields: map[string]string{
			"ID":     "int64",
			"UserID": "int64",
			"User":   "User",
			"Items":  "[]Item",
		},
	}

	tests := []struct {
		name     string
		relation string
		expected bool
	}{
		{
			name:     "Valid simple relation",
			relation: "User",
			expected: true,
		},
		{
			name:     "Valid array relation",
			relation: "Items",
			expected: true,
		},
		{
			name:     "Invalid relation",
			relation: "NonExistent",
			expected: false,
		},
		{
			name:     "Nested relation (first part valid)",
			relation: "User.Profile",
			expected: true,
		},
		{
			name:     "Nested relation (first part invalid)",
			relation: "NonExistent.Profile",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateRelationInStruct(tt.relation, structInfo)
			if result != tt.expected {
				t.Errorf("For relation '%s', expected %v, got %v", tt.relation, tt.expected, result)
			}
		})
	}
}

func TestGetStructStatistics(t *testing.T) {
	allStructs := map[string]models.StructInfo{
		"User": {
			Name: "User",
			Fields: map[string]string{
				"ID":   "int64",
				"Name": "string",
			},
		},
		"Order": {
			Name: "Order",
			Fields: map[string]string{
				"ID":     "int64",
				"UserID": "int64",
				"User":   "User",
			},
		},
	}

	stats := GetStructStatistics(allStructs)

	expectedStats := map[string]interface{}{
		"total_structs":         2,
		"total_fields":          5,
		"avg_fields_per_struct": 2.5,
	}

	for key, expectedValue := range expectedStats {
		actualValue, exists := stats[key]
		if !exists {
			t.Errorf("Missing statistic: %s", key)
			continue
		}

		if actualValue != expectedValue {
			t.Errorf("For statistic %s, expected %v, got %v", key, expectedValue, actualValue)
		}
	}
}
