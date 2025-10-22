package main

type Staff struct {
	ID       int64
	Email    string
	RoleID   int64
	OrgID    int64
	IsActive bool
	Role     Role
	Org      Organization
}

type Role struct {
	ID   int64
	Name string
	Org  Organization
}

type Organization struct {
	ID   int64
	Name string
}
