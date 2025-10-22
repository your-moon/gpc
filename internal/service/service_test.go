package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/your-moon/gpc/internal/testutils"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name         string
		outputFormat string
		outputFile   string
	}{
		{
			name:         "Console output",
			outputFormat: "console",
			outputFile:   "test.json",
		},
		{
			name:         "JSON output",
			outputFormat: "json",
			outputFile:   "output.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.outputFormat, tt.outputFile, false, false)

			if svc.outputFormat != tt.outputFormat {
				t.Errorf("Expected output format '%s', got '%s'", tt.outputFormat, svc.outputFormat)
			}

			if svc.outputFile != tt.outputFile {
				t.Errorf("Expected output file '%s', got '%s'", tt.outputFile, svc.outputFile)
			}
		})
	}
}

func TestAnalyzeTarget_File(t *testing.T) {
	tests := []struct {
		name         string
		files        []testutils.TestFile
		outputFormat string
		outputFile   string
	}{
		{
			name: "Single file analysis",
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
	var orders []Order
	db.Preload("User").Find(&orders)
}`,
				},
			},
			outputFormat: "json",
			outputFile:   "test_service_output.json",
		},
		{
			name: "Console output",
			files: []testutils.TestFile{
				{
					Name: "test.go",
					Content: `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

func TestPreloads() {
	var db *gorm.DB
	var users []User
	db.Preload("Profile").Find(&users)
}`,
				},
			},
			outputFormat: "console",
			outputFile:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup, filePaths := testutils.CreateTestFiles(t, tt.files)
			defer cleanup()

			// Test service
			svc := NewService(tt.outputFormat, tt.outputFile, false, false)
			err := svc.AnalyzeTarget(filePaths[0])
			if err != nil {
				t.Fatalf("Service analysis failed: %v", err)
			}

			// Verify output file was created (only for JSON output)
			if tt.outputFormat == "json" && tt.outputFile != "" {
				if _, err := os.Stat(tt.outputFile); os.IsNotExist(err) {
					t.Errorf("Output file was not created")
				}
				defer os.Remove(tt.outputFile)
			}
		})
	}
}

func TestAnalyzeTarget_Directory(t *testing.T) {
	tests := []struct {
		name         string
		files        []testutils.TestFile
		outputFormat string
		outputFile   string
	}{
		{
			name: "Directory analysis",
			files: []testutils.TestFile{
				{
					Name: "test1.go",
					Content: `package main
import "gorm.io/gorm"
type User struct { ID int64; Name string }
func Test1() { var db *gorm.DB; var users []User; db.Preload("Profile").Find(&users) }`,
				},
				{
					Name: "test2.go",
					Content: `package main
import "gorm.io/gorm"
type Order struct { ID int64; UserID int64 }
func Test2() { var db *gorm.DB; var orders []Order; db.Preload("User").Find(&orders) }`,
				},
			},
			outputFormat: "json",
			outputFile:   "test_dir_output.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup, filePaths := testutils.CreateTestFiles(t, tt.files)
			defer cleanup()

			// Test service - analyze the directory containing the test files
			svc := NewService(tt.outputFormat, tt.outputFile, false, false)
			err := svc.AnalyzeTarget(filepath.Dir(filePaths[0]))
			if err != nil {
				t.Fatalf("Service analysis failed: %v", err)
			}

			// Verify output file was created
			if tt.outputFormat == "json" && tt.outputFile != "" {
				if _, err := os.Stat(tt.outputFile); os.IsNotExist(err) {
					t.Errorf("Output file was not created")
				}
				defer os.Remove(tt.outputFile)
			}
		})
	}
}

func TestGetParentDir(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "Nested path",
			filePath: "/path/to/file.go",
			expected: "/path/to",
		},
		{
			name:     "Current directory file",
			filePath: "file.go",
			expected: ".",
		},
		{
			name:     "Root directory file",
			filePath: "/file.go",
			expected: "/",
		},
		{
			name:     "Single level path",
			filePath: "path/file.go",
			expected: "path",
		},
		{
			name:     "Multi-level path",
			filePath: "a/b/c/file.go",
			expected: "a/b/c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getParentDir(tt.filePath)
			if result != tt.expected {
				t.Errorf("For path '%s', expected '%s', got '%s'", tt.filePath, tt.expected, result)
			}
		})
	}
}
