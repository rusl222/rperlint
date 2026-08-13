package debug

import (
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"sync"
	"time"

	scada "github.com/rusl222/scada/types"
)

type Entry struct {
	Name string `json:"name"`
	Type string `json:"type"`

	Value   any       `json:"value"`
	Updated time.Time `json:"updated"`

	Favorite bool `json:"favorite"`
}

func (e Entry) MarshalJSON() ([]byte, error) {
	type Alias Entry

	a := Alias(e)

	switch v := a.Value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			a.Value = nil
		}
	case float32:
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			a.Value = nil
		}
	case *float64:
		if math.IsNaN(*v) || math.IsInf(*v, 0) {
			a.Value = nil
		}
	case *float32:
		if math.IsNaN(float64(*v)) || math.IsInf(float64(*v), 0) {
			a.Value = nil
		}
	}

	return json.Marshal(a)
}

type Proxy struct {
	scada.Api

	mu sync.RWMutex

	entries map[string]*Entry
}

func Wrap(a scada.Api) *Proxy {

	return &Proxy{
		Api:     a,
		entries: make(map[string]*Entry),
	}
}

func (p *Proxy) update(name string, typ ValueType, value any) {

	p.mu.Lock()
	defer p.mu.Unlock()

	e, ok := p.entries[name]
	if !ok {
		e = &Entry{
			Name: name,
			Type: typ.String(),
		}
		p.entries[name] = e
	}

	p.entries[name].Value = value
	p.entries[name].Updated = time.Now()
}

func (p *Proxy) Type(name string) (ValueType, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	e, ok := p.entries[name]
	if !ok {
		return 0, false
	}

	switch e.Type {
	case TypeInt.String():
		return TypeInt, true
	case TypeBool.String():
		return TypeBool, true
	case TypeFloat.String():
		return TypeFloat, true
	default:
		return TypeFloat, true
	}
}

func (p *Proxy) Get(name string, pValue any) error {
	copy := reflect.ValueOf(pValue)
	if copy.Kind() != reflect.Pointer || copy.IsNil() {
		return nil
	}

	err := p.Api.Get(name, pValue)
	if err == nil {
		p.update(name, typeOf(pValue), pValue)
	}

	return err
}
func (p *Proxy) Set(name string, value any) error {

	err := p.Api.Set(name, value)
	p.update(name, typeOf(value), value)

	return err
}

func typeOf(pValue any) ValueType {
	switch pValue.(type) {
	case *float64, float64:
		return TypeFloat
	case *int, int:
		return TypeInt
	case *time.Time, time.Time:
		return Datetime
	case *bool, bool:
		return TypeBool
	default:
		return TypeUnknown
	}
}

func (p *Proxy) Snapshot() []Entry {

	p.mu.RLock()

	defer p.mu.RUnlock()

	out := make([]Entry, 0, len(p.entries))

	for _, e := range p.entries {

		out = append(out, *e)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})

	return out
}
