package debug

import (
	"errors"
	"log/slog"
	"reflect"
	"sync"
)

type Mock struct {
	mu        sync.RWMutex
	data      map[string]any
	connected bool

	log *slog.Logger
}

func NewMock(logger *slog.Logger) *Mock {
	return &Mock{
		data:      make(map[string]any),
		connected: true,
		log:       logger,
	}
}

func (m *Mock) Connected() bool {
	return m.connected
}

func (m *Mock) Connect() {}

func (m *Mock) Get(reper string, pValue any) error {
	rv := reflect.ValueOf(pValue)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("pValue must be non-nil pointer")
	}

	elemType := rv.Elem().Type()

	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.data[reper]
	if !ok {
		// store the concrete zero value (not a reflect.Value) so later
		// reflection can match the expected element type.
		zv := reflect.Zero(elemType).Interface()
		val = zv
		m.data[reper] = val
	}

	v := reflect.ValueOf(val)

	if !v.IsValid() {
		rv.Elem().Set(reflect.Zero(elemType))
		return nil
	}

	if v.Type().AssignableTo(elemType) {
		rv.Elem().Set(v)
		return nil
	}

	if v.Type().ConvertibleTo(elemType) {
		rv.Elem().Set(v.Convert(elemType))
		return nil
	}

	return errors.New("type mismatch")
}

func (m *Mock) Set(reper string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[reper] = value
	return nil
}
