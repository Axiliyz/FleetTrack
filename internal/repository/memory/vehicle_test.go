package memory

import (
	"context"
	"fleettrack/internal/model"
	"testing"
)

func TestMemoryVehicleRepository_CreateAndGetByID(t *testing.T) {
	repo := NewMemoryVehicleRepository()
	v := model.Vehicle{OrganizationID: 1, VIN: "1HGCM82633A123456", NumberPlate: "A123BC77", Model: "Camry", Status: model.VehicleStatusIdle}

	if err := repo.Create(context.Background(), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID == 0 {
		t.Fatalf("expected Create to assign a non-zero ID")
	}
	if v.CreatedAt.IsZero() {
		t.Fatalf("expected Create to set CreatedAt")
	}

	got, err := repo.GetByID(context.Background(), v.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.VIN != v.VIN {
		t.Errorf("got VIN %v, want %v", got.VIN, v.VIN)
	}

	_, err = repo.GetByID(context.Background(), 9999)
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

func TestMemoryVehicleRepository_Delete(t *testing.T) {
	repo := NewMemoryVehicleRepository()
	v := model.Vehicle{OrganizationID: 1, VIN: "1HGCM82633A123456", NumberPlate: "A123BC77", Model: "Camry", Status: model.VehicleStatusIdle}
	repo.Create(context.Background(), &v)

	deleted, err := repo.Delete(context.Background(), v.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted.Status != model.VehicleStatusDeleted {
		t.Errorf("got status %v, want %v", deleted.Status, model.VehicleStatusDeleted)
	}

	// deletion must persist, not just mutate a copy
	got, err := repo.GetByID(context.Background(), v.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != model.VehicleStatusDeleted {
		t.Errorf("persisted status = %v, want %v", got.Status, model.VehicleStatusDeleted)
	}

	_, err = repo.Delete(context.Background(), 9999)
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

func TestMemoryVehicleRepository_Update(t *testing.T) {
	repo := NewMemoryVehicleRepository()
	v := model.Vehicle{OrganizationID: 1, VIN: "1HGCM82633A123456", NumberPlate: "A123BC77", Model: "Camry", Status: model.VehicleStatusIdle}
	repo.Create(context.Background(), &v)

	newPlate := "B999XY77"
	updated, err := repo.Update(context.Background(), v.ID, model.UpdateVehicle{NumberPlate: &newPlate})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.NumberPlate != newPlate {
		t.Errorf("got NumberPlate %v, want %v", updated.NumberPlate, newPlate)
	}
	if updated.UpdatedAt == nil {
		t.Errorf("expected UpdatedAt to be set")
	}

	_, err = repo.Update(context.Background(), 9999, model.UpdateVehicle{NumberPlate: &newPlate})
	if err != model.ErrNotFound {
		t.Errorf("got %v, want %v", err, model.ErrNotFound)
	}
}

func TestMemoryVehicleRepository_GetList(t *testing.T) {
	repo := NewMemoryVehicleRepository()
	for i := 0; i < 3; i++ {
		v := model.Vehicle{OrganizationID: 1, VIN: "1HGCM82633A123456", NumberPlate: "A123BC77", Model: "Camry", Status: model.VehicleStatusIdle}
		repo.Create(context.Background(), &v)
	}
	v := model.Vehicle{OrganizationID: 2, VIN: "2HGCM82633A123456", NumberPlate: "B999XY77", Model: "Corolla", Status: model.VehicleStatusIdle}
	repo.Create(context.Background(), &v)

	org1 := 1
	list, err := repo.GetList(context.Background(), model.VehicleFilter{OrganizationID: &org1, Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("got %d vehicles, want 3", len(list))
	}
}
