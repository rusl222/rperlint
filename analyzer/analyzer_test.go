package analyzer

import (
	"testing"

	"github.com/rusl222/scada/reperdb"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()

	cfg := Config{
		VarPackagePath: "scada",
		VarTypeName:    "Var",
		BindMethodName: "Bind",

		ReperRepository: testRepository{
			"ABC":         {},
			"DEF":         {},
			"1TF QСТ КУШ": {},
		},
	}

	analyzer := NewAnalyzer(cfg)

	analysistest.Run(
		t,
		testdata,
		analyzer,
		"example",
	)
}

type testRepository map[string]struct{}

func (r testRepository) Contains(value string) bool {
	_, ok := r[value]
	return ok
}

func (r testRepository) Count() int {
	return len(r)
}

var _ reperdb.Repository = testRepository{}
