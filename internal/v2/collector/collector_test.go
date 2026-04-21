package collector

import (
	"testing"

	"github.com/your-moon/gpc/internal/v2/loader"
	"github.com/your-moon/gpc/internal/v2/testutil"
)

func TestCollect_BasicChain(t *testing.T) {
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
		t.Fatalf("Load failed: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}

	chain := chains[0]
	if len(chain.Preloads) != 1 {
		t.Fatalf("expected 1 preload, got %d", len(chain.Preloads))
	}
	if chain.Preloads[0].Relation != "User" {
		t.Errorf("expected relation 'User', got '%s'", chain.Preloads[0].Relation)
	}
	if chain.Terminal == nil {
		t.Fatal("expected terminal call, got nil")
	}
	if chain.Terminal.Method != "Find" {
		t.Errorf("expected terminal method 'Find', got '%s'", chain.Terminal.Method)
	}
}

func TestCollect_MultiplePreloads(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type Profile struct {
	Bio string
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
	db.Preload("User").Preload("User.Profile").Find(&orders)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	if len(chains[0].Preloads) != 2 {
		t.Fatalf("expected 2 preloads, got %d", len(chains[0].Preloads))
	}
}

func TestCollect_SeparateChains(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type Order struct {
	ID   int64
	User User
}

type Trip struct {
	ID     int64
	Driver string
}

func GetData(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Find(&orders)

	var trips []Trip
	db.Preload("Driver").Find(&trips)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 2 {
		t.Fatalf("expected 2 chains, got %d", len(chains))
	}
}

func TestCollect_NonGormPreloadIgnored(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

type Cache struct{}

func (c *Cache) Preload(key string) {}

type User struct {
	ID int64
}

func GetData(db *gorm.DB) {
	cache := &Cache{}
	cache.Preload("key")

	var users []User
	db.Preload("Name").Find(&users)
}
`,
	})

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain (only gorm), got %d", len(chains))
	}
}

func TestCollect_ConstantPreloadArg(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
		"main.go": `package main

import "gorm.io/gorm"

const RelUser = "User"

type User struct {
	ID   int64
	Name string
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

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	if chains[0].Preloads[0].Relation != "User" {
		t.Errorf("expected constant-folded relation 'User', got '%s'", chains[0].Preloads[0].Relation)
	}
}

func TestCollect_DynamicPreloadArg(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
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

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	if chains[0].Preloads[0].Relation != "" {
		t.Errorf("expected empty relation for dynamic arg, got '%s'", chains[0].Preloads[0].Relation)
	}
	if !chains[0].Preloads[0].Dynamic {
		t.Error("expected Dynamic=true for non-literal arg")
	}
}

func TestCollect_ClauseAssociations(t *testing.T) {
	dir := testutil.CreateTestModule(t, map[string]string{
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

	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	chains := Collect(result)
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	if chains[0].Preloads[0].Relation != "clause.Associations" {
		t.Errorf("expected 'clause.Associations', got '%s'", chains[0].Preloads[0].Relation)
	}
}
