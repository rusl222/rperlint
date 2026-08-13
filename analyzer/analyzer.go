package analyzer

import (
	"fmt"
	"go/ast"

	"github.com/rusl222/scada/reperdb"

	"golang.org/x/tools/go/analysis"
)

// Config configures the reper analyzer.
type Config struct {
	// Full import path of package containing Var.
	VarPackagePath string

	// Name of Var type.
	VarTypeName string

	// Name of Bind method.
	BindMethodName string

	// Repository containing known reper values.
	ReperRepository reperdb.Repository
}

// NewAnalyzer creates reperlint analyzer.
func NewAnalyzer(cfg Config) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "reperlint",
		Doc:  "checks reper values used by scada.Var.Bind",
		Run: func(pass *analysis.Pass) (any, error) {
			return run(pass, cfg)
		},
	}
}

func run(
	pass *analysis.Pass,
	cfg Config,
) (any, error) {
	if cfg.ReperRepository == nil {
		return nil, fmt.Errorf("reper repository is not configured")
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			checkBindCall(
				pass,
				call,
				cfg,
				cfg.ReperRepository,
			)

			return true
		})
	}

	return nil, nil
}
