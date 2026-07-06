package validator

import (
	"fleettrack/internal/model"
	"testing"
)

func TestValidateVehicle(t *testing.T) {
	valid := func() model.Vehicle {
		return model.Vehicle{
			OrganizationID: 1,
			VIN:            "1HGCM82633A123456",
			NumberPlate:    "A123BC77",
			Model:          "Toyota Camry",
			Status:         model.VehicleStatusIdle,
		}
	}

	tests := []struct {
		name    string
		mutate  func(v *model.Vehicle)
		wantErr error
	}{
		{
			name:    "valid",
			mutate:  func(v *model.Vehicle) {},
			wantErr: nil,
		},
		{
			name:    "vin too short",
			mutate:  func(v *model.Vehicle) { v.VIN = "SHORT" },
			wantErr: model.ErrInvalidVIN,
		},
		{
			name:    "vin lowercase",
			mutate:  func(v *model.Vehicle) { v.VIN = "1hgcm82633a123456" },
			wantErr: model.ErrInvalidVIN,
		},
		{
			name:    "number plate too short",
			mutate:  func(v *model.Vehicle) { v.NumberPlate = "A123B" },
			wantErr: model.ErrInvalidNumberPlate,
		},
		{
			name:    "number plate too long",
			mutate:  func(v *model.Vehicle) { v.NumberPlate = "A123BC7789" },
			wantErr: model.ErrInvalidNumberPlate,
		},
		{
			name:    "edge number plate (8 chars)",
			mutate:  func(v *model.Vehicle) { v.NumberPlate = "A123BC77" },
			wantErr: nil,
		},
		{
			name:    "edge number plate (9 chars)",
			mutate:  func(v *model.Vehicle) { v.NumberPlate = "A123BC777" },
			wantErr: nil,
		},
		{
			name:    "invalid organization id",
			mutate:  func(v *model.Vehicle) { v.OrganizationID = 0 },
			wantErr: model.ErrInvalidOrganizationID,
		},
		{
			name:    "empty model",
			mutate:  func(v *model.Vehicle) { v.Model = "   " },
			wantErr: model.ErrInvalidModel,
		},
		{
			name:    "invalid status",
			mutate:  func(v *model.Vehicle) { v.Status = "UNKNOWN" },
			wantErr: model.ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := valid()
			tt.mutate(&v)
			err := ValidateVehicle(v)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}
