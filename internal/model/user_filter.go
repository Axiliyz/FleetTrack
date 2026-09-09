package model

type UserFilter struct {
	OrganizationID *int
	Role           *UserRole

	Limit  int
	Offset int
}
