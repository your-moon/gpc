package vms

import (
	"gorm.io/gorm"
)

// VMS Real-world models from your backend
type Staff struct {
	ID       int64 `gorm:"primaryKey"`
	Email    string
	RoleID   int64
	OrgID    int64
	IsActive bool
	Role     Role         `gorm:"foreignKey:RoleID"`
	Org      Organization `gorm:"foreignKey:OrgID"`
}

type Role struct {
	ID            int64 `gorm:"primaryKey"`
	Name          string
	PermissionIDs []int64
	GroupIDs      []int64
	OrgID         int64
	Org           Organization `gorm:"foreignKey:OrgID"`
}

type Organization struct {
	ID       int64 `gorm:"primaryKey"`
	Name     string
	IsActive bool
}

type Permission struct {
	ID    int64 `gorm:"primaryKey"`
	Name  string
	OrgID int64
	Org   Organization `gorm:"foreignKey:OrgID"`
}

// Test 1: Authentication flow - Staff login with Role and Org
func ExampleStaffLogin() {
	var db *gorm.DB

	// Real VMS example: Staff login with preloaded Role and Org
	var user Staff
	if err := db.Where("email = ?", "admin@example.com").
		Where("is_active = ?", true).
		Preload("Role").
		Preload("Org").
		First(&user).Error; err != nil {
		// Handle error
	}

	// Real VMS example: Get role with Org preloaded
	role := Role{}
	db.Preload("Org").First(&role, user.RoleID)

	// Real VMS example: Get permissions for role
	permissions := []Permission{}
	if len(role.PermissionIDs) > 0 {
		db.Where("id in (?)", role.PermissionIDs).Find(&permissions)
	}
}

// Test 2: Role management with Organization
func ExampleRoleManagement() {
	var db *gorm.DB

	// Real VMS example: Get role with organization
	role := Role{}
	db.Preload("Org").First(&role, 1)

	// Real VMS example: Get all roles for organization
	roles := []Role{}
	db.Where("org_id = ?", 1).Find(&roles)
}

// Test 3: Permission checking
func ExamplePermissionCheck() {
	var db *gorm.DB

	// Real VMS example: Get permissions with organization
	permissions := []Permission{}
	db.Where("org_id = ?", 1).Find(&permissions)
}
