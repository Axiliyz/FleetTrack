package validator

import (
	"fleettrack/internal/model"
	"strings"
	"testing"
)

func TestValidateDriver(t *testing.T) {
	valid := func() model.Driver {
		return model.Driver{
			OrganizationID: 1,
			Name:           "Ivan Petrov",
		}
	}

	tests := []struct {
		name    string
		mutate  func(d *model.Driver)
		wantErr error
	}{
		{name: "valid", mutate: func(d *model.Driver) {}, wantErr: nil},
		{name: "invalid organization id", mutate: func(d *model.Driver) { d.OrganizationID = 0 }, wantErr: model.ErrInvalidOrganizationID},
		{name: "empty name", mutate: func(d *model.Driver) { d.Name = "" }, wantErr: model.ErrInvalidDriverName},
		{name: "blank name", mutate: func(d *model.Driver) { d.Name = "   " }, wantErr: model.ErrInvalidDriverName},
		{name: "name too long", mutate: func(d *model.Driver) { d.Name = strings.Repeat("a", 36) }, wantErr: model.ErrInvalidDriverName},
		{name: "name at max length", mutate: func(d *model.Driver) { d.Name = strings.Repeat("a", 35) }, wantErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := valid()
			tt.mutate(&d)
			err := ValidateDriver(d)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}
