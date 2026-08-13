package scada

import (
	"math"
	"reflect"
	"sync"
	"time"
)

type Vartype interface {
	~int | ~float64 | time.Time
}

// Api defines transport/storage operations. Methods accept context and return errors.
type Api interface {
	Connected() bool
	// pValue must be a pointer to a value where result will be written
	Get(reper string, pValue any) error
	Set(reper string, value any) error
}

type Var[T Vartype] struct {
	reper string
	api   Api

	mu    sync.RWMutex
	value T
}

// Bind associates variable with external api and reper identifier.
func (v *Var[T]) Bind(reper string, api Api) {
	v.mu.Lock()
	v.reper = reper
	v.api = api
	v.api.Get(reper, &v.value)
	v.mu.Unlock()
}

// Value returns the locally cached value without contacting Api.
func (v *Var[T]) Get() T {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.api != nil {
		v.api.Get(v.reper, &v.value)
	}
	return v.value
}

// Valid reports whether the last read/set was successful and value is reliable.
func (v *Var[T]) Valid() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.api != nil {
		err := v.api.Get(v.reper, &v.value)
		if err != nil {
			return false
		}
	}
	return validValue(v.value)
}

func validValue[T Vartype](val T) bool {
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Float64:
		return !math.IsNaN(rv.Float())
	case reflect.Int:
		return rv.Int() > -1
	case reflect.Struct:
		timeType := reflect.TypeOf(time.Time{})
		if rv.Type().ConvertibleTo(timeType) {
			t := rv.Convert(timeType).Interface().(time.Time)
			return !t.IsZero()
		}
	}
	return false
}

func (v *Var[T]) Set(val T) {

	if v.api != nil {
		v.api.Set(v.reper, val)
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.value = val
}

func (v *Var[T]) SetValid(val T, valid bool) {
	corrected := correctValidValue(val, valid)
	v.Set(corrected)
}

func correctValidValue[T Vartype](val T, valid bool) T {
	if !valid {
		rv := reflect.ValueOf(val)
		switch rv.Kind() {
		case reflect.Float64:
			out := reflect.New(rv.Type()).Elem()
			out.SetFloat(math.NaN())
			return out.Interface().(T)
		case reflect.Int:
			out := reflect.New(rv.Type()).Elem()
			out.SetInt(-1)
			return out.Interface().(T)
		default:
			return reflect.Zero(rv.Type()).Interface().(T)
		}
	}
	return val
}
