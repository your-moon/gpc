package main

import (
	"gorm.io/gorm"
)

// VMS Machine-related models
type Machine struct {
	ID         int64 `gorm:"primaryKey"`
	Name       string
	ScanCode   string
	Status     string
	StaffID    int64
	LocationID int64
	ModelID    int64
	Staff      Staff        `gorm:"foreignKey:StaffID"`
	Location   Location     `gorm:"foreignKey:LocationID"`
	Model      MachineModel `gorm:"foreignKey:ModelID"`
	MachineQr  MachineQr
	Map        Map
}

type Location struct {
	ID               int64 `gorm:"primaryKey"`
	Name             string
	LocationCategory LocationCategory `gorm:"foreignKey:LocationCategoryID"`
	Staff            Staff            `gorm:"foreignKey:StaffID"`
}

type LocationCategory struct {
	ID   int64 `gorm:"primaryKey"`
	Name string
}

type MachineModel struct {
	ID   int64 `gorm:"primaryKey"`
	Name string
}

type MachineQr struct {
	ID        int64 `gorm:"primaryKey"`
	MachineID int64
	QRCode    string
}

type Map struct {
	ID   int64 `gorm:"primaryKey"`
	Name string
}

type MachineLocationHistory struct {
	ID         int64 `gorm:"primaryKey"`
	MachineID  int64
	LocationID int64
	OrgID      int64
	Org        Organization `gorm:"foreignKey:OrgID"`
	Location   Location     `gorm:"foreignKey:LocationID"`
}

// Test 1: Machine details with all relations
func TestMachineDetails() {
	var db *gorm.DB

	// Real VMS example: Get machine with all relations
	machine := Machine{}
	if err := db.Where("org_id = ?", 1).
		Preload("Location").
		Preload("Staff").
		Preload("Model").
		Preload("Map").
		Preload("MachineQr").
		First(&machine, 1).Error; err != nil {
		// Handle error
	}

	// Real VMS example: Get machine with different org
	if err := db.
		Preload("Location").
		Preload("Staff").
		Preload("Model").
		Preload("Map").
		Preload("MachineQr").
		First(&machine, 1).Error; err != nil {
		// Handle error
	}
}

// Test 2: Machine location history
func TestMachineLocationHistory() {
	var db *gorm.DB

	// Real VMS example: Get machine location history
	var items []MachineLocationHistory
	if err := db.Where("org_id = ?", 1).
		Preload("Org").
		Preload("Location").
		Find(&items).Error; err != nil {
		// Handle error
	}
}

// Test 3: Location management
func TestLocationManagement() {
	var db *gorm.DB

	// Real VMS example: Get locations with category and staff
	var items []Location
	if err := db.
		Preload("LocationCategory").
		Preload("Staff").
		Find(&items).Error; err != nil {
		// Handle error
	}

	// Real VMS example: Get organization locations
	var orgItems []Location
	if err := db.Where("org_id = ?", 1).
		Preload("LocationCategory").
		Preload("Staff").
		Find(&orgItems).Error; err != nil {
		// Handle error
	}
}
