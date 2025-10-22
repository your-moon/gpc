package parser

import (
	"testing"

	"github.com/your-moon/gpc/internal/testutils"
)

func TestFindPreloadCalls(t *testing.T) {
	tests := []struct {
		name     string
		files    []testutils.TestFile
		expected []testutils.ExpectedPreloadCall
	}{
		{
			name: "Basic preload calls",
			files: []testutils.TestFile{
				{
					Name: "test.go",
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
}`,
				},
			},
			expected: []testutils.ExpectedPreloadCall{
				{Relation: "User", Scope: "TestPreloads"},
				{Relation: "User", Scope: "TestPreloads"},
				{Relation: "Items", Scope: "TestPreloads"},
				{Relation: "User.Profile", Scope: "TestPreloads"},
			},
		},
		{
			name: "Multi-line preload calls",
			files: []testutils.TestFile{
				{
					Name: "multiline.go",
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

func TestMultiLine() {
	var db *gorm.DB
	var orders []Order
	
	db.
		Preload("User").
		Preload("User.Profile").
		Where("id = ?", 1).
		Find(&orders)
}`,
				},
			},
			expected: []testutils.ExpectedPreloadCall{
				{Relation: "User", Scope: "TestMultiLine"},
				{Relation: "User.Profile", Scope: "TestMultiLine"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup, filePaths := testutils.CreateTestFiles(t, tt.files)
			defer cleanup()

			// Test parsing
			preloadCalls := FindPreloadCalls(filePaths[0])

			// Verify results
			testutils.AssertPreloadCalls(t, preloadCalls, tt.expected)
		})
	}
}

func TestFindGormCalls(t *testing.T) {
	tests := []struct {
		name     string
		files    []testutils.TestFile
		expected []testutils.ExpectedGormCall
	}{
		{
			name: "Basic GORM calls",
			files: []testutils.TestFile{
				{
					Name: "test.go",
					Content: `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

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
}`,
				},
			},
			expected: []testutils.ExpectedGormCall{
				{Method: "Find", Scope: "TestGormCalls"},
				{Method: "First", Scope: "TestGormCalls"},
				{Method: "FirstOrCreate", Scope: "TestGormCalls"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup, filePaths := testutils.CreateTestFiles(t, tt.files)
			defer cleanup()

			// Test parsing
			gormCalls := FindGormCalls(filePaths[0])

			// Verify results
			testutils.AssertGormCalls(t, gormCalls, tt.expected)
		})
	}
}

func TestFindVariableTypes(t *testing.T) {
	tests := []struct {
		name     string
		files    []testutils.TestFile
		expected []testutils.ExpectedVariableType
	}{
		{
			name: "Variable type detection",
			files: []testutils.TestFile{
				{
					Name: "test.go",
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

func TestVariables() {
	// Var declaration
	var users []User
	var order Order
	
	// Short declaration
	orders := []Order{}
	currentUser := User{}
	
	// With assignment
	var db *gorm.DB
}`,
				},
			},
			expected: []testutils.ExpectedVariableType{
				{VarName: "users", TypeName: "[]User", ModelName: "User", Scope: "TestVariables"},
				{VarName: "order", TypeName: "Order", ModelName: "Order", Scope: "TestVariables"},
				{VarName: "orders", TypeName: "[]Order", ModelName: "Order", Scope: "TestVariables"},
				{VarName: "currentUser", TypeName: "User", ModelName: "User", Scope: "TestVariables"},
				{VarName: "db", TypeName: "*gorm.DB", ModelName: "DB", Scope: "TestVariables"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup, filePaths := testutils.CreateTestFiles(t, tt.files)
			defer cleanup()

			// Test parsing
			variableTypes := FindVariableTypes(filePaths[0])

			// Verify results
			testutils.AssertVariableTypes(t, variableTypes, tt.expected)
		})
	}
}

func TestParseStructsFromFile(t *testing.T) {
	tests := []struct {
		name     string
		files    []testutils.TestFile
		expected map[string][]string
	}{
		{
			name: "Struct parsing",
			files: []testutils.TestFile{
				{
					Name: "test.go",
					Content: `package main

type User struct {
	ID   int64
	Name string
	Age  int
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}`,
				},
			},
			expected: map[string][]string{
				"User":  {"ID", "Name", "Age"},
				"Order": {"ID", "UserID", "User"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup, filePaths := testutils.CreateTestFiles(t, tt.files)
			defer cleanup()

			// Test parsing
			structs := ParseStructsFromFile(filePaths[0])

			// Verify results
			if len(structs) != len(tt.expected) {
				t.Errorf("Expected %d structs, got %d", len(tt.expected), len(structs))
				return
			}

			for structName, expectedFields := range tt.expected {
				structInfo, exists := structs[structName]
				if !exists {
					t.Errorf("Missing struct %s", structName)
					continue
				}

				if structInfo.Name != structName {
					t.Errorf("Expected struct name %s, got %s", structName, structInfo.Name)
				}

				for _, field := range expectedFields {
					if _, exists := structInfo.Fields[field]; !exists {
						t.Errorf("Missing field %s in struct %s", field, structName)
					}
				}
			}
		})
	}
}
