package loader

import (
	"testing"

	"github.com/your-moon/gpc/internal/testutil"
)

func TestLoad(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

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

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Find(&orders)
}
`,
	})

	result, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(result.Packages) == 0 {
		t.Fatal("no packages loaded")
	}
	pkg := result.Packages[0]
	if pkg.TypesInfo == nil {
		t.Fatal("TypesInfo is nil")
	}
	if len(pkg.Syntax) == 0 {
		t.Fatal("no syntax trees loaded")
	}
}

func TestLoad_InvalidDir(t *testing.T) {
	_, err := Load("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for invalid directory")
	}
}

func TestLoad_MultiplePackages(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "testmod/models"

func main() {
	_ = models.User{}
}
`,
		"models/models.go": `package models

type User struct {
	ID   int64
	Name string
}
`,
	})

	result, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(result.Packages) < 2 {
		t.Fatalf("expected at least 2 packages, got %d", len(result.Packages))
	}
}
