package engine

import (
	"testing"

	"github.com/your-moon/gpc/internal/testutil"
)

func TestAnalyze_EndToEnd(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type Address struct {
	City string
}

type Profile struct {
	Bio     string
	Address Address
}

type User struct {
	ID      int64
	Name    string
	Profile Profile
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Find(&orders)
	db.Preload("User.Profile").Find(&orders)
	db.Preload("User.Profile.Address").Find(&orders)
	db.Preload("User.Profil").Find(&orders)
	db.Preload("Customer").Find(&orders)
}
`,
	})

	results, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}

	counts := map[string]int{}
	for _, r := range results {
		counts[r.Status]++
	}

	if counts["valid"] != 3 {
		t.Errorf("expected 3 correct, got %d", counts["valid"])
	}
	if counts["error"] != 2 {
		t.Errorf("expected 2 errors, got %d", counts["error"])
	}
}

func TestAnalyze_CrossPackage(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import (
	"gorm.io/gorm"
	"testmod/db"
)

func GetOrders(dbConn *gorm.DB) {
	var orders []db.Order
	dbConn.Preload("User.Profile").Find(&orders)
	dbConn.Preload("User.Profil").Find(&orders)
}
`,
		"db/models.go": `package db

type Profile struct {
	Bio string
}

type User struct {
	ID      int64
	Profile Profile
}

type Order struct {
	ID   int64
	User User
}
`,
	})

	results, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Status != "valid" {
		t.Errorf("expected first result 'correct', got '%s'", results[0].Status)
	}
	if results[1].Status != "error" {
		t.Errorf("expected second result 'error', got '%s'", results[1].Status)
	}
}

func TestAnalyze_ConstantFolding(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

const RelUser = "User"

type User struct {
	ID int64
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload(RelUser).Find(&orders)
}
`,
	})

	results, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "valid" {
		t.Errorf("expected 'correct', got '%s'", results[0].Status)
	}
	if results[0].Relation != "User" {
		t.Errorf("expected relation 'User', got '%s'", results[0].Relation)
	}
}

func TestAnalyze_NoPreloads(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID int64
}

func GetUsers(db *gorm.DB) {
	var users []User
	db.Find(&users)
}
`,
	})

	results, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
