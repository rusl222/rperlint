package scada

import (
	"sync"
	"time"
)

type Vartype interface {
	~int | ~float64 | time.Time
}

type Api interface {
	Connected() bool
	Get(reper string, pValue any) error
	Set(reper string, value any) error
}

type Var[T Vartype] struct {
	reper string
	api   Api

	mu    sync.RWMutex
	value T
}

func (v *Var[T]) Bind(reper string, api Api) {
	v.mu.Lock()
	v.reper = reper
	v.api = api
	v.api.Get(reper, &v.value)
	v.mu.Unlock()
}

type Other struct{}

func (o *Other) Bind(reper string, api Api) {}
