package examples

import "gorm.io/gorm"

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

func CorrectUsage(db *gorm.DB) {
	var orders []Order

	db.Preload("User").Find(&orders)
	db.Preload("User.Profile").Find(&orders)
	db.Preload("User.Profile.Address").Find(&orders)
}
