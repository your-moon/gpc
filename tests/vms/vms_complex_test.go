package vms

import (
	"gorm.io/gorm"
)

// VMS Complex patterns that actually fail in real codebase
// Note: Machine, Staff, Location, MachineModel, Product types are defined in vms_auth_test.go

type Slot struct {
	ID        int64 `gorm:"primaryKey"`
	ProductID int64
	Product   Product `gorm:"foreignKey:ProductID"`
}

type Trip struct {
	ID        int64 `gorm:"primaryKey"`
	DriverID  int64
	MachineID int64
	Driver    Driver  `gorm:"foreignKey:DriverID"`
	Machine   Machine `gorm:"foreignKey:MachineID"`
	Items     []TripItem
}

type Driver struct {
	ID   int64 `gorm:"primaryKey"`
	Name string
}

type TripItem struct {
	ID        int64 `gorm:"primaryKey"`
	TripID    int64
	ProductID int64
	Product   Product `gorm:"foreignKey:ProductID"`
}

// Test 1: Very long multi-line method chain (like real VMS machine.go:215)
func ExampleComplexMachineQuery() {
	var db *gorm.DB

	// This is the pattern that fails in real VMS backend
	var machines []Machine
	if err := db.
		Scopes(func(db *gorm.DB) *gorm.DB {
			return db.Limit(10).Offset(0)
		}).
		Order("machines.created_at desc").
		Select(`
			machines.id as id,
			machines.name as name,
			machines.scan_code as scan_code,
			machines.status as status,
			machines.online_status as online_status,
			machines.power_status as power_status,
			machines.staff_id as staff_id,
			machines.location_id as location_id,
			machines.model_id as model_id,
			machines.is_sale_locked as is_sale_locked,
			machines.created_at as created_at
		`).
		Preload("Staff", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name, code")
		}).
		Preload("Location", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name")
		}).
		Preload("Model", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name")
		}).
		Find(&machines).Error; err != nil {
		// Handle error
	}
}

// Test 2: Product with slots (like real VMS product.go:104)
func ExampleProductWithSlots() {
	var db *gorm.DB

	// This pattern also fails in real VMS backend
	var product Product
	if err := db.
		Preload("Slots").
		First(&product, 1).Error; err != nil {
		// Handle error
	}
}

// Test 3: Trip with complex nested relations (like real VMS trip.go:563)
func ExampleTripWithComplexRelations() {
	var db *gorm.DB

	// This pattern fails in real VMS backend
	var trip Trip
	if err := db.
		Preload("Driver").
		Preload("Machine").
		Preload("Items.Product").
		First(&trip, 1).Error; err != nil {
		// Handle error
	}
}

// Test 4: Multiple preloads with different scopes
func ExampleMultipleScopes() {
	var db *gorm.DB

	// First scope
	var machines1 []Machine
	db.Preload("Staff").Find(&machines1)

	// Second scope with same variable name
	var machines2 []Machine
	db.Preload("Location").Find(&machines2)

	// Third scope
	var machines3 []Machine
	db.Preload("Model").Find(&machines3)
}

// Test 5: Dynamic query building
func ExampleDynamicQuery() {
	var db *gorm.DB

	// Build query dynamically
	query := db.Model(&Machine{})

	// Add conditions
	query = query.Where("status = ?", "active")
	query = query.Where("org_id = ?", 1)

	// Add preloads
	query = query.Preload("Staff")
	query = query.Preload("Location")

	// Execute
	var machines []Machine
	query.Find(&machines)
}
