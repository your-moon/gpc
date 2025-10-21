package testdata

// This test reproduces the exact bug described where wrong models are picked
// for Preload calls in GORM queries

type Driver struct {
	ID   uint
	Name string
}

type Machine struct {
	ID   uint
	Name string
}

type TripItem struct {
	ID     uint
	TripID uint
}

type Trip struct {
	ID        uint
	DriverID  uint
	Driver    *Driver
	MachineID uint
	Machine   *Machine
	Items     []TripItem
}

type Customer struct {
	ID   uint
	Name string
}

type Item struct {
	ID        uint
	InvoiceID uint
}

type Invoice struct {
	ID         uint
	CustomerID uint
	Customer   *Customer
	MachineID  uint
	Machine    *Machine
	StaffID    uint
	Staff      *Driver // Using Driver as Staff for simplicity
	Items      []Item
}

// Mock GORM DB interface
type DB struct{}

func (db *DB) Preload(associations ...string) *DB               { return db }
func (db *DB) Where(query interface{}, args ...interface{}) *DB { return db }
func (db *DB) Order(value interface{}) *DB                      { return db }
func (db *DB) Find(dest interface{}) *DB                        { return db }

func TripQueries() {
	var db *DB

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

func InvoiceQueries() {
	var db *DB

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

func WrongModelDetection() {
	var db *DB

	// This should fail - Trip doesn't have Customer field
	var trips []Trip
	db.Preload("Customer").Find(&trips) // ❌ Should fail - Trip has no Customer

	// This should fail - Invoice doesn't have Driver field
	var invoices []Invoice
	db.Preload("Driver").Find(&invoices) // ❌ Should fail - Invoice has no Driver
}

func main() {
	// This is just a regular Go file, not a test file
}
