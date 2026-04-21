package resolver

import (
	"testing"

	"github.com/your-moon/gpc/internal/v2/collector"
	"github.com/your-moon/gpc/internal/v2/loader"
	"github.com/your-moon/gpc/internal/v2/testutil"
)

func TestResolve_BasicModel(t *testing.T) {
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

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	chains := collector.Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}

	model := Resolve(chains[0])
	if model == nil {
		t.Fatal("expected resolved model, got nil")
	}
	if model.Name != "Order" {
		t.Errorf("expected model name 'Order', got '%s'", model.Name)
	}
}

func TestResolve_PointerModel(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID int64
}

func GetUser(db *gorm.DB) {
	var user User
	db.Preload("Profile").First(&user)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	chains := collector.Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}

	model := Resolve(chains[0])
	if model == nil {
		t.Fatal("expected resolved model, got nil")
	}
	if model.Name != "User" {
		t.Errorf("expected 'User', got '%s'", model.Name)
	}
}

func TestResolve_CrossPackageModel(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import (
	"gorm.io/gorm"
	"testmod/models"
)

func GetOrders(db *gorm.DB) {
	var orders []models.Order
	db.Preload("User").Find(&orders)
}
`,
		"models/models.go": `package models

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID     int64
	UserID int64
	User   User
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	chains := collector.Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}

	model := Resolve(chains[0])
	if model == nil {
		t.Fatal("expected resolved model, got nil")
	}
	if model.Name != "Order" {
		t.Errorf("expected 'Order', got '%s'", model.Name)
	}
	if model.Pkg == nil || model.Pkg.Name() != "models" {
		t.Errorf("expected package 'models', got %v", model.Pkg)
	}
}
