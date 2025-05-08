package models

import "time"

type User struct {
	ID                  int          `json:"id"`
	Name                string       `json:"name"`
	FirstName           string       `json:"first_name"`
	LastName            string       `json:"last_name"`
	PrimaryEmailAddress string       `json:"primary_email_address"`
	UpdatedAt           time.Time    `json:"updated_at"`
	CreatedAt           time.Time    `json:"created_at"`
	Disabled            bool         `json:"disabled"`
	SiteAdmin           bool         `json:"site_admin,omitempty"`
	Emails              []string     `json:"emails,omitempty"`
	EmployeeID          int          `json:"employee_id"`
	LinkedCandidateIDs  []int        `json:"linked_candidate_ids,omitempty"`
	Offices             []Office     `json:"offices,omitempty"`
	Departments         []Department `json:"departments,omitempty"`
}

type Office struct {
	ID                     int      `json:"id,omitempty"`
	Name                   string   `json:"name,omitempty"`
	Location               Location `json:"location,omitempty"`
	PrimaryContactUserID   int      `json:"primary_contact_user_id,omitempty"`
	ParentID               int      `json:"parent_id,omitempty"`
	ParentOfficeExternalID string   `json:"parent_office_external_id,omitempty"`
	ChildIDs               []int    `json:"child_ids,omitempty"`
	ChildOfficeExternalIDs []string `json:"child_office_external_ids,omitempty"`
	ExternalID             string   `json:"external_id,omitempty"`
}

type Department struct {
	ID                         int      `json:"id,omitempty"`
	Name                       string   `json:"name,omitempty"`
	ParentID                   int      `json:"parent_id,omitempty"`
	ParentDepartmentExternalID string   `json:"parent_department_external_id,omitempty"`
	ChildIDs                   []int    `json:"child_ids,omitempty"`
	ChildDepartmentExternalIDs []string `json:"child_department_external_ids,omitempty"`
	ExternalID                 string   `json:"external_id,omitempty"`
}

type Location struct {
	Name string `json:"name,omitempty"`
}
