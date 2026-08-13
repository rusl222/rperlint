package debug

import (
	"testing"

	"github.com/rusl222/scada/logger"
)

func TestSetAndGetAssignable(t *testing.T) {
	m := NewMock(logger.Logger(true))
	if err := m.Set("k", 42); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	var v int
	if err := m.Get("k", &v); err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

func TestGetCreatesZeroWhenMissing(t *testing.T) {
	m := NewMock(logger.Logger(true))

	var s string
	if err := m.Get("missing", &s); err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if s != "" {
		t.Fatalf("expected zero string, got %q", s)
	}

	// ensure the key was stored with zero value
	if _, ok := m.data["missing"]; !ok {
		t.Fatalf("expected key 'missing' to be created in map")
	}
}

func TestTypeMismatch(t *testing.T) {
	m := NewMock(logger.Logger(true))
	if err := m.Set("k", "str"); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	var v int
	if err := m.Get("k", &v); err == nil {
		t.Fatalf("expected type mismatch error, got nil")
	}
}

func TestConvertibleTypes(t *testing.T) {
	m := NewMock(logger.Logger(true))
	if err := m.Set("k", int32(100)); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	var v int64
	if err := m.Get("k", &v); err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if v != 100 {
		t.Fatalf("expected 100, got %d", v)
	}
}

func TestGetRequiresNonNilPointer(t *testing.T) {
	m := NewMock(logger.Logger(true))

	// non-pointer value
	if err := m.Get("k", 5); err == nil {
		t.Fatalf("expected error for non-pointer pValue, got nil")
	}

	// nil pointer
	var p *int
	if err := m.Get("k", p); err == nil {
		t.Fatalf("expected error for nil pointer pValue, got nil")
	}
}
