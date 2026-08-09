package translate

import (
	"go/token"

	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

func (t *packageTranslator) go126HashTrieMapReplacement(expr dst.Expr) (dst.Expr, bool) {
	if t.pkgPath != "internal/sync" {
		return nil, false
	}

	selector, ok := expr.(*dst.SelectorExpr)
	if !ok {
		return nil, false
	}

	replacement := ""
	switch selector.Sel.Name {
	case "Hasher":
		if ident, ok := selector.X.(*dst.Ident); ok && ident.Name == "mapType" {
			replacement = "MapHasher"
		}
	case "Equal":
		if elem, ok := selector.X.(*dst.SelectorExpr); ok && elem.Sel.Name == "Elem" {
			if ident, ok := elem.X.(*dst.Ident); ok && ident.Name == "mapType" {
				replacement = "MapValueEqual"
			}
		}
	}
	if replacement == "" {
		return nil, false
	}

	return &dst.CallExpr{
		Fun:  t.newRuntimeSelector(replacement),
		Args: []dst.Expr{dst.NewIdent("m")},
	}, true
}

func (t *packageTranslator) isGo126HashTrieMapTypeAssignment(stmt *dst.AssignStmt) bool {
	if t.pkgPath != "internal/sync" || stmt.Tok != token.DEFINE || len(stmt.Lhs) != 1 {
		return false
	}
	ident, ok := stmt.Lhs[0].(*dst.Ident)
	return ok && ident.Name == "mapType"
}

func (t *packageTranslator) rewriteGo126HashTrieMap(c *dstutil.Cursor) {
	if assignment, ok := c.Node().(*dst.AssignStmt); ok && t.isGo126HashTrieMapTypeAssignment(assignment) {
		c.Delete()
		return
	}
	expr, ok := c.Node().(dst.Expr)
	if !ok {
		return
	}
	if replacement, ok := t.go126HashTrieMapReplacement(expr); ok {
		c.Replace(replacement)
	}
}
