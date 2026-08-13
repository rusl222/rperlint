package analyzer

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()

	cfg := Config{
		VarPackagePath: "github.com/rusl222/scada/analyzer/testdata/src/scada",
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
