package model

// UserRole - перечисление ролей
type UserRole string

const (
	// UserRoleAdmin - админ: полный доступ ко всему
	UserRoleAdmin UserRole = "ADMIN"
	// UserRoleDispatcher - диспетчер: управляет машинами/водителями/рейсами
	UserRoleDispatcher UserRole = "DISPATCHER"
	// UserRoleDriver - водитель: только чтение своих рейсов/телеметрии
	UserRoleDriver UserRole = "DRIVER"
	// UserRoleAnalytic - аналитик: только чтение всех данных своей организации
	UserRoleAnalytic UserRole = "ANALYTIC"
)

// User определяет сущность пользователя
type User struct {
	ID             int  `json:"id"`
	OrganizationID int  `json:"organization_id"`
	DriverID       *int `json:"driver_id"`

	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`

	Role UserRole `json:"role"`
}
