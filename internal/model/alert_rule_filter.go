package model

import "time"

// AlertRuleFilter задаёт параметры фильтрации и пагинации списка правил алертов
type AlertRuleFilter struct {
	OrganizationID *int           // Фильтр по ID организации-владельца
	Type           *AlertRuleType // Фильтр по типу правила (SPEED_EXCEEDED, etc.)
	Name           *string        // Подстрока для поиска по названию (ILIKE)
	MinThreshold   *float64       // Минимальный порог срабатывания
	MaxThreshold   *float64       // Максимальный порог срабатывания
	Severity       *SeverityLevel // Фильтр по уровню критичности
	Enabled        *bool          // Фильтр по статусу активности
	CreatedAt      *time.Time     // Выборка правил, созданных начиная с этой даты

	Limit  *int // Ограничение количества возвращаемых строк (пагинация)
	Offset *int // Смещение выборки (пагинация)
}
