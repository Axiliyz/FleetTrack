package dto

import "fleettrack/internal/model"

// UpdateDriverRequest описывает тело HTTP запроса на частичное обновление водителя.
// Поля organization_id нет: хендлер разбирает тело с DisallowUnknownFields,
// и попытка перенести водителя в другую организацию получает 400
type UpdateDriverRequest struct {
	Name *string `json:"name"`
}

// ToDomainModel преобразует UpdateDriverRequest в доменную модель
func (u *UpdateDriverRequest) ToDomainModel() model.UpdateDriver {
	return model.UpdateDriver{
		Name: u.Name,
	}
}
