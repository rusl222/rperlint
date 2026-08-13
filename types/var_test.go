package scada

import (
	"math"
	"testing"
	"time"
)

func TestVarIntWithoutApi(t *testing.T) {
	var v Var[int]
	v.Set(42)

	if got := v.Get(); got != 42 {
		t.Fatalf("value 42. Get() = %d, want %d", got, 42)
	}

	if got := v.Valid(); !got {
		t.Fatalf("value 42. Valid() = %v, want %v", got, true)
	}

	v.Set(-1)
	if got := v.Get(); got != -1 {
		t.Fatalf("value -1. Get() = %d, want %d", got, -1)
	}

	if got := v.Valid(); got {
		t.Fatalf("value -1. Valid() = %v, want %v", got, false)
	}
}

func TestVarFloatWithoutApi(t *testing.T) {
	var v Var[float64]
	v.Set(42.0)

	if got := v.Get(); got != 42.0 {
		t.Fatalf("value 42.0 Get() = %f, want %f", got, 42.0)
	}

	if got := v.Valid(); !got {
		t.Fatalf("value 42.0 Valid() = %v, want %v", got, true)
	}

	v.Set(math.NaN())
	if got := v.Get(); !math.IsNaN(got) {
		t.Fatalf("value NaN. Get() = %f, want %f", got, math.NaN())
	}

	if got := v.Valid(); got {
		t.Fatalf("value NaN. Valid() = %v, want %v", got, false)
	}
}

func TestVarTimeWithoutApi(t *testing.T) {
	var v Var[time.Time]
	now := time.Now()
	v.Set(now)

	if got := v.Get(); !got.Equal(now) {
		t.Fatalf("value now. Get() = %v, want %v", got, now)
	}

	if got := v.Valid(); !got {
		t.Fatalf("value now. Valid() = %v, want %v", got, true)
	}

	v.Set(time.Time{})
	if got := v.Get(); !got.Equal(time.Time{}) {
		t.Fatalf("value zero time. Get() = %v, want %v", got, time.Time{})
	}

	if got := v.Valid(); got {
		t.Fatalf("value zero time. Valid() = %v, want %v", got, false)
	}
}
