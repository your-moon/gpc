package examples

import "gorm.io/gorm"

type Tag struct {
	Name string
}

type Category struct {
	Name string
	Tags []Tag
}

type Image struct {
	URL string
}

type Product struct {
	Name     string
	Category Category
	Images   []Image
}

type OrderItem struct {
	Quantity int
	Product  Product
}

type Customer struct {
	Name  string
	Email string
}

type ComplexOrder struct {
	ID         uint
	Customer   Customer
	OrderItems []OrderItem
}

func ComplexPreloads(db *gorm.DB) {
	var orders []ComplexOrder

	db.Preload("Customer").
		Preload("OrderItems").
		Find(&orders)

	db.Preload("OrderItems.Product").Find(&orders)
	db.Preload("OrderItems.Product.Category").Find(&orders)

	db.Preload("OrderItems.Product.Category.Tags").
		Preload("OrderItems.Product.Images").
		Find(&orders)
}

func ComplexErrors(db *gorm.DB) {
	var orders []ComplexOrder

	db.Preload("OrderItems.Product.Categor").Find(&orders) // typo: Categor → Category
	db.Preload("Items").Find(&orders)                      // wrong name: Items → OrderItems
	db.Preload("Product").Find(&orders)                    // skips a level
}
