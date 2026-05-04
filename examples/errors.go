package examples

import "gorm.io/gorm"

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

func CommonErrors(db *gorm.DB) {
	var employees []Employee

	db.Preload("Departmen").Find(&employees)         // typo
	db.Preload("Department.Compan").Find(&employees) // typo in nested path
	db.Preload("Manager").Find(&employees)           // field doesn't exist
	db.Preload("Company").Find(&employees)           // not a direct relation of Employee
}
