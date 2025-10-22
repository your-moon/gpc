package parser

import (
	"os"
	"testing"
)

func TestFindPreloadCalls(t *testing.T) {
	// Create a test file content
	testFile := "test_preload.go"
	testContent := `package main

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

func TestPreloads() {
	var db *gorm.DB
	
	// Simple preload
	var orders []Order
	db.Preload("User").Find(&orders)
	
	// Multiple preloads
	var order Order
	db.Preload("User").Preload("Items").First(&order)
	
	// Nested preload
	db.Preload("User.Profile").Find(&orders)
}`

	// Write test file
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test parsing
	preloadCalls := FindPreloadCalls(testFile)

	// Verify results - check that we have the expected relations (order may vary)
	expectedRelations := map[string]int{
		"User":         2, // Should appear twice
		"Items":        1,
		"User.Profile": 1,
	}

	actualRelations := make(map[string]int)
	for _, call := range preloadCalls {
		actualRelations[call.Relation]++
	}

	if len(preloadCalls) != 4 {
		t.Errorf("Expected 4 preload calls, got %d", len(preloadCalls))
	}

	for relation, expectedCount := range expectedRelations {
		actualCount := actualRelations[relation]
		if actualCount != expectedCount {
			t.Errorf("Expected relation '%s' to appear %d times, got %d", relation, expectedCount, actualCount)
		}
	}
}

func TestFindGormCalls(t *testing.T) {
	// Create a test file content
	testFile := "test_gorm.go"
	testContent := `package main

import "gorm.io/gorm"

func TestGormCalls() {
	var db *gorm.DB
	
	// Find call
	var users []User
	db.Find(&users)
	
	// First call
	var user User
	db.First(&user)
	
	// FirstOrCreate call
	db.FirstOrCreate(&user)
}`

	// Write test file
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test parsing
	gormCalls := FindGormCalls(testFile)

	// Verify results - adjust expected line numbers based on actual content
	expectedCalls := []struct {
		method string
	}{
		{"Find"},
		{"First"},
		{"FirstOrCreate"},
	}

	if len(gormCalls) != len(expectedCalls) {
		t.Errorf("Expected %d gorm calls, got %d", len(expectedCalls), len(gormCalls))
	}

	for i, expected := range expectedCalls {
		if i >= len(gormCalls) {
			t.Errorf("Missing gorm call for method %s", expected.method)
			continue
		}

		call := gormCalls[i]
		if call.Method != expected.method {
			t.Errorf("Expected method %s, got %s", expected.method, call.Method)
		}
	}
}

func TestFindVariableTypes(t *testing.T) {
	// Create a test file content
	testFile := "test_vars.go"
	testContent := `package main

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

func TestVariables() {
	// Var declaration
	var users []User
	var order Order
	
	// Short declaration
	orders := []Order{}
	currentUser := User{}
	
	// With assignment
	var db *gorm.DB
}`

	// Write test file
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test parsing
	variableTypes := FindVariableTypes(testFile)

	// Verify results
	expectedVars := []struct {
		name     string
		typeName string
		model    string
	}{
		{"users", "[]User", "User"},
		{"order", "Order", "Order"},
		{"orders", "[]Order", "Order"},
		{"currentUser", "User", "User"},
		{"db", "*gorm.DB", "DB"},
	}

	if len(variableTypes) != len(expectedVars) {
		t.Errorf("Expected %d variable types, got %d", len(expectedVars), len(variableTypes))
	}

	for i, expected := range expectedVars {
		if i >= len(variableTypes) {
			t.Errorf("Missing variable %s", expected.name)
			continue
		}

		varType := variableTypes[i]
		if varType.VarName != expected.name {
			t.Errorf("Expected var name %s, got %s", expected.name, varType.VarName)
		}
		if varType.TypeName != expected.typeName {
			t.Errorf("Expected type %s, got %s", expected.typeName, varType.TypeName)
		}
		if varType.ModelName != expected.model {
			t.Errorf("Expected model %s, got %s", expected.model, varType.ModelName)
		}
	}
}

func TestParseStructsFromFile(t *testing.T) {
	// Create a test file content
	testFile := "test_structs.go"
	testContent := `package main

type User struct {
	ID   int64
	Name string
	Age  int
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}`

	// Write test file
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test parsing
	structs := ParseStructsFromFile(testFile)

	// Verify results
	expectedStructs := []struct {
		name   string
		fields []string
	}{
		{"User", []string{"ID", "Name", "Age"}},
		{"Order", []string{"ID", "UserID", "User"}},
	}

	if len(structs) != len(expectedStructs) {
		t.Errorf("Expected %d structs, got %d", len(expectedStructs), len(structs))
	}

	for _, expected := range expectedStructs {
		structInfo, exists := structs[expected.name]
		if !exists {
			t.Errorf("Missing struct %s", expected.name)
			continue
		}

		if structInfo.Name != expected.name {
			t.Errorf("Expected struct name %s, got %s", expected.name, structInfo.Name)
		}

		for _, field := range expected.fields {
			if _, exists := structInfo.Fields[field]; !exists {
				t.Errorf("Missing field %s in struct %s", field, expected.name)
			}
		}
	}
}
