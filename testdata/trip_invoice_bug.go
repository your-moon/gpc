package testdata

import "gorm.io/gorm"

// Reproduces the exact bug described: wrong model picked for Preload calls
// where Trip and Invoice models are confused

type Staff struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

type Machine struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

type Location struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

type TripItem struct {
	ID     uint `gorm:"primaryKey"`
	TripID uint
}

type Trip struct {
	ID         uint       `gorm:"primaryKey"`
	DriverID   uint       `gorm:"column:driver_id;index"`
	Driver     *Staff     `gorm:"foreignKey:DriverID"`
	MachineID  uint       `gorm:"column:machine_id;index"`
	Machine    *Machine   `gorm:"foreignKey:MachineID"`
	LocationID uint       `gorm:"column:location_id;index"`
	Location   *Location  `gorm:"foreignKey:LocationID"`
	Items      []TripItem `gorm:"foreignKey:TripID"`
}

type Customer struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

type Item struct {
	ID        uint `gorm:"primaryKey"`
	InvoiceID uint
}

type Invoice struct {
	ID         uint      `gorm:"primaryKey"`
	CustomerID uint      `gorm:"column:customer_id;index"`
	Customer   *Customer `gorm:"foreignKey:CustomerID"`
	MachineID  uint      `gorm:"column:machine_id;index"`
	Machine    *Machine  `gorm:"foreignKey:MachineID"`
	LocationID uint      `gorm:"column:location_id;index"`
	Location   *Location `gorm:"foreignKey:LocationID"`
	StaffID    uint      `gorm:"column:staff_id;index"`
	Staff      *Staff    `gorm:"foreignKey:StaffID"`
	Items      []Item    `gorm:"foreignKey:InvoiceID"`
}

func TestTripQueries() {
	var db *gorm.DB

	// This should work correctly - Trip has Driver, Machine, Items fields
	var trips []Trip
	db.
		Preload("Driver").
		Preload("Machine").
		Preload("Items").
		Where("id = ?", 140924).
		Order("created_at asc").
		Find(&trips)
}

func TestInvoiceQueries() {
	var db *gorm.DB

	// This should work correctly - Invoice has Customer, Machine, Staff, Items fields
	var invoices []Invoice
	db.
		Preload("Customer").
		Preload("Machine").
		Preload("Staff").
		Preload("Items").
		Where("id = ?", 140924).
		Order("created_at asc").
		Find(&invoices)
}

func TestMixedQueries() {
	var db *gorm.DB

	// Trip query - should use Trip model
	var trips []Trip
	db.Preload("Driver").Find(&trips) // ✅ Should work - Trip has Driver

	// Invoice query - should use Invoice model
	var invoices []Invoice
	db.Preload("Customer").Find(&invoices) // ✅ Should work - Invoice has Customer

	// This should fail - Trip doesn't have Customer field
	var trips2 []Trip
	db.Preload("Customer").Find(&trips2) // ❌ Should fail - Trip has no Customer

	// This should fail - Invoice doesn't have Driver field
	var invoices2 []Invoice
	db.Preload("Driver").Find(&invoices2) // ❌ Should fail - Invoice has no Driver
}
