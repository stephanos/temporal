package boundary

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var discoveryPackages = []string{"crypto/rand", "net", "os", "runtime"}

var discoveryRoots = map[string]struct{}{
	"runtime.nanotime":        {},
	"runtime.time_runtimeNow": {},
}

type discoveredFunction struct {
	key         string
	packagePath string
	symbol      string
	receiver    string
	directSink  bool
	callKeys    []string
}

type discoveryResult struct {
	functions  map[string]*discoveredFunction
	candidates []string
}

var capabilityReceivers = map[string]map[string]struct{}{
	"net": {
		"*IPConn": {}, "*TCPConn": {}, "*TCPListener": {}, "*UDPConn": {},
		"*UnixConn": {}, "*UnixListener": {},
	},
	"os": {"*File": {}, "*Process": {}, "*Root": {}},
}

// DiscoverCandidates returns exported standard-library entry points that can
// reach a host capability sink on the selected platform.
func DiscoverCandidates() ([]string, error) {
	result, err := discoverCandidates()
	if err != nil {
		return nil, err
	}
	return result.candidates, nil
}

func discoverCandidates() (discoveryResult, error) {
	functions := make(map[string]*discoveredFunction)
	selected := make(map[string]struct{}, len(discoveryPackages))
	for _, packagePath := range discoveryPackages {
		selected[packagePath] = struct{}{}
		discovered, err := discoverPackage(packagePath, selected)
		if err != nil {
			return discoveryResult{}, err
		}
		for _, function := range discovered {
			if _, duplicate := functions[function.key]; duplicate {
				return discoveryResult{}, fmt.Errorf("discovered boundary target is duplicated: %s", function.key)
			}
			functions[function.key] = function
		}
	}
	reachesSink := make(map[string]bool, len(functions))
	for key, function := range functions {
		reachesSink[key] = function.directSink
	}
	changed := true
	for changed {
		changed = false
		for key, function := range functions {
			if reachesSink[key] {
				continue
			}
			for _, called := range function.callKeys {
				if reachesSink[called] {
					reachesSink[key] = true
					changed = true
					break
				}
			}
		}
	}
	var candidates []string
	for key, function := range functions {
		exportedReceiver := function.receiver == "" || ast.IsExported(strings.TrimPrefix(function.receiver, "*"))
		_, capabilityMethod := capabilityReceivers[function.packagePath][function.receiver]
		_, explicitRoot := discoveryRoots[key]
		exportedEntry := function.packagePath != "runtime" && ast.IsExported(function.symbol) && exportedReceiver
		if (reachesSink[key] || capabilityMethod) && (explicitRoot || exportedEntry) {
			candidates = append(candidates, key)
		}
	}
	slices.Sort(candidates)
	return discoveryResult{functions: functions, candidates: candidates}, nil
}

// CheckCandidateCoverage fails when source discovery and the reviewed manifest
// disagree. This makes additions and removals in the pinned standard library
// explicit upgrade decisions.
func CheckCandidateCoverage(root string) error {
	definition, err := load(filepath.Join(root, filepath.FromSlash(manifestPath)))
	if err != nil {
		return err
	}
	discovery, err := discoverCandidates()
	if err != nil {
		return err
	}
	if err := validateCandidateCoverage(definition, discovery.candidates); err != nil {
		return err
	}
	return validateDelegateReachability(definition, discovery.functions)
}

func validateCandidateCoverage(definition manifest, candidates []string) error {
	classified := make(map[string]struct{}, len(definition.Intercepts)+len(definition.ReviewedCandidates))
	for _, entry := range definition.Intercepts {
		classified[entry.Package+"."+targetName(entry.Receiver, entry.Symbol)] = struct{}{}
	}
	for _, candidate := range definition.ReviewedCandidates {
		classified[candidate.Target] = struct{}{}
	}
	discovered := make(map[string]struct{}, len(candidates))
	var missing []string
	for _, candidate := range candidates {
		discovered[candidate] = struct{}{}
		if _, found := classified[candidate]; !found {
			missing = append(missing, candidate)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("unclassified host-capability candidates:\n%s", strings.Join(missing, "\n"))
	}
	var stale []string
	for _, candidate := range definition.ReviewedCandidates {
		if _, found := discovered[candidate.Target]; !found {
			stale = append(stale, candidate.Target)
		}
	}
	if len(stale) != 0 {
		slices.Sort(stale)
		return fmt.Errorf("reviewed host-capability candidates are no longer discovered:\n%s", strings.Join(stale, "\n"))
	}
	return nil
}

func validateDelegateReachability(definition manifest, functions map[string]*discoveredFunction) error {
	probeTargets := make(map[string]string, len(definition.Intercepts))
	for _, entry := range definition.Intercepts {
		probeTargets[entry.Probe] = entry.Package + "." + targetName(entry.Receiver, entry.Symbol)
	}
	for _, candidate := range definition.ReviewedCandidates {
		if candidate.Disposition != "delegate" {
			continue
		}
		reachable := false
		for _, probe := range candidate.Boundaries {
			if functionReaches(functions, candidate.Target, probeTargets[probe], make(map[string]struct{})) {
				reachable = true
				break
			}
		}
		if !reachable {
			return fmt.Errorf("reviewed delegate %s does not reach a controlling boundary", candidate.Target)
		}
	}
	return nil
}

func functionReaches(functions map[string]*discoveredFunction, current, target string, seen map[string]struct{}) bool {
	if current == target {
		return true
	}
	if _, visited := seen[current]; visited {
		return false
	}
	seen[current] = struct{}{}
	function := functions[current]
	if function == nil {
		return false
	}
	for _, called := range function.callKeys {
		if functionReaches(functions, called, target, seen) {
			return true
		}
	}
	return false
}

func discoverPackage(packagePath string, selected map[string]struct{}) ([]*discoveredFunction, error) {
	context := build.Default
	context.CgoEnabled = false
	description, err := context.Import(packagePath, "", 0)
	if err != nil {
		return nil, fmt.Errorf("locate boundary discovery package %s: %w", packagePath, err)
	}
	files := append([]string(nil), description.GoFiles...)
	files = append(files, description.CgoFiles...)
	slices.Sort(files)
	fileSet := token.NewFileSet()
	var parsedFiles []*ast.File
	gomadFiles := make(map[*ast.File]struct{})
	for _, name := range files {
		path := filepath.Join(description.Dir, name)
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read boundary discovery source %s: %w", path, err)
		}
		parsed, err := parser.ParseFile(fileSet, path, contents, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse boundary discovery source %s: %w", path, err)
		}
		parsedFiles = append(parsedFiles, parsed)
		if name == "gomad.go" {
			gomadFiles[parsed] = struct{}{}
		}
	}
	typeInfo := &types.Info{
		Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	configuration := types.Config{Importer: importer.ForCompiler(fileSet, "source", nil)}
	if _, err := configuration.Check(packagePath, fileSet, parsedFiles, typeInfo); err != nil {
		return nil, fmt.Errorf("type-check boundary discovery package %s: %w", packagePath, err)
	}
	var functions []*discoveredFunction
	for _, parsed := range parsedFiles {
		if _, gomadFile := gomadFiles[parsed]; gomadFile {
			continue
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if function.Name.Name == "init" {
				continue
			}
			object, ok := typeInfo.Defs[function.Name].(*types.Func)
			if !ok {
				return nil, fmt.Errorf("resolve boundary discovery function %s.%s", packagePath, function.Name.Name)
			}
			key, receiverName := discoveredFunctionKey(object)
			discovered := &discoveredFunction{
				key: key, packagePath: packagePath,
				symbol: function.Name.Name, receiver: receiverName,
				directSink: function.Body == nil || hasLinknameDirective(function.Doc),
			}
			if function.Body != nil {
				ast.Inspect(function.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					called := calledFunction(call.Fun, typeInfo)
					if unresolvedCapabilityCall(call.Fun, called, typeInfo) {
						discovered.directSink = true
					}
					if called == nil || called.Pkg() == nil {
						return true
					}
					calledPackage := called.Pkg().Path()
					if isHostCapabilityFunction(called) {
						discovered.directSink = true
					}
					if calledPackage == packagePath {
						discovered.callKeys = append(discovered.callKeys, functionKeyOnly(called))
					} else if _, tracked := selected[calledPackage]; tracked {
						discovered.callKeys = append(discovered.callKeys, functionKeyOnly(called))
					}
					return true
				})
			}
			functions = append(functions, discovered)
		}
	}
	return functions, nil
}

func hasLinknameDirective(comments *ast.CommentGroup) bool {
	if comments == nil {
		return false
	}
	for _, comment := range comments.List {
		if strings.HasPrefix(comment.Text, "//go:linkname ") {
			return true
		}
	}
	return false
}

func unresolvedCapabilityCall(expression ast.Expr, function *types.Func, info *types.Info) bool {
	if selector, ok := expression.(*ast.SelectorExpr); ok {
		if selection := info.Selections[selector]; selection != nil {
			if _, dynamic := selection.Recv().Underlying().(*types.Interface); dynamic {
				return true
			}
		}
	}
	if function != nil {
		return false
	}
	switch value := expression.(type) {
	case *ast.Ident:
		_, functionValue := info.Uses[value].(*types.Var)
		return functionValue
	case *ast.SelectorExpr:
		_, functionValue := info.Uses[value.Sel].(*types.Var)
		return functionValue
	default:
		return false
	}
}

func calledFunction(expression ast.Expr, info *types.Info) *types.Func {
	switch value := expression.(type) {
	case *ast.Ident:
		function, _ := info.Uses[value].(*types.Func)
		return function
	case *ast.SelectorExpr:
		if selection := info.Selections[value]; selection != nil {
			function, _ := selection.Obj().(*types.Func)
			return function
		}
		function, _ := info.Uses[value.Sel].(*types.Func)
		return function
	default:
		return nil
	}
}

func functionKeyOnly(function *types.Func) string {
	key, _ := discoveredFunctionKey(function)
	return key
}

func discoveredFunctionKey(function *types.Func) (string, string) {
	receiverName := ""
	signature, _ := function.Type().(*types.Signature)
	if signature != nil && signature.Recv() != nil {
		receiverType := signature.Recv().Type()
		prefix := ""
		if pointer, ok := receiverType.(*types.Pointer); ok {
			prefix = "*"
			receiverType = pointer.Elem()
		}
		if named, ok := receiverType.(*types.Named); ok {
			receiverName = prefix + named.Obj().Name()
		}
	}
	return function.Pkg().Path() + "." + discoveredTarget(receiverName, function.Name()), receiverName
}

func isHostCapabilityFunction(function *types.Func) bool {
	packagePath := function.Pkg().Path()
	if packagePath == "syscall" {
		signature, _ := function.Type().(*types.Signature)
		if signature != nil && signature.Recv() != nil {
			return false
		}
		switch function.Name() {
		case "BytePtrFromString", "ByteSliceFromString", "Clearenv", "Environ", "Exit", "Getenv", "NsecToTimespec", "NsecToTimeval", "Setenv", "SlicePtrFromStrings", "StringBytePtr", "StringByteSlice", "StringSlicePtr", "TimespecToNsec", "TimevalToNsec", "Unsetenv":
			return false
		}
		return true
	}
	return packagePath == "internal/poll" || strings.HasPrefix(packagePath, "internal/syscall/") || packagePath == "crypto/internal/sysrand" || packagePath == "os/exec" || packagePath == "plugin" || packagePath == "runtime/cgo"
}

func discoveredTarget(receiverName, symbol string) string {
	if receiverName == "" {
		return symbol
	}
	return "(" + receiverName + ")." + symbol
}
