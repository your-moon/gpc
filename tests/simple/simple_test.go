package main

import (
	"gorm.io/gorm"
)

// Test structs
type Address struct {
	City string
}

type Profile struct {
	Address Address
}

type User struct {
	Profile Profile
}

type Order struct {
	User User
}

type Trip struct {
	Driver string
	Items  []string
}

type Invoice struct {
	Customer string
	Items    []string
}

func TestMoon() {
	var db *gorm.DB

	// ✅ Valid preloads
	var orders []Order
	var user User
	orderDB := db.Preload("User")
	userDB := db.Preload("User")
	orderDB.Find(&orders)
	userDB.Find(&user)
}

// Test functions with various GORM preload scenarios
func TestBasicPreloads() {
	var db *gorm.DB

	// ✅ Valid preloads
	var orders []Order
	db.Preload("User").Find(&orders)
	db.Preload("User.Profile").Find(&orders)
	db.Preload("User.Profile.Address").Find(&orders)

	// ❌ Invalid preloads (typos)
	db.Preload("Usr").Find(&orders)                 // typo: Usr instead of User
	db.Preload("User.Profil").Find(&orders)         // typo: Profil instead of Profile
	db.Preload("User.Profile.Addres").Find(&orders) // typo: Addres instead of Address
}

func TestNumberedVariables() {
	var db *gorm.DB

	// ✅ Test numbered variables (the key fix)
	var trips []Trip
	var trips2 []Trip
	var invoices []Invoice
	var invoices2 []Invoice

	// These should all work correctly
	db.Preload("Driver").Find(&trips)       // trips -> Trip
	db.Preload("Driver").Find(&trips2)      // trips2 -> Trip (not Trips2!)
	db.Preload("Customer").Find(&invoices)  // invoices -> Invoice
	db.Preload("Customer").Find(&invoices2) // invoices2 -> Invoice (not Invoices2!)

	// ❌ Wrong model preloads
	db.Preload("Customer").Find(&trips)  // Trip doesn't have Customer
	db.Preload("Driver").Find(&invoices) // Invoice doesn't have Driver
}

func TestMultiLineCalls() {
	var db *gorm.DB

	// ✅ Multi-line GORM calls
	var orders []Order
	db.
		Preload("User").
		Preload("User.Profile").
		Where("id = ?", 1).
		Find(&orders)

	// ✅ Complex multi-line with different models
	var trips []Trip
	db.
		Preload("Driver").
		Preload("Items").
		Order("created_at DESC").
		Find(&trips)
}

func TestEdgeCases() {
	var db *gorm.DB

	// ❌ Empty preloads
	db.Preload("").Find(&orders)

	// ❌ Malformed preloads
	db.Preload("...").Find(&orders)
	db.Preload("User..Profile").Find(&orders)
	db.Preload(".User").Find(&orders)
	db.Preload("User.").Find(&orders)
}
