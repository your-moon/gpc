package relations

import "testing"

func TestVerify_SimpleValid(t *testing.T) {
	chains := loadAndCollect(t, map[string]string{
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
	results := Verify(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "valid" {
		t.Errorf("expected 'valid', got '%s'", results[0].Status)
	}
}

func TestVerify_SimpleInvalid(t *testing.T) {
	chains := loadAndCollect(t, map[string]string{
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
	results := Verify(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "error" {
		t.Errorf("expected 'error', got '%s'", results[0].Status)
	}
}

func TestVerify_NestedValid(t *testing.T) {
	chains := loadAndCollect(t, map[string]string{
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
	results := Verify(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "valid" {
		t.Errorf("expected 'valid', got '%s'", results[0].Status)
	}
}

func TestVerify_NestedInvalid_DeepTypo(t *testing.T) {
	chains := loadAndCollect(t, map[string]string{
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
	results := Verify(chains)
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

func TestVerify_DynamicSkipped(t *testing.T) {
	chains := loadAndCollect(t, map[string]string{
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
	results := Verify(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "skipped" {
		t.Errorf("expected 'skipped' for dynamic arg, got '%s'", results[0].Status)
	}
}

func TestVerify_CrossPackageNested(t *testing.T) {
	chains := loadAndCollect(t, map[string]string{
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
	results := Verify(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "valid" {
		t.Errorf("expected 'valid', got '%s'", results[0].Status)
	}
}

func TestVerify_EmbeddedStruct(t *testing.T) {
	chains := loadAndCollect(t, map[string]string{
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
	results := Verify(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "valid" {
		t.Errorf("expected 'valid' for embedded field, got '%s'", results[0].Status)
	}
}

func TestVerify_ClauseAssociations(t *testing.T) {
	chains := loadAndCollect(t, map[string]string{
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
	results := Verify(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "valid" {
		t.Errorf("expected 'valid' for clause.Associations, got '%s'", results[0].Status)
	}
}

func TestVerify_EmptyRelation(t *testing.T) {
	chains := loadAndCollect(t, map[string]string{
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
	results := Verify(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "error" {
		t.Errorf("expected 'error' for empty relation, got '%s'", results[0].Status)
	}
}

func TestVerify_LineNumberPropagated(t *testing.T) {
	chains := loadAndCollect(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID int64
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Find(&orders)
}
`,
	})
	results := Verify(chains)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// The Preload("User") call sits on line 16 of the fixture (after the
	// opening newline + the leading `package main`). What we care about is
	// that Line is non-zero — i.e. collector pre-resolved the position.
	if results[0].Line == 0 {
		t.Errorf("expected non-zero Line, got 0")
	}
}
