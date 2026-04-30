package relations

import "testing"

func TestResolveModel_Basic(t *testing.T) {
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
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	m := resolveModel(chains[0])
	if m == nil {
		t.Fatal("expected resolved model, got nil")
	}
	if m.name != "Order" {
		t.Errorf("expected model name 'Order', got '%s'", m.name)
	}
}

func TestResolveModel_Pointer(t *testing.T) {
	chains := loadAndCollect(t, map[string]string{
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
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	m := resolveModel(chains[0])
	if m == nil {
		t.Fatal("expected resolved model, got nil")
	}
	if m.name != "User" {
		t.Errorf("expected 'User', got '%s'", m.name)
	}
}

func TestResolveModel_CrossPackage(t *testing.T) {
	chains := loadAndCollect(t, map[string]string{
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
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	m := resolveModel(chains[0])
	if m == nil {
		t.Fatal("expected resolved model, got nil")
	}
	if m.name != "Order" {
		t.Errorf("expected 'Order', got '%s'", m.name)
	}
	if m.pkg == nil || m.pkg.Name() != "models" {
		t.Errorf("expected package 'models', got %v", m.pkg)
	}
}
