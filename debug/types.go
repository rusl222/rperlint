package debug

import "sync"

type ValueType uint8

const (
	TypeFloat ValueType = iota
	TypeInt
	Datetime
	TypeBool
	TypeUnknown
)

func (t ValueType) String() string {
	switch t {
	case TypeFloat:
		return "float"
	case TypeInt:
		return "int"
	case TypeBool:
		return "bool"
	case Datetime:
		return "datetime"
	default:
		return "unknown"
	}
}

type Reper struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

type registry struct {
	mu sync.RWMutex

	repers map[string]ValueType
}

func newRegistry() *registry {
	return &registry{
		repers: make(map[string]ValueType),
	}
}

func (r *registry) add(name string, t ValueType) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.repers[name]; !ok {
		r.repers[name] = t
	}
}

func (r *registry) list() map[string]ValueType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[string]ValueType, len(r.repers))

	for k, v := range r.repers {
		out[k] = v
	}

	return out
}
