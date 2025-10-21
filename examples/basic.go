package examples

import "gorm.io/gorm"

// Basic example showing correct usage

type Address struct {
	City    string
	Country string
}

type Profile struct {
	Bio     string
	Address Address
}

type User struct {
	Name    string
	Email   string
	Profile Profile
}

type Order struct {
	ID     uint
	Amount float64
	User   User
}

// CorrectUsage demonstrates valid Preload calls
func CorrectUsage(db *gorm.DB) {
	var orders []Order

	// ✅ Single level preload
	db.Preload("User").Find(&orders)

	// ✅ Nested preload
	db.Preload("User.Profile").Find(&orders)

	// ✅ Deep nested preload
	db.Preload("User.Profile.Address").Find(&orders)
}
