package dto

// RegisterOrganizationRequest — тело запроса POST /register: данные новой организации и её администратора
type RegisterOrganizationRequest struct {
	OrganizationName string `json:"organization_name"`
	AdminName        string `json:"admin_name"`
	AdminEmail       string `json:"admin_email"`
	AdminPassword    string `json:"admin_password"`
}
