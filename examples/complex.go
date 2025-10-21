package examples

import "gorm.io/gorm"

// Complex example with multiple relations and deep nesting

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

// ComplexPreloads demonstrates complex preload scenarios
func ComplexPreloads(db *gorm.DB) {
	var orders []ComplexOrder

	// ✅ Multiple separate preloads
	db.Preload("Customer").
		Preload("OrderItems").
		Find(&orders)

	// ✅ Deep nested preload
	db.Preload("OrderItems.Product").Find(&orders)

	// ✅ Very deep nested preload
	db.Preload("OrderItems.Product.Category").Find(&orders)

	// ✅ Multiple deep nested preloads
	db.Preload("OrderItems.Product.Category.Tags").
		Preload("OrderItems.Product.Images").
		Find(&orders)
}

// ComplexErrors demonstrates complex error scenarios
func ComplexErrors(db *gorm.DB) {
	var orders []ComplexOrder

	// ❌ Typo in deep nested path: "Categor" instead of "Category"
	db.Preload("OrderItems.Product.Categor").Find(&orders)

	// ❌ Wrong relation name
	db.Preload("Items").Find(&orders)

	// ❌ Skipping a level in the path
	db.Preload("Product").Find(&orders)
}
