package handler

import "fleettrack/internal/model"

// scopeOrganizationID возвращает organization_id, который реально должен применяться к фильтру:
// всегда собственная организация вызывающего, что бы клиент ни прислал в теле - включая ADMIN
func scopeOrganizationID(authCtx model.AuthContext) *int {
	orgID := authCtx.OrganizationID
	return &orgID
}

// scopeDriverID при роли DRIVER принудительно ограничивает фильтр их собственным driver_id,
// игнорируя то, что прислал клиент. Для остальных ролей requested не трогается.
// Возвращает model.ErrDriverNotLinked, если у DRIVER-аккаунта нет привязанного водителя -
// в этом случае filter.DriverID нельзя оставлять nil, иначе фильтр снимется и DRIVER увидит чужие данные.
func scopeDriverID(authCtx model.AuthContext, requested *int) (*int, error) {
	if authCtx.Role != model.UserRoleDriver {
		return requested, nil
	}
	if authCtx.DriverID == nil {
		return nil, model.ErrDriverNotLinked
	}
	return authCtx.DriverID, nil
}

// requireOwnOrg проверяет доступ к уже загруженному ресурсу конкретной организации.
// Чужая организация (для любой роли, включая ADMIN) возвращается как model.ErrNotFound
// (а не 403) - чтобы не подтверждать сам факт существования записи в чужой организации.
func requireOwnOrg(authCtx model.AuthContext, resourceOrgID int) error {
	if resourceOrgID != authCtx.OrganizationID {
		return model.ErrNotFound
	}
	return nil
}
