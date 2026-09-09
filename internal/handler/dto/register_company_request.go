package dto

type RegisterCompanyRequest struct {
	OrganizationName string `json:"organization_name"`
	AdminName        string `json:"admin_name"`
	AdminEmail       string `json:"admin_email"`
	AdminPassword    string `json:"admin_password"`
}
