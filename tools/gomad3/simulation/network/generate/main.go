package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	check := flag.Bool("check", false, "verify the native network projection without writing")
	flag.Parse()
	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(check bool) error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	generated, err := generate(root)
	if err != nil {
		return err
	}
	destination := filepath.Join(root, "simulation/network/generated.go")
	if check {
		actual, err := os.ReadFile(destination)
		if err != nil {
			return err
		}
		if !bytes.Equal(actual, generated) {
			return fmt.Errorf("generated network source is stale: %s", destination)
		}
		return nil
	}
	return os.WriteFile(destination, generated, 0o644)
}

func moduleRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		module, err := os.ReadFile(filepath.Join(root, "go.mod"))
		if err == nil && bytes.HasPrefix(module, []byte("module go.temporal.io/server/tools/gomad3\n")) {
			return root, nil
		}
		parent := filepath.Dir(root)
		if parent == root {
			return "", fmt.Errorf("gomad3 module root not found")
		}
		root = parent
	}
}

var nativeBridges = map[string]bool{
	"BeginSimulation": true, "PartitionSimulation": true, "HealSimulation": true,
	"DelaySimulation": true, "ChangeSimulationGroup": true, "RevokeSimulation": true,
	"FinishSimulation": true, "simulationNetworkForRun": true, "currentSimulationNetwork": true,
	"changeSimulationPair": true, "changeSimulationLink": true,
}

var supportTypes = map[string]bool{
	"Address": true, "listenerImplementation": true, "connImplementation": true,
	"Listener": true, "Conn": true, "pairedConn": true, "connState": true,
	"connShared": true, "networkChunk": true,
}

func generate(root string) ([]byte, error) {
	fset := token.NewFileSet()
	var selected strings.Builder
	selected.WriteString("package network\n\n")
	imports := map[string]string{"time": "time"}
	bridges := 0
	for _, name := range []string{"network.go", "simulation_network.go", "simulation_handles.go", "simulation_wire.go"} {
		path := filepath.Join(root, "backends/native/toolchain/runtime/overlay/src/internal/gomadio", name)
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return nil, err
			}
			if strings.HasPrefix(path, "internal/") || path == "unsafe" {
				continue
			}
			imports[filepath.Base(path)] = path
		}
		for _, declaration := range file.Decls {
			include := false
			start := declaration.Pos()
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				if declaration.Tok == token.IMPORT {
					continue
				}
				include = name != "network.go"
				if name == "simulation_network.go" && declaration.Tok == token.VAR {
					include = declaration.Specs[0].(*ast.ValueSpec).Names[0].Name != "simulationNetworkState"
				}
				if name == "network.go" {
					include = declaration.Tok == token.CONST || declaration.Tok == token.VAR && declaration.Specs[0].(*ast.ValueSpec).Names[0].Name == "ErrAddressInUse"
					if declaration.Tok == token.TYPE {
						include = supportTypes[declaration.Specs[0].(*ast.TypeSpec).Name.Name]
					}
				}
				if declaration.Doc != nil {
					start = declaration.Doc.Pos()
				}
			case *ast.FuncDecl:
				include = name != "network.go"
				if name == "simulation_network.go" && declaration.Recv == nil && nativeBridges[declaration.Name.Name] {
					include = false
					bridges++
				}
				if name == "network.go" {
					receiver := receiverType(declaration)
					include = receiver == "Listener" || receiver == "Conn" || receiver == "connShared" && declaration.Name.Name == "signal" || declaration.Recv == nil && (declaration.Name.Name == "newConnStates" || declaration.Name.Name == "earliestDeadline")
				}
				if declaration.Doc != nil {
					start = declaration.Doc.Pos()
				}
			}
			if include {
				selected.Write(source[fset.Position(start).Offset:fset.Position(declaration.End()).Offset])
				selected.WriteString("\n\n")
			}
		}
	}
	if bridges != len(nativeBridges) {
		return nil, fmt.Errorf("native bridge inventory changed: got %d, expected %d", bridges, len(nativeBridges))
	}
	projectedSet := token.NewFileSet()
	file, err := parser.ParseFile(projectedSet, "projection.go", selected.String(), parser.ParseComments)
	if err != nil {
		return nil, err
	}
	counts, err := adapt(file)
	if err != nil {
		return nil, err
	}
	reset, err := resetConnection(file, projectedSet)
	if err != nil {
		return nil, err
	}
	expected := map[string]int{"record": 14, "record_input": 1, "domain": 3, "descriptor": 1, "closed": 1, "validate": 1, "fields": 1, "now": 2, "deadline": 3, "wait": 4, "dial_wait": 2, "wait_continue": 3}
	for name, want := range expected {
		if counts[name] != want {
			return nil, fmt.Errorf("native network rewrite %s changed: got %d, expected %d", name, counts[name], want)
		}
	}
	used := make(map[string]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			if identifier, ok := selector.X.(*ast.Ident); ok {
				if identifier.Name == "gomadsim" || strings.HasPrefix(identifier.Name, "gomadvfd") {
					err = fmt.Errorf("unadapted native selector %s.%s", identifier.Name, selector.Sel.Name)
				}
				used[identifier.Name] = true
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(imports))
	for name := range imports {
		if used[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var output bytes.Buffer
	output.WriteString("// Copyright 2026 The Go Authors. All rights reserved.\n// Use of this source code is governed by a BSD-style\n// license that can be found in the LICENSE file.\n\n// Code generated from the native Gomad network model; DO NOT EDIT.\n\npackage network\n\nimport (\n")
	for _, name := range names {
		fmt.Fprintf(&output, "\t%q\n", imports[name])
	}
	output.WriteString(")\n\n")
	var body bytes.Buffer
	if err := format.Node(&body, token.NewFileSet(), file); err != nil {
		return nil, err
	}
	output.Write(bytes.TrimPrefix(body.Bytes(), []byte("package network\n")))
	output.WriteString("\n\n")
	output.Write(reset)
	return format.Source(output.Bytes())
}

func receiverType(function *ast.FuncDecl) string {
	if function.Recv == nil {
		return ""
	}
	expression := function.Recv.List[0].Type
	if pointer, ok := expression.(*ast.StarExpr); ok {
		expression = pointer.X
	}
	if identifier, ok := expression.(*ast.Ident); ok {
		return identifier.Name
	}
	return ""
}

func expression(source string) (ast.Expr, error) {
	return parser.ParseExpr(source)
}

func adapt(file *ast.File) (map[string]int, error) {
	counts := make(map[string]int)
	var adaptationErr error
	parseExpression := func(source string) ast.Expr {
		parsed, err := expression(source)
		if err != nil && adaptationErr == nil {
			adaptationErr = fmt.Errorf("parse projection expression: %w", err)
		}
		return parsed
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.Field:
			if selector, ok := node.Type.(*ast.SelectorExpr); ok {
				if identifier, ok := selector.X.(*ast.Ident); ok && identifier.Name == "gomadsim" && selector.Sel.Name == "NetworkDomain" {
					node.Type = ast.NewIdent("Domain")
					counts["domain"]++
				}
			}
		case *ast.TypeSpec:
			if node.Name.Name == "connState" {
				structure := node.Type.(*ast.StructType)
				var fields []*ast.Field
				for _, field := range structure.Fields.List {
					if identifier, ok := field.Type.(*ast.Ident); ok && identifier.Name == "descriptorIdentity" {
						counts["descriptor"]++
						continue
					}
					fields = append(fields, field)
				}
				structure.Fields.List = fields
			}
			if node.Name.Name == "simulationNetwork" {
				structure := node.Type.(*ast.StructType)
				for _, field := range []struct{ name, typ string }{{"current", "Domain"}, {"now", "time.Time"}, {"live", "map[string]Domain"}} {
					structure.Fields.List = append(structure.Fields.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(field.name)}, Type: parseExpression(field.typ)})
				}
				counts["fields"]++
			}
		case *ast.ValueSpec:
			if len(node.Names) == 1 && node.Names[0].Name == "ErrClosed" {
				node.Values = []ast.Expr{parseExpression(`errors.New("use of closed network connection")`)}
				counts["closed"]++
			}
		case *ast.FuncDecl:
			if node.Name.Name == "validateSimulationEndpoint" {
				node.Body.List = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{parseExpression("network.validateEndpoint(endpoint)")}}}
				counts["validate"]++
			}
		case *ast.CallExpr:
			if selector, ok := node.Fun.(*ast.SelectorExpr); ok {
				if identifier, ok := selector.X.(*ast.Ident); ok && identifier.Name == "time" && selector.Sel.Name == "Now" {
					node.Fun = parseExpression("connection.network.nowTime")
					counts["now"]++
				}
			}
		case *ast.BlockStmt:
			var statements []ast.Stmt
			for index, statement := range node.List {
				if branch, ok := statement.(*ast.BranchStmt); ok && branch.Tok == token.CONTINUE && len(statements) != 0 && isWaiting(statements[len(statements)-1]) {
					counts["wait_continue"]++
					continue
				}
				if assignment, ok := statement.(*ast.AssignStmt); ok && len(assignment.Lhs) == 1 && len(assignment.Rhs) == 1 {
					left, leftOK := assignment.Lhs[0].(*ast.Ident)
					right, rightOK := assignment.Rhs[0].(*ast.Ident)
					if leftOK && rightOK && left.Name == "input" && right.Name == "source" && assignment.Tok == token.DEFINE {
						counts["record_input"]++
						continue
					}
				}
				if callStatement, ok := statement.(*ast.ExprStmt); ok {
					if call, ok := callStatement.X.(*ast.CallExpr); ok {
						if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "record" {
							counts["record"]++
							continue
						}
						if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "waitForChange" {
							if len(call.Args) != 2 || index == 0 || !unlocks(node.List[index-1]) {
								adaptationErr = fmt.Errorf("native wait moved before unlock")
								return false
							}
							callStatement.X = waiting(call.Args[0], call.Args[1])
							counts["wait"]++
						}
					}
				}
				if selection, ok := statement.(*ast.SelectStmt); ok {
					if !dialWait(selection) || index == 0 || !unlocks(node.List[index-1]) {
						adaptationErr = fmt.Errorf("native dial select shape or unlock changed")
						return false
					}
					statement = &ast.ExprStmt{X: waiting(ast.NewIdent("changed"), parseExpression("deadlineOf(ctx)"))}
					counts["dial_wait"]++
				}
				statements = append(statements, statement)
			}
			node.List = statements
		}
		return true
	})
	// The receiver selects the host clock used by deadline checks.
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		receiver := "network"
		if receiverType(function) == "simulationConn" {
			receiver = "connection.network"
		}
		ast.Inspect(function, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "deadlineExpired" {
					call.Fun = parseExpression(receiver + ".deadlineExpired")
					counts["deadline"]++
				}
			}
			return true
		})
	}
	return counts, adaptationErr
}

func unlocks(statement ast.Stmt) bool {
	callStatement, ok := statement.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := callStatement.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	receiver, ok := selector.X.(*ast.Ident)
	return ok && (receiver.Name == "network" && selector.Sel.Name == "Unlock" || receiver.Name == "connection" && selector.Sel.Name == "unlockState")
}

func dialWait(selection *ast.SelectStmt) bool {
	if len(selection.Body.List) != 2 {
		return false
	}
	for index, statement := range selection.Body.List {
		clause, ok := statement.(*ast.CommClause)
		if !ok || len(clause.Body) != 0 {
			return false
		}
		receive, ok := clause.Comm.(*ast.ExprStmt)
		if !ok {
			return false
		}
		unary, ok := receive.X.(*ast.UnaryExpr)
		if !ok || unary.Op != token.ARROW {
			return false
		}
		if index == 0 {
			identifier, ok := unary.X.(*ast.Ident)
			if !ok || identifier.Name != "changed" {
				return false
			}
		} else {
			call, ok := unary.X.(*ast.CallExpr)
			if !ok || len(call.Args) != 0 {
				return false
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Done" {
				return false
			}
			identifier, ok := selector.X.(*ast.Ident)
			if !ok || identifier.Name != "ctx" {
				return false
			}
		}
	}
	return true
}

func waiting(changed, deadline ast.Expr) ast.Expr {
	fields := []ast.Expr{&ast.KeyValueExpr{Key: ast.NewIdent("Changed"), Value: changed}}
	if deadline != nil {
		fields = append(fields, &ast.KeyValueExpr{Key: ast.NewIdent("Deadline"), Value: deadline})
	}
	return &ast.CallExpr{Fun: ast.NewIdent("panic"), Args: []ast.Expr{&ast.CompositeLit{Type: ast.NewIdent("waiting"), Elts: fields}}}
}

func isWaiting(statement ast.Stmt) bool {
	callStatement, ok := statement.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := callStatement.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	identifier, ok := call.Fun.(*ast.Ident)
	if !ok || identifier.Name != "panic" {
		return false
	}
	literal, ok := call.Args[0].(*ast.CompositeLit)
	if !ok {
		return false
	}
	identifier, ok = literal.Type.(*ast.Ident)
	return ok && identifier.Name == "waiting"
}

// Named connection resets derive their stream mutation from the native crash path.
func resetConnection(file *ast.File, fset *token.FileSet) ([]byte, error) {
	var loop *ast.RangeStmt
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "revoke" {
			continue
		}
		for _, statement := range function.Body.List {
			candidate, ok := statement.(*ast.RangeStmt)
			if !ok {
				continue
			}
			selector, ok := candidate.X.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "connections" || len(candidate.Body.List) == 0 {
				continue
			}
			filter, ok := candidate.Body.List[0].(*ast.IfStmt)
			if !ok || len(filter.Body.List) != 1 {
				continue
			}
			branch, ok := filter.Body.List[0].(*ast.BranchStmt)
			if ok && branch.Tok == token.CONTINUE {
				if loop != nil {
					return nil, fmt.Errorf("native revoke has multiple reset loops")
				}
				loop = candidate
			}
		}
	}
	if loop == nil {
		return nil, fmt.Errorf("native revoke reset loop is missing")
	}
	var source bytes.Buffer
	source.WriteString("package network\nfunc (network *simulationNetwork) resetConnection(identity uint64) error {\nnetwork.Lock()\ndefer network.Unlock()\nif network.finished { return syscall.ESTALE }\nfound := false\n")
	if err := format.Node(&source, fset, loop); err != nil {
		return nil, err
	}
	source.WriteString("\nif !found { return syscall.ESTALE }\nnetwork.signalLocked()\nreturn nil\n}")
	derivedSet := token.NewFileSet()
	derived, err := parser.ParseFile(derivedSet, "reset.go", source.Bytes(), 0)
	if err != nil {
		return nil, err
	}
	function := derived.Decls[0].(*ast.FuncDecl)
	loop = function.Body.List[4].(*ast.RangeStmt)
	condition, err := expression("connection.identity != identity")
	if err != nil {
		return nil, fmt.Errorf("parse reset projection filter: %w", err)
	}
	loop.Body.List[0].(*ast.IfStmt).Cond = condition
	var statements []ast.Stmt
	resetBranches := 0
	for index, statement := range loop.Body.List {
		if index == 1 {
			statements = append(statements, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("found")}, Tok: token.ASSIGN, Rhs: []ast.Expr{ast.NewIdent("true")}})
		}
		if branch, ok := statement.(*ast.IfStmt); ok {
			if identifier, ok := branch.Cond.(*ast.Ident); ok && identifier.Name == "graceful" {
				otherwise, ok := branch.Else.(*ast.BlockStmt)
				if !ok {
					return nil, fmt.Errorf("native revoke reset branch changed")
				}
				statements = append(statements, otherwise.List...)
				resetBranches++
				continue
			}
		}
		statements = append(statements, statement)
	}
	if resetBranches != 1 {
		return nil, fmt.Errorf("native revoke reset branch inventory changed: %d", resetBranches)
	}
	loop.Body.List = statements
	var output bytes.Buffer
	if err := format.Node(&output, token.NewFileSet(), function); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
