package archcheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"
	"strings"
)

// HostSite is one host-effect site in a selected host file: a reference to a
// host clock, timer or deadline entry point, a host entropy source, a
// goroutine start, a range over a map (directly or through maps.Keys,
// maps.Values or maps.All), whose order is randomized, or an unstable sort,
// whose result depends on input order when keys compare equal. Reviewers
// dispose of each site in an inventory instead of exempting whole files; the
// pure set keeps the stricter host-effect analysis in calls.go.
type HostSite struct {
	File, Function, Kind, Symbol string
	Count                        int
}

// Key identifies the site independently of line numbers.
func (s HostSite) Key() string {
	return strings.Join([]string{s.File, s.Function, s.Kind, s.Symbol}, "|")
}

// hostClockFunctions are the time package entry points that read the host
// clock or arm a host timer.
var hostClockFunctions = []string{"Now", "Since", "Until", "Sleep", "After", "AfterFunc", "NewTimer", "NewTicker", "Tick"}

// hostClockMethods are timer methods that re-arm a host timer.
var hostClockMethods = []string{"Reset"}

// hostDeadlineFunctions derive a context whose deadline is host time.
var hostDeadlineFunctions = []string{"WithTimeout", "WithDeadline", "WithTimeoutCause", "WithDeadlineCause"}

// entropyPackages are host entropy sources; every package-level function
// reference counts.
var entropyPackages = []string{"math/rand", "math/rand/v2", "crypto/rand"}

// mapIterators range over a map in its randomized order.
var mapIterators = []string{"Keys", "Values", "All"}

// unstableSorts are the sort entry points whose result order for equal keys
// depends on the input order.
var unstableSorts = map[string][]string{"sort": {"Slice", "Sort"}, "slices": {"SortFunc"}}

// HostEffectSites type-checks the packages holding the selected files with the
// purity analyzer's loader and reports every host clock, deadline and entropy
// reference, goroutine start, map range and unstable sort in those files,
// aggregated by file, enclosing function, kind and symbol. Function values
// count as well as calls, so `f := time.Now` is a site.
func HostEffectSites(root, goCommand string, platform Platform, selected func(string) bool) ([]HostSite, error) {
	p, err := load(root, goCommand, platform, selected)
	if err != nil {
		return nil, err
	}
	return p.hostSites(), nil
}

func (p *program) hostSites() []HostSite {
	counts := map[HostSite]int{}
	for _, path := range sortedKeys(p.packages) {
		pkg := p.packages[path]
		for _, file := range pkg.files {
			relative := p.relative(p.fset.Position(file.Pos()).Filename)
			if !p.pure(relative) {
				continue
			}
			for _, declaration := range file.Decls {
				function := declarationName(declaration)
				ast.Inspect(declaration, func(node ast.Node) bool {
					site := HostSite{File: relative, Function: function}
					switch node := node.(type) {
					case *ast.GoStmt:
						site.Kind, site.Symbol = "goroutine", goTarget(pkg.info, node.Call)
					case *ast.SelectorExpr:
						if name, ok := packageFunction(pkg.info, node, "time"); ok && slices.Contains(hostClockFunctions, name) {
							site.Kind, site.Symbol = "clock", "time."+name
						} else if name, ok := packageFunction(pkg.info, node, "context"); ok && slices.Contains(hostDeadlineFunctions, name) {
							site.Kind, site.Symbol = "clock", "context."+name
						} else if method, ok := timerMethod(pkg.info, node); ok {
							site.Kind, site.Symbol = "clock", method
						} else if path, name, ok := entropyFunction(pkg.info, node); ok {
							site.Kind, site.Symbol = "entropy", path+"."+name
						} else {
							return true
						}
					case *ast.CallExpr:
						path, name, ok := sortFunction(pkg.info, node.Fun)
						if !ok || len(node.Args) == 0 {
							return true
						}
						site.Kind, site.Symbol = "unstable-sort", path+"."+name+" "+types.ExprString(node.Args[0])
					case *ast.RangeStmt:
						typ := pkg.info.TypeOf(node.X)
						if typ == nil {
							return true
						}
						if call, ok := ast.Unparen(node.X).(*ast.CallExpr); ok && len(call.Args) == 1 {
							if name, ok := packageFunction(pkg.info, call.Fun, "maps"); ok && slices.Contains(mapIterators, name) {
								site.Kind, site.Symbol = "map-range", "maps."+name+" "+types.ExprString(call.Args[0])
								break
							}
						}
						if _, ok := typ.Underlying().(*types.Map); !ok {
							return true
						}
						site.Kind, site.Symbol = "map-range", types.ExprString(node.X)
					default:
						return true
					}
					counts[site]++
					return true
				})
			}
		}
	}
	sites := make([]HostSite, 0, len(counts))
	for site, count := range counts {
		site.Count = count
		sites = append(sites, site)
	}
	slices.SortFunc(sites, func(a, b HostSite) int { return strings.Compare(a.Key(), b.Key()) })
	return sites
}

func declarationName(declaration ast.Decl) string {
	function, ok := declaration.(*ast.FuncDecl)
	if !ok {
		return "package"
	}
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return function.Name.Name
	}
	receiver := function.Recv.List[0].Type
	if star, ok := receiver.(*ast.StarExpr); ok {
		receiver = star.X
	}
	if index, ok := receiver.(*ast.IndexExpr); ok {
		receiver = index.X
	}
	if index, ok := receiver.(*ast.IndexListExpr); ok {
		receiver = index.X
	}
	return fmt.Sprintf("%s.%s", types.ExprString(receiver), function.Name.Name)
}

// packageFunction returns the name of a package-level function of the given
// standard package that expression refers to.
func packageFunction(info *types.Info, expression ast.Expr, path string) (string, bool) {
	expression = ast.Unparen(expression)
	if index, ok := expression.(*ast.IndexExpr); ok {
		expression = index.X
	}
	if index, ok := expression.(*ast.IndexListExpr); ok {
		expression = index.X
	}
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	function, ok := info.Uses[selector.Sel].(*types.Func)
	if !ok || function.Pkg() == nil || function.Pkg().Path() != path || function.Type().(*types.Signature).Recv() != nil {
		return "", false
	}
	return function.Name(), true
}

func entropyFunction(info *types.Info, expression ast.Expr) (string, string, bool) {
	for _, path := range entropyPackages {
		if name, ok := packageFunction(info, expression, path); ok {
			return path, name, true
		}
	}
	return "", "", false
}

// timerMethod reports a Reset of a host time.Timer or time.Ticker.
func timerMethod(info *types.Info, selector *ast.SelectorExpr) (string, bool) {
	function, ok := info.Uses[selector.Sel].(*types.Func)
	if !ok || function.Pkg() == nil || function.Pkg().Path() != "time" || !slices.Contains(hostClockMethods, function.Name()) {
		return "", false
	}
	receiver := function.Type().(*types.Signature).Recv()
	if receiver == nil {
		return "", false
	}
	return "time." + symbol(function), true
}

func sortFunction(info *types.Info, expression ast.Expr) (string, string, bool) {
	for path, names := range unstableSorts {
		if name, ok := packageFunction(info, expression, path); ok && slices.Contains(names, name) {
			return path, name, true
		}
	}
	return "", "", false
}

func goTarget(info *types.Info, call *ast.CallExpr) string {
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.FuncLit:
		return "func literal"
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		if function, ok := info.Uses[fun.Sel].(*types.Func); ok {
			return symbol(function)
		}
		return types.ExprString(fun)
	default:
		return types.ExprString(call.Fun)
	}
}
