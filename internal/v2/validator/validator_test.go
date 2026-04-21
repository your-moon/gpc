package validator

import (
	"testing"

	"github.com/your-moon/gpc/internal/v2/collector"
	"github.com/your-moon/gpc/internal/v2/loader"
	"github.com/your-moon/gpc/internal/v2/testutil"
)

func loadAndCollect(t *testing.T, files map[string]string) ([]collector.Chain, *loader.Result) {
	t.Helper()
	dir := testutil.CreateTestModule(t, files)
	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	chains := collector.Collect(result)
	return chains, result
}

func TestValidate_SimpleValid(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
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

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "correct" {
		t.Errorf("expected 'correct', got '%s'", results[0].Status)
	}
}

func TestValidate_SimpleInvalid(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
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
	db.Preload("Customer").Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "error" {
		t.Errorf("expected 'error', got '%s'", results[0].Status)
	}
}

func TestValidate_NestedValid(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
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
	Profile Profile
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User.Profile.Address").Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "correct" {
		t.Errorf("expected 'correct', got '%s'", results[0].Status)
	}
}

func TestValidate_NestedInvalid_DeepTypo(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
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
	Profile Profile
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User.Profil.Address").Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "error" {
		t.Errorf("expected 'error', got '%s'", results[0].Status)
	}
	if results[0].Relation != "User.Profil.Address" {
		t.Errorf("expected relation 'User.Profil.Address', got '%s'", results[0].Relation)
	}
}

func TestValidate_DynamicSkipped(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID int64
}

func GetData(db *gorm.DB, field string) {
	var users []User
	db.Preload(field).Find(&users)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "skipped" {
		t.Errorf("expected 'skipped' for dynamic arg, got '%s'", results[0].Status)
	}
}

func TestValidate_CrossPackageNested(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import (
	"gorm.io/gorm"
	"testmod/models"
)

func GetOrders(db *gorm.DB) {
	var orders []models.Order
	db.Preload("User.Profile").Find(&orders)
}
`,
		"models/models.go": `package models

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

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "correct" {
		t.Errorf("expected 'correct', got '%s'", results[0].Status)
	}
}

func TestValidate_EmbeddedStruct(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type BaseModel struct {
	Creator User
}

type Order struct {
	BaseModel
	ID int64
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("Creator").Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "correct" {
		t.Errorf("expected 'correct' for embedded field, got '%s'", results[0].Status)
	}
}

func TestValidate_ClauseAssociations(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type User struct {
	ID int64
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload(clause.Associations).Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "correct" {
		t.Errorf("expected 'correct' for clause.Associations, got '%s'", results[0].Status)
	}
}

func TestValidate_EmptyRelation(t *testing.T) {
	chains, _ := loadAndCollect(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type Order struct {
	ID int64
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("").Find(&orders)
}
`,
	})

	results := Validate(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "error" {
		t.Errorf("expected 'error' for empty relation, got '%s'", results[0].Status)
	}
}
