package models

import "time"

type User struct {
	ID                  int
	Name                string
	LastName            string
	PrimaryEmailAddress string
	UpdatedAt           time.Time
	CreatedAt           time.Time
	Disabled            bool
	SiteAdmin           bool
	Emails              []string
	EmployeeID          int
	LinkedCandidateIDs  []int
	Offices             []Office
	Departments         []Department
}

type Office struct {
	ID                     int
	Name                   string
	Location               Location
	PrimaryContactUserID   int
	ParentID               int
	ParentOfficeExternalID string
	ChildIDs               []int
	ChildOfficeExternalIDs []string
	ExternalID             string
}

type Department struct {
	ID                         int
	Name                       string
	ParentID                   int
	ParentDepartmentExternalID string
	ChildIDs                   []int
	ChildDeparmentExternalIDs  []string
	ExternalID                 string
}

type Location struct {
	Name string
}
