package gui

import (
	"math"
	"testing"
)

func TestProject(t *testing.T) {
	cases := []struct {
		name         string
		lat, lon     float64
		w, h         float32
		wantX, wantY float32
	}{
		{"top-left antimeridian-ish", 90, -180, 1000, 500, 0, 0},
		{"bottom-right antimeridian", -90, 180, 1000, 500, 1000, 500},
		{"prime meridian equator", 0, 0, 1000, 500, 500, 250},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			x, y := projectLatLon(c.lat, c.lon, c.w, c.h)
			if math.Abs(float64(x-c.wantX)) > 0.5 || math.Abs(float64(y-c.wantY)) > 0.5 {
				t.Errorf("projectLatLon(%v,%v) = (%v,%v), want (%v,%v)",
					c.lat, c.lon, x, y, c.wantX, c.wantY)
			}
		})
	}
}
