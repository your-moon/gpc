package testdata

import "gorm.io/gorm"

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

func TestPreload() {
	var db *gorm.DB
	var orders []Order

	db.Preload("User.Profile.Address").Find(&orders) // ✅ ok
	db.Preload("User.Profil.Address").Find(&orders)  // ❌ typo detected
}
