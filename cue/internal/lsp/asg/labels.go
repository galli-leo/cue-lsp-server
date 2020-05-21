package asg

import "cuelang.org/go/cue/ast"

func ResolveLabels(idx *index, expr ast.Expr) []string {
	ret := []string{}
	switch n := expr.(type) {
	case *ast.SelectorExpr:
		lhs, rhs := n.X, n.Sel
		label, _, err := ast.LabelName(rhs)
		idx.addErr(err)
		ret = ResolveLabels(idx, lhs)
		ret = append(ret, label)
	case *ast.Ident:
		label, _, err := ast.LabelName(n)
		idx.addErr(err)
		ret = []string{label}
	}
	return ret
}
