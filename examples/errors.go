package examples

import "gorm.io/gorm"

// Examples showing common errors that the linter will catch

type Company struct {
	Name string
}

type Department struct {
	Name    string
	Company Company
}

type Employee struct {
	Name       string
	Department Department
}

// CommonErrors demonstrates invalid Preload calls that will be caught
func CommonErrors(db *gorm.DB) {
	var employees []Employee

	// ❌ Typo: "Departmen" instead of "Department"
	// Error: invalid preload: Departmen not found in Employee
	db.Preload("Departmen").Find(&employees)

	// ❌ Typo in nested relation: "Compan" instead of "Company"
	// Error: invalid preload: Department.Compan not found in Employee
	db.Preload("Department.Compan").Find(&employees)

	// ❌ Non-existent relation
	// Error: invalid preload: Manager not found in Employee
	db.Preload("Manager").Find(&employees)

	// ❌ Wrong nested path
	// Error: invalid preload: Company not found in Employee
	db.Preload("Company").Find(&employees)
}
