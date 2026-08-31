package service

import (
	"fleettrack/internal/model"
	"math"
	"testing"
	"time"
)

func TestMotionServiceCalculate(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		last    *model.Telemetry
		cur     model.Telemetry
		wantErr error
	}{
		{
			name: "valid",
			last: &model.Telemetry{
				Lat:             55.75,
				Lon:             37.61,
				DeviceTimestamp: base,
			},
			cur: model.Telemetry{
				Lat:             55.76,
				Lon:             37.62,
				DeviceTimestamp: base.Add(time.Hour),
			},
			wantErr: nil,
		},
		{
			name: "no last value",
			last: nil,
			cur: model.Telemetry{
				Lat:             55.75,
				Lon:             37.61,
				DeviceTimestamp: base,
			},
			wantErr: model.ErrNoValue,
		},
		{
			name: "cur before last",
			last: &model.Telemetry{
				Lat:             55.75,
				Lon:             37.61,
				DeviceTimestamp: base,
			},
			cur: model.Telemetry{
				Lat:             55.76,
				Lon:             37.62,
				DeviceTimestamp: base.Add(-time.Hour),
			},
			wantErr: model.ErrInvalidTime,
		},
		{
			name: "cur equals last time",
			last: &model.Telemetry{
				Lat:             55.75,
				Lon:             37.61,
				DeviceTimestamp: base,
			},
			cur: model.Telemetry{
				Lat:             55.76,
				Lon:             37.62,
				DeviceTimestamp: base,
			},
			wantErr: model.ErrInvalidTime,
		},
		{
			name: "invalid last coords",
			last: &model.Telemetry{
				Lat:             255.75,
				Lon:             37.61,
				DeviceTimestamp: base,
			},
			cur: model.Telemetry{
				Lat:             55.76,
				Lon:             37.62,
				DeviceTimestamp: base.Add(time.Hour),
			},
			wantErr: model.ErrInvalidCoords,
		},
		{
			name: "invalid cur coords",
			last: &model.Telemetry{
				Lat:             55.75,
				Lon:             37.61,
				DeviceTimestamp: base,
			},
			cur: model.Telemetry{
				Lat:             55.76,
				Lon:             317.62,
				DeviceTimestamp: base.Add(time.Hour),
			},
			wantErr: model.ErrInvalidCoords,
		},
		{
			name: "edge coords (lat=90, lon=180)",
			last: &model.Telemetry{
				Lat:             90,
				Lon:             180,
				DeviceTimestamp: base,
			},
			cur: model.Telemetry{
				Lat:             90,
				Lon:             180,
				DeviceTimestamp: base.Add(time.Hour),
			},
			wantErr: nil,
		},
		{
			name: "same point, no distance",
			last: &model.Telemetry{
				Lat:             55.75,
				Lon:             37.61,
				DeviceTimestamp: base,
			},
			cur: model.Telemetry{
				Lat:             55.75,
				Lon:             37.61,
				DeviceTimestamp: base.Add(time.Hour),
			},
			wantErr: nil,
		},
	}

	s := &MotionServiceImpl{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Calculate(tt.last, tt.cur)
			if err != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestMotionServiceCalculateResult(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	last := &model.Telemetry{
		Lat:             0,
		Lon:             0,
		DeviceTimestamp: base,
	}
	cur := model.Telemetry{
		Lat:             0,
		Lon:             1,
		DeviceTimestamp: base.Add(time.Hour),
	}

	s := &MotionServiceImpl{}
	res, err := s.Calculate(last, cur)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantDistance := CalculateDistance(*last, cur)
	if math.Abs(res.DistanceKm-wantDistance) > 1e-9 {
		t.Errorf("got distance %f, want %f", res.DistanceKm, wantDistance)
	}

	wantSpeed := wantDistance / cur.DeviceTimestamp.Sub(last.DeviceTimestamp).Hours()
	if math.Abs(res.SpeedKmh-wantSpeed) > 1e-9 {
		t.Errorf("got speed %f, want %f", res.SpeedKmh, wantSpeed)
	}
}

func TestValidateTime(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		last time.Time
		cur  time.Time
		want bool
	}{
		{
			name: "cur after last",
			last: base,
			cur:  base.Add(time.Second),
			want: true,
		},
		{
			name: "cur before last",
			last: base,
			cur:  base.Add(-time.Second),
			want: false,
		},
		{
			name: "cur equals last",
			last: base,
			cur:  base,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateTime(tt.last, tt.cur)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateCoords(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lon  float64
		want bool
	}{
		{name: "valid", lat: 55.75, lon: 37.61, want: true},
		{name: "edge lat=-90", lat: -90, lon: 0, want: true},
		{name: "edge lat=90", lat: 90, lon: 0, want: true},
		{name: "edge lon=-180", lat: 0, lon: -180, want: true},
		{name: "edge lon=180", lat: 0, lon: 180, want: true},
		{name: "invalid lat", lat: 255.75, lon: 37.61, want: false},
		{name: "invalid lon", lat: 55.75, lon: 317.61, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateCoords(tt.lat, tt.lon)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateDistance(t *testing.T) {
	last := model.Telemetry{Lat: 0, Lon: 0}
	cur := model.Telemetry{Lat: 0, Lon: 0}

	if d := CalculateDistance(last, cur); math.Abs(d) > 1e-9 {
		t.Errorf("distance between identical points should be 0, got %f", d)
	}

	cur = model.Telemetry{Lat: 0, Lon: 1}
	// 1 degree of longitude on the equator is approximately 111.19 km
	want := 111.19
	if d := CalculateDistance(last, cur); math.Abs(d-want) > 0.5 {
		t.Errorf("got distance %f, want approximately %f", d, want)
	}
}
