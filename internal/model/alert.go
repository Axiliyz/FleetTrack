// Package model содержит основные доменные сущности и типы бизнес-логики
package model

import (
	"time"
)

// AlertRuleType определяет категорию события или нарушения, отслеживаемого правилом
type AlertRuleType string

const (
	// AlertRuleSpeedExceed срабатывает при превышении транспортным средством заданной скорости
	AlertRuleSpeedExceed AlertRuleType = "SPEED_EXCEEDED"

	// AlertRuleDeviceOffline срабатывает при отсутствии телеметрии от устройства дольше допустимого времени
	AlertRuleDeviceOffline AlertRuleType = "DEVICE_OFFLINE"

	// AlertRuleLowFuel срабатывает при падении уровня топлива ниже установленного процента
	AlertRuleLowFuel AlertRuleType = "LOW_FUEL"
)

// SeverityLevel определяет уровень критичности правила или возникшего алерта
type SeverityLevel string

const (
	// SeverityLevelLow — информационный уровень нарушения
	SeverityLevelLow SeverityLevel = "LOW"

	// SeverityLevelMedium — предупреждение умеренной важности
	SeverityLevelMedium SeverityLevel = "MEDIUM"

	// SeverityLevelHigh — высокая критичность, требующая оперативного внимания диспетчера
	SeverityLevelHigh SeverityLevel = "HIGH"

	// SeverityLevelCritical — критическая авария или опасное нарушение, требующие немедленной реакции
	SeverityLevelCritical SeverityLevel = "CRITICAL"
)

// AlertStatus отражает состояние жизненного цикла алерта в stateful alerting модели
type AlertStatus string

const (
	// AlertStatusFired — активное незакрытое нарушение, только что обнаруженное движком
	AlertStatusFired AlertStatus = "FIRED"

	// AlertStatusAcknowledged — алерт принят в обработку оператором/диспетчером, но нарушение ещё актуально
	AlertStatusAcknowledged AlertStatus = "ACKNOWLEDGED"

	// AlertStatusResolved — нарушение устранено (автоматически по телеметрии или вручную оператором)
	AlertStatusResolved AlertStatus = "RESOLVED"
)

// Alert представляет сработавший алерт для транспортного средства
type Alert struct {
	ID             int           // Уникальный ID алерта
	OrganizationID int           // ID организации, которой принадлежит машина
	VehicleID      int           // ID автомобиля
	RuleID         int           // ID правила, породившего алерт
	Type           AlertRuleType // Тип нарушения на момент срабатывания
	Message        string        // Человекочитаемое описание инцидента
	Severity       SeverityLevel // Уровень критичности инцидента
	Value          *float64      // Фактическое измеренное значение при срабатывании
	Status         AlertStatus   // Текущий статус жизненного цикла алерта
	CreatedAt      time.Time     // Время фиксации нарушения
	AcknowledgedAt *time.Time    // Время принятия алерта в работу оператором
	AcknowledgedBy *int          // ID пользователя-оператора, принявшего алерт
	ResolvedAt     *time.Time    // Время закрытия инцидента
	ResolvedBy     *int          // ID пользователя, закрывшего инцидент (nil при авто-закрытии системой)
}

// OfflineVehicleInfo содержит информацию об автомобиле, потерявшем связь с трекером
type OfflineVehicleInfo struct {
	VehicleID      int        // ID офлайн машины
	OrganizationID int        // Какой организации она принадлежит
	DeviceID       int        // Номер девайса, который замолчал
	LastSeen       *time.Time // Когда была последняя телеметрия от трекера
	MinutesOffline float64    // Сколько минут нет связи
}
