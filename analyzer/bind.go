package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/rusl222/scada/reperdb"

	"golang.org/x/tools/go/analysis"
)

func checkBindCall(
	pass *analysis.Pass,
	call *ast.CallExpr,
	cfg Config,
	repo reperdb.Repository,
) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	if selector.Sel.Name != cfg.BindMethodName {
		return
	}

	// Resolve the method through go/types.
	obj := pass.TypesInfo.Uses[selector.Sel]

	fn, ok := obj.(*types.Func)
	if !ok {
		return
	}

	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return
	}

	receiver := sig.Recv()
	if receiver == nil {
		return
	}

	// This is the important check:
	//
	//     *scada.Var[T]
	//
	// rather than just:
	//
	//     something.Bind(...)
	if !isScadaVar(receiver.Type(), cfg) {
		return
	}

	if len(call.Args) == 0 {
		return
	}

	reperExpr := call.Args[0]

	reper, ok := constantString(pass.TypesInfo, reperExpr)
	if !ok {
		pass.Reportf(
			reperExpr.Pos(),
			"reper must be a string constant",
		)

		return
	}

	if !repo.Contains(reper) {
		pass.Reportf(
			reperExpr.Pos(),
			"unknown reper %q",
			reper,
		)
	}
}

func isScadaVar(t types.Type, cfg Config) bool {
	// Bind is declared on *Var[T].
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	named, ok := t.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil {
		return false
	}

	if obj.Name() != cfg.VarTypeName {
		return false
	}

	pkg := obj.Pkg()
	if pkg == nil {
		return false
	}

	return pkg.Path() == cfg.VarPackagePath
}

func constantString(
	info *types.Info,
	expr ast.Expr,
) (string, bool) {
	tv, ok := info.Types[expr]
	if !ok {
		return "", false
	}

	if tv.Value == nil {
		return "", false
	}

	if tv.Type == nil {
		return "", false
	}

	// Ensure the expression has a basic string type.
	if basic, ok := tv.Type.Underlying().(*types.Basic); !ok || basic.Kind() != types.String {
		return "", false
	}

	return tv.Value.String(), true
}
