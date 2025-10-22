package analyzer

import (
	"testing"

	"github.com/your-moon/gpc/internal/models"
)

func TestAnalyzePreloads(t *testing.T) {
	// Create test data
	preloadCalls := []models.PreloadCall{
		{
			File:     "test.go",
			Line:     10,
			Relation: "User",
			Scope:    "TestFunc",
			LineContent: "db.Preload(\"User\").Find(&orders)",
		},
		{
			File:     "test.go",
			Line:     15,
			Relation: "Profile",
			Scope:    "TestFunc",
			LineContent: "db.Preload(\"Profile\").First(&user)",
		},
	}

	gormCalls := []models.GormCall{
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
	}

	varAssignments := []models.VariableAssignment{
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
	}

	variableTypes := []models.VariableType{
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
	}

	// Test analysis
	results := AnalyzePreloads(preloadCalls, gormCalls, varAssignments, variableTypes)

	// Verify results
	expectedResults := []struct {
		relation string
		model    string
		variable string
		status   string
		findLine int
	}{
		{"User", "Order", "orders", "correct", 10},
		{"Profile", "User", "user", "correct", 15},
	}

	if len(results) != len(expectedResults) {
		t.Errorf("Expected %d results, got %d", len(expectedResults), len(results))
	}

	for i, expected := range expectedResults {
		if i >= len(results) {
			t.Errorf("Missing result for relation %s", expected.relation)
			continue
		}

		result := results[i]
		if result.Relation != expected.relation {
			t.Errorf("Expected relation %s, got %s", expected.relation, result.Relation)
		}
		if result.Model != expected.model {
			t.Errorf("Expected model %s, got %s", expected.model, result.Model)
		}
		if result.Variable != expected.variable {
			t.Errorf("Expected variable %s, got %s", expected.variable, result.Variable)
		}
		if result.Status != expected.status {
			t.Errorf("Expected status %s, got %s", expected.status, result.Status)
		}
		if result.FindLine != expected.findLine {
			t.Errorf("Expected find line %d, got %d", expected.findLine, result.FindLine)
		}
	}
}

func TestFindVariableAndFindCall(t *testing.T) {
	// Test case 1: Same line method chain
	preloadCall := models.PreloadCall{
		File:        "test.go",
		Line:        10,
		Relation:    "User",
		Scope:       "TestFunc",
		LineContent: "db.Preload(\"User\").Find(&orders)",
	}

	gormCalls := []models.GormCall{
		{
			File:        "test.go",
			Line:        10,
			Method:      "Find",
			Scope:       "TestFunc",
			LineContent: "db.Preload(\"User\").Find(&orders)",
		},
	}

	varMap := map[string]models.VariableAssignment{}

	varName, findCall := findVariableAndFindCall(preloadCall, gormCalls, varMap)

	if varName != "orders" {
		t.Errorf("Expected variable name 'orders', got '%s'", varName)
	}
	if findCall != "line 10" {
		t.Errorf("Expected find call 'line 10', got '%s'", findCall)
	}
}

func TestExtractVariableNameFromFindCall(t *testing.T) {
	testCases := []struct {
		lineContent string
		expected    string
	}{
		{"db.Preload(\"User\").Find(&orders)", "orders"},
		{"db.First(&user)", "user"},
		{"db.FirstOrCreate(&currentUser)", "currentUser"},
		{"db.Find(&items, id)", "items"},
		{"db.First(&user, \"id = ?\", 1)", "user"},
	}

	for _, tc := range testCases {
		result := extractVariableNameFromFindCall(tc.lineContent)
		if result != tc.expected {
			t.Errorf("For line '%s', expected '%s', got '%s'", tc.lineContent, tc.expected, result)
		}
	}
}

func TestExtractVariableFromPreloadCall(t *testing.T) {
	testCases := []struct {
		lineContent string
		expected    string
	}{
		{"orderDB := db.Preload(\"User\")", "orderDB"},
		{"userDB = db.Preload(\"Profile\")", "userDB"},
		{"db.Preload(\"User\").Find(&orders)", ""}, // No assignment
		{"var orderDB = db.Preload(\"User\")", "orderDB"},
	}

	for _, tc := range testCases {
		result := extractVariableFromPreloadCall(tc.lineContent)
		if result != tc.expected {
			t.Errorf("For line '%s', expected '%s', got '%s'", tc.lineContent, tc.expected, result)
		}
	}
}

func TestInferModelFromVariableName(t *testing.T) {
	testCases := []struct {
		varName  string
		expected string
	}{
		{"orders", "Order"},
		{"users", "User"},
		{"orderList", "Order"},
		{"userItems", "UserItem"}, // Fixed: should be UserItem, not User
		{"order", "Order"},
		{"user", "User"},
		{"", ""},
		{"a", "A"},
	}

	for _, tc := range testCases {
		result := inferModelFromVariableName(tc.varName)
		if result != tc.expected {
			t.Errorf("For variable '%s', expected '%s', got '%s'", tc.varName, tc.expected, result)
		}
	}
}
