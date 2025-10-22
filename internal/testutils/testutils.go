package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/your-moon/gpc/internal/models"
)

// TestFile represents a test file with its content
type TestFile struct {
	Name    string
	Content string
}

// TestCase represents a test case with expected results
type TestCase struct {
	Name        string
	Files       []TestFile
	Expected    ExpectedResults
	Description string
}

// ExpectedResults represents the expected results for a test case
type ExpectedResults struct {
	PreloadCalls    []ExpectedPreloadCall
	GormCalls       []ExpectedGormCall
	VariableTypes   []ExpectedVariableType
	AnalysisResults []ExpectedAnalysisResult
}

// ExpectedPreloadCall represents an expected preload call
type ExpectedPreloadCall struct {
	Relation string
	Line     int
	Scope    string
}

// ExpectedGormCall represents an expected GORM call
type ExpectedGormCall struct {
	Method string
	Line   int
	Scope  string
}

// ExpectedVariableType represents an expected variable type
type ExpectedVariableType struct {
	VarName   string
	TypeName  string
	ModelName string
	Scope     string
}

// ExpectedAnalysisResult represents an expected analysis result
type ExpectedAnalysisResult struct {
	Relation string
	Model    string
	Variable string
	Status   string
	FindLine int
}

// CreateTestFiles creates temporary test files and returns cleanup function and file paths
func CreateTestFiles(t *testing.T, files []TestFile) (func(), []string) {
	tempDir := t.TempDir()
	var filePaths []string

	for _, file := range files {
		filePath := filepath.Join(tempDir, file.Name)
		err := os.WriteFile(filePath, []byte(file.Content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.Name, err)
		}
		filePaths = append(filePaths, filePath)
	}

	return func() {
		os.RemoveAll(tempDir)
	}, filePaths
}

// AssertPreloadCalls compares actual preload calls with expected ones
func AssertPreloadCalls(t *testing.T, actual []models.PreloadCall, expected []ExpectedPreloadCall) {
	if len(actual) != len(expected) {
		t.Errorf("Expected %d preload calls, got %d", len(expected), len(actual))
		return
	}

	// Create a map for easier comparison
	expectedMap := make(map[string]int)
	for _, exp := range expected {
		key := exp.Relation + ":" + exp.Scope
		expectedMap[key]++
	}

	actualMap := make(map[string]int)
	for _, act := range actual {
		key := act.Relation + ":" + act.Scope
		actualMap[key]++
	}

	for key, expectedCount := range expectedMap {
		actualCount := actualMap[key]
		if actualCount != expectedCount {
			t.Errorf("Expected %d calls for %s, got %d", expectedCount, key, actualCount)
		}
	}
}

// AssertGormCalls compares actual GORM calls with expected ones
func AssertGormCalls(t *testing.T, actual []models.GormCall, expected []ExpectedGormCall) {
	if len(actual) != len(expected) {
		t.Errorf("Expected %d GORM calls, got %d", len(expected), len(actual))
		return
	}

	// Create a map for easier comparison
	expectedMap := make(map[string]int)
	for _, exp := range expected {
		key := exp.Method + ":" + exp.Scope
		expectedMap[key]++
	}

	actualMap := make(map[string]int)
	for _, act := range actual {
		key := act.Method + ":" + act.Scope
		actualMap[key]++
	}

	for key, expectedCount := range expectedMap {
		actualCount := actualMap[key]
		if actualCount != expectedCount {
			t.Errorf("Expected %d calls for %s, got %d", expectedCount, key, actualCount)
		}
	}
}

// AssertVariableTypes compares actual variable types with expected ones
func AssertVariableTypes(t *testing.T, actual []models.VariableType, expected []ExpectedVariableType) {
	if len(actual) != len(expected) {
		t.Errorf("Expected %d variable types, got %d", len(expected), len(actual))
		return
	}

	// Create a map for easier comparison
	expectedMap := make(map[string]ExpectedVariableType)
	for _, exp := range expected {
		key := exp.VarName + ":" + exp.Scope
		expectedMap[key] = exp
	}

	actualMap := make(map[string]models.VariableType)
	for _, act := range actual {
		key := act.VarName + ":" + act.Scope
		actualMap[key] = act
	}

	for key, expected := range expectedMap {
		actual, exists := actualMap[key]
		if !exists {
			t.Errorf("Missing variable type for %s", key)
			continue
		}

		if actual.TypeName != expected.TypeName {
			t.Errorf("For %s: expected type %s, got %s", key, expected.TypeName, actual.TypeName)
		}
		if actual.ModelName != expected.ModelName {
			t.Errorf("For %s: expected model %s, got %s", key, expected.ModelName, actual.ModelName)
		}
	}
}

// AssertAnalysisResults compares actual analysis results with expected ones
func AssertAnalysisResults(t *testing.T, actual []models.PreloadResult, expected []ExpectedAnalysisResult) {
	if len(actual) != len(expected) {
		t.Errorf("Expected %d analysis results, got %d", len(expected), len(actual))
		return
	}

	// Create a map for easier comparison
	expectedMap := make(map[string]ExpectedAnalysisResult)
	for _, exp := range expected {
		key := exp.Relation
		expectedMap[key] = exp
	}

	actualMap := make(map[string]models.PreloadResult)
	for _, act := range actual {
		key := act.Relation
		actualMap[key] = act
	}

	for key, expected := range expectedMap {
		actual, exists := actualMap[key]
		if !exists {
			t.Errorf("Missing analysis result for relation %s", key)
			continue
		}

		if actual.Model != expected.Model {
			t.Errorf("For %s: expected model %s, got %s", key, expected.Model, actual.Model)
		}
		if actual.Variable != expected.Variable {
			t.Errorf("For %s: expected variable %s, got %s", key, expected.Variable, actual.Variable)
		}
		if actual.Status != expected.Status {
			t.Errorf("For %s: expected status %s, got %s", key, expected.Status, actual.Status)
		}
		if actual.FindLine != expected.FindLine {
			t.Errorf("For %s: expected find line %d, got %d", key, expected.FindLine, actual.FindLine)
		}
	}
}

// Common test templates
var (
	BasicGoFile = TestFile{
		Name: "basic.go",
		Content: `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}

func TestBasic() {
	var db *gorm.DB
	var orders []Order
	db.Preload("User").Find(&orders)
}`,
	}

	MultiPreloadFile = TestFile{
		Name: "multi.go",
		Content: `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}

func TestMulti() {
	var db *gorm.DB
	var orders []Order
	db.Preload("User").Preload("User.Profile").Find(&orders)
}`,
	}

	AssignmentFile = TestFile{
		Name: "assignment.go",
		Content: `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

func TestAssignment() {
	var db *gorm.DB
	orderDB := db.Preload("User")
	userDB := db.Preload("Profile")
	
	var orders []Order
	orderDB.Find(&orders)
	userDB.Find(&orders)
}`,
	}
)
