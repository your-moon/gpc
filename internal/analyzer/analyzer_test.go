package analyzer

import (
	"testing"

	"github.com/your-moon/gpc/internal/models"
	"github.com/your-moon/gpc/internal/testutils"
)

func TestAnalyzePreloads(t *testing.T) {
	tests := []struct {
		name           string
		preloadCalls   []models.PreloadCall
		gormCalls      []models.GormCall
		varAssignments []models.VariableAssignment
		variableTypes  []models.VariableType
		expected       []testutils.ExpectedAnalysisResult
	}{
		{
			name: "Basic analysis",
			preloadCalls: []models.PreloadCall{
				{
					File:        "test.go",
					Line:        10,
					Relation:    "User",
					Scope:       "TestFunc",
					LineContent: "db.Preload(\"User\").Find(&orders)",
				},
				{
					File:        "test.go",
					Line:        15,
					Relation:    "Profile",
					Scope:       "TestFunc",
					LineContent: "db.Preload(\"Profile\").First(&user)",
				},
			},
			gormCalls: []models.GormCall{
				{
					File:        "test.go",
					Line:        10,
					Method:      "Find",
					Scope:       "TestFunc",
					LineContent: "db.Preload(\"User\").Find(&orders)",
				},
				{
					File:        "test.go",
					Line:        15,
					Method:      "First",
					Scope:       "TestFunc",
					LineContent: "db.Preload(\"Profile\").First(&user)",
				},
			},
			varAssignments: []models.VariableAssignment{
				{
					VarName:     "orders",
					AssignedTo:  "[]Order",
					Line:        8,
					File:        "test.go",
					Scope:       "TestFunc",
					LineContent: "var orders []Order",
				},
				{
					VarName:     "user",
					AssignedTo:  "User",
					Line:        13,
					File:        "test.go",
					Scope:       "TestFunc",
					LineContent: "var user User",
				},
			},
			variableTypes: []models.VariableType{
				{
					VarName:   "orders",
					TypeName:  "[]Order",
					ModelName: "Order",
					Scope:     "TestFunc",
					File:      "test.go",
					Line:      8,
				},
				{
					VarName:   "user",
					TypeName:  "User",
					ModelName: "User",
					Scope:     "TestFunc",
					File:      "test.go",
					Line:      13,
				},
			},
			expected: []testutils.ExpectedAnalysisResult{
				{Relation: "User", Model: "Order", Variable: "orders", Status: "correct", FindLine: 10},
				{Relation: "Profile", Model: "User", Variable: "user", Status: "correct", FindLine: 15},
			},
		},
		{
			name: "Unknown model case",
			preloadCalls: []models.PreloadCall{
				{
					File:        "test.go",
					Line:        10,
					Relation:    "UnknownRelation",
					Scope:       "TestFunc",
					LineContent: "db.Preload(\"UnknownRelation\").Find(&orders)",
				},
			},
			gormCalls: []models.GormCall{
				{
					File:        "test.go",
					Line:        10,
					Method:      "Find",
					Scope:       "TestFunc",
					LineContent: "db.Preload(\"UnknownRelation\").Find(&orders)",
				},
			},
			varAssignments: []models.VariableAssignment{},
			variableTypes: []models.VariableType{
				{
					VarName:   "orders",
					TypeName:  "[]Order",
					ModelName: "Order",
					Scope:     "TestFunc",
					File:      "test.go",
					Line:      8,
				},
			},
			expected: []testutils.ExpectedAnalysisResult{
				{Relation: "UnknownRelation", Model: "Order", Variable: "orders", Status: "correct", FindLine: 10},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test analysis
			results := AnalyzePreloads(tt.preloadCalls, tt.gormCalls, tt.varAssignments, tt.variableTypes)

			// Verify results
			testutils.AssertAnalysisResults(t, results, tt.expected)
		})
	}
}

func TestFindVariableAndFindCall(t *testing.T) {
	tests := []struct {
		name         string
		preloadCall  models.PreloadCall
		gormCalls    []models.GormCall
		varMap       map[string]models.VariableAssignment
		expectedVar  string
		expectedCall string
	}{
		{
			name: "Same line method chain",
			preloadCall: models.PreloadCall{
				File:        "test.go",
				Line:        10,
				Relation:    "User",
				Scope:       "TestFunc",
				LineContent: "db.Preload(\"User\").Find(&orders)",
			},
			gormCalls: []models.GormCall{
				{
					File:        "test.go",
					Line:        10,
					Method:      "Find",
					Scope:       "TestFunc",
					LineContent: "db.Preload(\"User\").Find(&orders)",
				},
			},
			varMap:       map[string]models.VariableAssignment{},
			expectedVar:  "orders",
			expectedCall: "line 10",
		},
		{
			name: "Multi-line method chain",
			preloadCall: models.PreloadCall{
				File:        "test.go",
				Line:        10,
				Relation:    "User",
				Scope:       "TestFunc",
				LineContent: "db. Preload(\"User\"). Preload(\"User.Profile\"). Where(\"id = ?\", 1). Find(&orders)",
			},
			gormCalls: []models.GormCall{
				{
					File:        "test.go",
					Line:        10,
					Method:      "Find",
					Scope:       "TestFunc",
					LineContent: "db. Preload(\"User\"). Preload(\"User.Profile\"). Where(\"id = ?\", 1). Find(&orders)",
				},
			},
			varMap:       map[string]models.VariableAssignment{},
			expectedVar:  "orders",
			expectedCall: "line 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			varName, findCall := findVariableAndFindCall(tt.preloadCall, tt.gormCalls, tt.varMap)

			if varName != tt.expectedVar {
				t.Errorf("Expected variable name '%s', got '%s'", tt.expectedVar, varName)
			}
			if findCall != tt.expectedCall {
				t.Errorf("Expected find call '%s', got '%s'", tt.expectedCall, findCall)
			}
		})
	}
}

func TestExtractVariableNameFromFindCall(t *testing.T) {
	tests := []struct {
		name        string
		lineContent string
		expected    string
	}{
		{
			name:        "Find call",
			lineContent: "db.Preload(\"User\").Find(&orders)",
			expected:    "orders",
		},
		{
			name:        "First call",
			lineContent: "db.First(&user)",
			expected:    "user",
		},
		{
			name:        "FirstOrCreate call",
			lineContent: "db.FirstOrCreate(&currentUser)",
			expected:    "currentUser",
		},
		{
			name:        "Find with parameters",
			lineContent: "db.Find(&items, id)",
			expected:    "items",
		},
		{
			name:        "First with parameters",
			lineContent: "db.First(&user, \"id = ?\", 1)",
			expected:    "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVariableNameFromFindCall(tt.lineContent)
			if result != tt.expected {
				t.Errorf("For line '%s', expected '%s', got '%s'", tt.lineContent, tt.expected, result)
			}
		})
	}
}

func TestExtractVariableFromPreloadCall(t *testing.T) {
	tests := []struct {
		name        string
		lineContent string
		expected    string
	}{
		{
			name:        "Assignment with :=",
			lineContent: "orderDB := db.Preload(\"User\")",
			expected:    "orderDB",
		},
		{
			name:        "Assignment with =",
			lineContent: "userDB = db.Preload(\"Profile\")",
			expected:    "userDB",
		},
		{
			name:        "No assignment",
			lineContent: "db.Preload(\"User\").Find(&orders)",
			expected:    "",
		},
		{
			name:        "Var declaration",
			lineContent: "var orderDB = db.Preload(\"User\")",
			expected:    "orderDB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVariableFromPreloadCall(tt.lineContent)
			if result != tt.expected {
				t.Errorf("For line '%s', expected '%s', got '%s'", tt.lineContent, tt.expected, result)
			}
		})
	}
}

func TestInferModelFromVariableName(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		expected string
	}{
		{
			name:     "Plural to singular",
			varName:  "orders",
			expected: "Order",
		},
		{
			name:     "Plural to singular",
			varName:  "users",
			expected: "User",
		},
		{
			name:     "Compound name",
			varName:  "orderList",
			expected: "Order",
		},
		{
			name:     "Compound name with multiple words",
			varName:  "userItems",
			expected: "UserItem",
		},
		{
			name:     "Single word",
			varName:  "order",
			expected: "Order",
		},
		{
			name:     "Single word",
			varName:  "user",
			expected: "User",
		},
		{
			name:     "Empty string",
			varName:  "",
			expected: "",
		},
		{
			name:     "Single character",
			varName:  "a",
			expected: "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferModelFromVariableName(tt.varName)
			if result != tt.expected {
				t.Errorf("For variable '%s', expected '%s', got '%s'", tt.varName, tt.expected, result)
			}
		})
	}
}
