// Package dto содержит тесты DTO правил алертов
package dto

import (
	"fleettrack/internal/model"
	"testing"
	"time"
)

// TestCreateAlertRuleRequest_ToDomainModel проверяет корректность конвертации DTO в доменную модель
func TestCreateAlertRuleRequest_ToDomainModel(t *testing.T) {
	enabledFalse := false
	enabledTrue := true

	tests := []struct {
		name        string
		req         CreateAlertRuleRequest
		wantEnabled bool
	}{
		{
			name: "enabled по умолчанию true когда nil",
			req: CreateAlertRuleRequest{
				OrganizationID: 10,
				Type:           model.AlertRuleSpeedExceed,
				Name:           "Превышение скорости",
				Threshold:      90,
				Severity:       model.SeverityLevelHigh,
				Enabled:        nil,
			},
			wantEnabled: true,
		},
		{
			name: "enabled явно передан false",
			req: CreateAlertRuleRequest{
				OrganizationID: 10,
				Type:           model.AlertRuleLowFuel,
				Name:           "Низкое топливо",
				Threshold:      15,
				Severity:       model.SeverityLevelMedium,
				Enabled:        &enabledFalse,
			},
			wantEnabled: false,
		},
		{
			name: "enabled явно передан true",
			req: CreateAlertRuleRequest{
				OrganizationID: 10,
				Type:           model.AlertRuleSpeedExceed,
				Name:           "Превышение",
				Threshold:      100,
				Severity:       model.SeverityLevelCritical,
				Enabled:        &enabledTrue,
			},
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.ToDomainModel()

			if got.OrganizationID != tt.req.OrganizationID {
				t.Errorf("OrganizationID = %d, want %d", got.OrganizationID, tt.req.OrganizationID)
			}
			if got.Type != tt.req.Type {
				t.Errorf("Type = %s, want %s", got.Type, tt.req.Type)
			}
			if got.Name != tt.req.Name {
				t.Errorf("Name = %s, want %s", got.Name, tt.req.Name)
			}
			if got.Threshold != tt.req.Threshold {
				t.Errorf("Threshold = %f, want %f", got.Threshold, tt.req.Threshold)
			}
			if got.Severity != tt.req.Severity {
				t.Errorf("Severity = %s, want %s", got.Severity, tt.req.Severity)
			}
			if got.Enabled != tt.wantEnabled {
				t.Errorf("Enabled = %v, want %v", got.Enabled, tt.wantEnabled)
			}
		})
	}
}

// TestNewAlertRuleResponse проверяет корректность конвертации доменной модели в ответ API
func TestNewAlertRuleResponse(t *testing.T) {
	now := time.Now()
	rule := model.AlertRule{
		ID:             1,
		OrganizationID: 2,
		Type:           model.AlertRuleSpeedExceed,
		Name:           "Быстрая езда",
		Threshold:      80,
		Severity:       model.SeverityLevelHigh,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      &now,
	}

	res := NewAlertRuleResponse(rule)

	if res.ID != rule.ID || res.OrganizationID != rule.OrganizationID || res.Name != rule.Name {
		t.Errorf("некорректная конвертация полей ответа: got %+v, want %+v", res, rule)
	}

	list := NewAlertRuleListResponse([]model.AlertRule{rule})
	if len(list) != 1 {
		t.Fatalf("длина списка ответов = %d, ожидалась 1", len(list))
	}
	if list[0].ID != rule.ID {
		t.Errorf("ID элемента списка = %d, ожидался %d", list[0].ID, rule.ID)
	}
}
