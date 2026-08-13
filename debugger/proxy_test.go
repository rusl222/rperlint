package debug

import (
	"reflect"
	"testing"
)

type proxyTestApi struct {
	value any
}

func (a *proxyTestApi) Connected() bool { return true }

func (a *proxyTestApi) Get(reper string, pValue any) error {
	rv := reflect.ValueOf(pValue)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return nil
	}

	if rv.Elem().Type() != reflect.TypeOf(0) {
		return nil
	}

	rv.Elem().Set(reflect.ValueOf(a.value))
	return nil
}

func (a *proxyTestApi) Set(reper string, value any) error { return nil }

func TestProxyGetPassesConcreteTargetTypeToApi(t *testing.T) {
	api := &proxyTestApi{value: 42}
	p := Wrap(api)

	var got int
	if err := p.Get("x", &got); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if got != 42 {
		t.Fatalf("expected value 42, got %d", got)
	}
}
