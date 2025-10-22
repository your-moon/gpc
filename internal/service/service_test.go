package service

import (
	"os"
	"testing"
)

func TestNewService(t *testing.T) {
	svc := NewService("console", "test.json")

	if svc.outputFormat != "console" {
		t.Errorf("Expected output format 'console', got '%s'", svc.outputFormat)
	}

	if svc.outputFile != "test.json" {
		t.Errorf("Expected output file 'test.json', got '%s'", svc.outputFile)
	}
}

func TestAnalyzeTarget_File(t *testing.T) {
	// Create a test file
	testFile := "test_service.go"
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
	var orders []Order
	db.Preload("User").Find(&orders)
}`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test service
	svc := NewService("json", "test_service_output.json")
	err = svc.AnalyzeTarget(testFile)
	if err != nil {
		t.Fatalf("Service analysis failed: %v", err)
	}
	defer os.Remove("test_service_output.json")

	// Verify output file was created
	if _, err := os.Stat("test_service_output.json"); os.IsNotExist(err) {
		t.Errorf("Output file was not created")
	}
}

func TestAnalyzeTarget_Directory(t *testing.T) {
	// Create a test directory with Go files
	testDir := "test_service_dir"
	err := os.Mkdir(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create test files
	testFiles := []struct {
		name    string
		content string
	}{
		{
			"test1.go",
			`package main
import "gorm.io/gorm"
type User struct { ID int64; Name string }
func Test1() { var db *gorm.DB; var users []User; db.Preload("Profile").Find(&users) }`,
		},
		{
			"test2.go",
			`package main
import "gorm.io/gorm"
type Order struct { ID int64; UserID int64 }
func Test2() { var db *gorm.DB; var orders []Order; db.Preload("User").Find(&orders) }`,
		},
	}

	for _, tf := range testFiles {
		filePath := testDir + "/" + tf.name
		err := os.WriteFile(filePath, []byte(tf.content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file %s: %v", tf.name, err)
		}
	}

	// Test service
	svc := NewService("json", "test_dir_output.json")
	err = svc.AnalyzeTarget(testDir)
	if err != nil {
		t.Fatalf("Service analysis failed: %v", err)
	}
	defer os.Remove("test_dir_output.json")

	// Verify output file was created
	if _, err := os.Stat("test_dir_output.json"); os.IsNotExist(err) {
		t.Errorf("Output file was not created")
	}
}

func TestGetParentDir(t *testing.T) {
	testCases := []struct {
		filePath string
		expected string
	}{
		{"/path/to/file.go", "/path/to"},
		{"file.go", "."},
		{"/file.go", "/"},
		{"path/file.go", "path"},
		{"a/b/c/file.go", "a/b/c"},
	}

	for _, tc := range testCases {
		result := getParentDir(tc.filePath)
		if result != tc.expected {
			t.Errorf("For path '%s', expected '%s', got '%s'", tc.filePath, tc.expected, result)
		}
	}
}
