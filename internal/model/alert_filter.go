package model

import "time"

// AlertFilter задаёт параметры фильтрации для выборки алертов на дашборде
type AlertFilter struct {
	OrganizationID *int           // Фильтр по ID организации
	VehicleID      *int           // Фильтр по конкретному автомобилю
	Severity       *SeverityLevel // Фильтр по критичности (LOW, MEDIUM, HIGH, CRITICAL)
	Status         *AlertStatus   // Фильтр по статусу (FIRED, ACKNOWLEDGED, RESOLVED)
	CreatedAt      *time.Time     // Выборка алертов, созданных начиная с этой даты
	AcknowledgedAt *time.Time     // Фильтр по дате подтверждения оператором
	AcknowledgedBy *int           // Фильтр по ID оператора, подтвердившего алерт
	ResolvedAt     *time.Time     // Фильтр по дате закрытия инцидента
	ResolvedBy     *int           // Фильтр по ID пользователя, закрывшего инцидент
}
