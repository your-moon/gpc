package vms_invoice_test

import (
	"gorm.io/gorm"
)

// VMS Invoice and payment models
type Invoice struct {
	ID            int64 `gorm:"primaryKey"`
	InvoiceNumber string
	TotalAmount   float64
	MachineID     int64
	StaffID       int64
	LocationID    int64
	OrgID         int64
	Machine       Machine      `gorm:"foreignKey:MachineID"`
	Staff         Staff        `gorm:"foreignKey:StaffID"`
	Location      Location     `gorm:"foreignKey:LocationID"`
	Org           Organization `gorm:"foreignKey:OrgID"`
	InvoiceRefund InvoiceRefund
	Items         []InvoiceItem
	Ebarimt       Ebarimt
}

type InvoiceItem struct {
	ID        int64 `gorm:"primaryKey"`
	InvoiceID int64
	ProductID int64
	Product   InvoiceProduct `gorm:"foreignKey:ProductID"`
}

// Note: Product type is defined in vms_auth_test.go, but this one has additional fields
type InvoiceProduct struct {
	ID         int64 `gorm:"primaryKey"`
	Name       string
	CategoryID int64
	Category   ProductCategory `gorm:"foreignKey:CategoryID"`
	Locales    []ProductLocale
}

type ProductCategory struct {
	ID      int64 `gorm:"primaryKey"`
	Name    string
	Locales []CategoryLocale
}

type ProductLocale struct {
	ID        int64 `gorm:"primaryKey"`
	ProductID int64
	Language  string
	Name      string
}

type CategoryLocale struct {
	ID         int64 `gorm:"primaryKey"`
	CategoryID int64
	Language   string
	Name       string
}

type InvoiceRefund struct {
	ID        int64 `gorm:"primaryKey"`
	InvoiceID int64
	Amount    float64
	Reason    string
}

type Ebarimt struct {
	ID        int64 `gorm:"primaryKey"`
	InvoiceID int64
	Status    string
	Data      string
}

type OrderRequest struct {
	ID             int64 `gorm:"primaryKey"`
	InvoiceID      int64
	MachineID      int64
	CustomerID     int64
	Invoice        Invoice  `gorm:"foreignKey:InvoiceID"`
	Machine        Machine  `gorm:"foreignKey:MachineID"`
	Customer       Customer `gorm:"foreignKey:CustomerID"`
	OrderGoodsList []OrderGoods
	Candidates     []Candidate
}

type OrderGoods struct {
	ID             int64 `gorm:"primaryKey"`
	OrderRequestID int64
	ProductID      int64
	Product        Product `gorm:"foreignKey:ProductID"`
}

type Candidate struct {
	ID             int64 `gorm:"primaryKey"`
	OrderRequestID int64
	ProductID      int64
	Category       ProductCategory `gorm:"foreignKey:CategoryID"`
}

type Customer struct {
	ID   int64 `gorm:"primaryKey"`
	Name string
}

// Test 1: Invoice with all relations
func ExampleInvoiceDetails() {
	var db *gorm.DB

	// Real VMS example: Get invoice with all relations
	var items []Invoice
	if err := db.
		Preload("Machine").
		Preload("Staff").
		Preload("Location.LocationCategory").
		Preload("InvoiceRefund").
		Preload("Org").
		Find(&items).Error; err != nil {
		// Handle error
	}
}

// Test 2: Invoice with product details
func ExampleInvoiceWithProducts() {
	var db *gorm.DB

	// Real VMS example: Get invoice with product details
	var items []Invoice
	if err := db.
		Preload("Items.Product").
		Preload("Items.Product.Category.Locales").
		Preload("Items.Product.Locales").
		Find(&items).Error; err != nil {
		// Handle error
	}
}

// Test 3: Order request with complex relations
func ExampleOrderRequest() {
	var db *gorm.DB

	// Real VMS example: Get order request with all relations
	var items []OrderRequest
	if err := db.
		Preload("Invoice").
		Preload("Machine").
		Preload("OrderGoodsList").
		Preload("Candidates").
		Find(&items).Error; err != nil {
		// Handle error
	}
}

// Test 4: Customer transaction with product details
func ExampleCustomerTransaction() {
	var db *gorm.DB

	// Real VMS example: Get customer transaction
	var items []InvoiceItem
	if err := db.
		Preload("Product.Category.Locales").
		Preload("Invoice").
		Preload("Product.Locales").
		Find(&items).Error; err != nil {
		// Handle error
	}
}
