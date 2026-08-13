package analyzer

import "github.com/rusl222/scada/reperdb"

type testRepository map[string]struct{}

func (r testRepository) Contains(value string) bool {
	_, ok := r[value]
	return ok
}

func (r testRepository) Count() int {
	return len(r)
}

var _ reperdb.Repository = testRepository{}
