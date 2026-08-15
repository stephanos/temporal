// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gomadintercept rewrites selected function bodies to call Gomad hooks.
package gomadintercept

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"cmd/compile/internal/base"
	"cmd/compile/internal/ir"
	"cmd/compile/internal/typecheck"
	"cmd/compile/internal/types"
	"internal/buildcfg"
)

type receiverSpec struct {
	Name    string
	Pointer bool
}

type spec struct {
	PackagePath       string
	Receiver          *receiverSpec
	Function          string
	Hook              string
	DeclarationSHA256 string
	ProbeID           uint64
}

type interception struct {
	spec     spec
	target   *ir.Func
	hook     *ir.Func
	observer *ir.Func
}

// Apply installs all interceptions selected for the package.
func Apply(pkg *ir.Package) {
	if !qualifiedPlatform(buildcfg.GOOS, buildcfg.GOARCH) {
		return
	}
	var selected []spec
	for _, candidate := range specs {
		if candidate.PackagePath == types.LocalPkg.Path {
			selected = append(selected, candidate)
		}
	}
	if len(selected) == 0 {
		return
	}

	interceptions := make([]interception, 0, len(selected))
	seenTargets := make(map[*ir.Func]spec)
	var observer *ir.Func
	for _, candidate := range selected {
		targets := findFuncs(pkg, candidate.Function, candidate.Receiver)
		if len(targets) == 0 {
			base.Fatalf("gomad interception target is missing: %s", candidate.targetName())
		}
		if len(targets) != 1 {
			base.Fatalf("gomad interception target is ambiguous: %s has %d definitions", candidate.targetName(), len(targets))
		}
		target := targets[0]
		if previous, ok := seenTargets[target]; ok {
			base.Fatalf("gomad interception target is duplicated in manifest: %s and %s", previous.targetName(), candidate.targetName())
		}
		seenTargets[target] = candidate
		if len(target.Body) == 0 {
			base.Fatalf("gomad interception target has no body: %s", candidate.targetName())
		}
		if err := validateDeclarationFingerprint(target, candidate); err != nil {
			base.Fatalf("gomad interception declaration fingerprint mismatch for %s: %v", candidate.targetName(), err)
		}

		hooks := findFuncs(pkg, candidate.Hook, nil)
		if len(hooks) == 0 {
			base.Fatalf("gomad interception hook is missing: %s", candidate.Hook)
		}
		if len(hooks) != 1 {
			base.Fatalf("gomad interception hook is ambiguous: %s has %d definitions", candidate.Hook, len(hooks))
		}
		hook := hooks[0]
		if len(hook.Body) == 0 {
			base.Fatalf("gomad interception hook has no body: %s", candidate.Hook)
		}
		if err := validateSignature(target, hook); err != nil {
			base.Fatalf("gomad interception signature mismatch for %s: %v", candidate.targetName(), err)
		}
		if candidate.ProbeID != 0 && observer == nil {
			observers := findFuncs(pkg, "gomadObserveBoundary", nil)
			if len(observers) == 0 {
				base.Fatalf("gomad semantic probe observer is missing")
			}
			if len(observers) != 1 {
				base.Fatalf("gomad semantic probe observer is ambiguous: %d definitions", len(observers))
			}
			observer = observers[0]
			if len(observer.Body) == 0 {
				base.Fatalf("gomad semantic probe observer has no body")
			}
			if err := validateObserver(observer); err != nil {
				base.Fatalf("gomad semantic probe observer signature mismatch: %v", err)
			}
		}
		var candidateObserver *ir.Func
		if candidate.ProbeID != 0 {
			candidateObserver = observer
		}
		interceptions = append(interceptions, interception{spec: candidate, target: target, hook: hook, observer: candidateObserver})
	}

	for _, interception := range interceptions {
		prependHook(interception.target, interception.hook, interception.observer, interception.spec.ProbeID)
		if base.Flag.LowerM != 0 {
			fmt.Printf("gomad intercept applied: %s.%s -> %s.%s\n", types.LocalPkg.Path, interception.spec.targetName(), types.LocalPkg.Path, interception.spec.Hook)
		}
	}
	ir.CurFunc = nil
}

func validateDeclarationFingerprint(target *ir.Func, candidate spec) error {
	if candidate.DeclarationSHA256 == "" {
		return nil
	}
	position := base.Ctxt.OutermostPos(target.Pos())
	filename := position.AbsFilename()
	const dollarGOROOT = "$GOROOT"
	if buildcfg.GOROOT != "" && strings.HasPrefix(filename, dollarGOROOT+"/") {
		filename = filepath.FromSlash(buildcfg.GOROOT + filename[len(dollarGOROOT):])
	}
	contents, err := os.ReadFile(filename)
	if err != nil && buildcfg.GOROOT != "" && !filepath.IsAbs(filename) {
		filename = filepath.Join(buildcfg.GOROOT, "src", filepath.FromSlash(filename))
		contents, err = os.ReadFile(filename)
	}
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, filename, contents, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse source: %w", err)
	}
	var match *ast.FuncDecl
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != candidate.Function || !matchesASTReceiver(function.Recv, candidate.Receiver) {
			continue
		}
		if match != nil {
			return fmt.Errorf("source declaration is ambiguous")
		}
		match = function
	}
	if match == nil {
		return fmt.Errorf("source declaration is missing")
	}
	var declaration bytes.Buffer
	if err := format.Node(&declaration, files, match); err != nil {
		return fmt.Errorf("format source declaration: %w", err)
	}
	digest := sha256.Sum256(declaration.Bytes())
	actual := fmt.Sprintf("sha256:%x", digest)
	if actual != candidate.DeclarationSHA256 {
		return fmt.Errorf("got %s, want %s", actual, candidate.DeclarationSHA256)
	}
	return nil
}

func matchesASTReceiver(fields *ast.FieldList, want *receiverSpec) bool {
	if want == nil {
		return fields == nil
	}
	if fields == nil || len(fields.List) != 1 {
		return false
	}
	receiver := fields.List[0].Type
	if pointer, ok := receiver.(*ast.StarExpr); ok {
		if !want.Pointer {
			return false
		}
		receiver = pointer.X
	} else if want.Pointer {
		return false
	}
	name, ok := receiver.(*ast.Ident)
	return ok && name.Name == want.Name
}

func findFuncs(pkg *ir.Package, name string, receiver *receiverSpec) []*ir.Func {
	var matches []*ir.Func
	for _, fn := range pkg.Funcs {
		if matchesFuncName(fn.Sym().Name, name, receiver) && matchesReceiver(fn.Type().Recv(), receiver) {
			matches = append(matches, fn)
		}
	}
	return matches
}

func matchesFuncName(actual, name string, receiver *receiverSpec) bool {
	if receiver == nil {
		return actual == name
	}
	prefix := ""
	if receiver.Pointer {
		prefix = "*"
	}
	return actual == fmt.Sprintf("(%s%s).%s", prefix, receiver.Name, name)
}

func matchesReceiver(field *types.Field, want *receiverSpec) bool {
	if want == nil {
		return field == nil
	}
	if field == nil {
		return false
	}
	receiver := field.Type
	if receiver.IsPtr() != want.Pointer {
		return false
	}
	if want.Pointer {
		receiver = receiver.Elem()
	}
	return receiver.Sym() != nil && receiver.Sym().Name == want.Name
}

func validateSignature(target, hook *ir.Func) error {
	targetType := target.Type()
	hookType := hook.Type()
	targetInputs := targetType.RecvParams()

	if hookType.Recv() != nil {
		return fmt.Errorf("hook must be a package function")
	}
	if len(hookType.Params()) != len(targetInputs) {
		return fmt.Errorf("hook has %d parameters, want %d", len(hookType.Params()), len(targetInputs))
	}
	for i, input := range targetInputs {
		if !types.IdenticalStrict(input.Type, hookType.Param(i).Type) {
			return fmt.Errorf("hook parameter %d has type %v, want %v", i+1, hookType.Param(i).Type, input.Type)
		}
	}
	if targetType.IsVariadic() != hookType.IsVariadic() {
		return fmt.Errorf("hook variadic form does not match target")
	}
	if hookType.NumResults() != targetType.NumResults()+1 {
		return fmt.Errorf("hook has %d results, want %d", hookType.NumResults(), targetType.NumResults()+1)
	}
	for i, result := range targetType.Results() {
		if !types.IdenticalStrict(result.Type, hookType.Result(i).Type) {
			return fmt.Errorf("hook result %d has type %v, want %v", i+1, hookType.Result(i).Type, result.Type)
		}
	}
	if !types.IdenticalStrict(hookType.Result(hookType.NumResults()-1).Type, types.Types[types.TBOOL]) {
		return fmt.Errorf("hook final result must be bool")
	}
	return nil
}

func validateObserver(observer *ir.Func) error {
	type_ := observer.Type()
	if type_.Recv() != nil || type_.IsVariadic() || len(type_.Params()) != 1 || type_.NumResults() != 0 {
		return fmt.Errorf("observer must have signature func(uint64)")
	}
	if !types.IdenticalStrict(type_.Param(0).Type, types.Types[types.TUINT64]) {
		return fmt.Errorf("observer parameter has type %v, want uint64", type_.Param(0).Type)
	}
	return nil
}

func prependHook(target, hook, observer *ir.Func, probeID uint64) {
	pos := target.Pos()
	ir.CurFunc = target
	target.Pragma |= ir.Noinline
	var prologue []ir.Node
	if observer != nil {
		id := typecheck.DefaultLit(ir.NewInt(pos, int64(probeID)), types.Types[types.TUINT64])
		prologue = append(prologue, typecheck.Stmt(typecheck.Call(pos, observer.Nname, []ir.Node{id}, false)))
	}

	args := make([]ir.Node, 0, target.Type().NumRecvs()+target.Type().NumParams())
	for _, input := range target.Type().RecvParams() {
		args = append(args, input.Nname.(*ir.Name))
	}

	leadingResults := make([]ir.Node, 0, target.Type().NumResults())
	lhs := make([]ir.Node, 0, target.Type().NumResults()+1)
	for _, result := range target.Type().Results() {
		temporary := typecheck.TempAt(pos, target, result.Type)
		leadingResults = append(leadingResults, temporary)
		lhs = append(lhs, temporary)
	}
	handled := typecheck.TempAt(pos, target, types.Types[types.TBOOL])
	lhs = append(lhs, handled)

	call := typecheck.Call(pos, hook.Nname, args, target.Type().IsVariadic())
	var assignment ir.Node
	if len(lhs) == 1 {
		assignment = typecheck.Stmt(ir.NewAssignStmt(pos, lhs[0], call))
	} else {
		assignment = typecheck.Stmt(ir.NewAssignListStmt(pos, ir.OAS2, lhs, []ir.Node{call}))
	}
	ret := typecheck.Stmt(ir.NewReturnStmt(pos, leadingResults))
	conditional := typecheck.Stmt(ir.NewIfStmt(pos, handled, []ir.Node{ret}, nil))
	prologue = append(prologue, assignment, conditional)
	target.Body.Prepend(prologue...)
}

func (s spec) targetName() string {
	if s.Receiver == nil {
		return s.Function
	}
	prefix := ""
	if s.Receiver.Pointer {
		prefix = "*"
	}
	return fmt.Sprintf("(%s%s).%s", prefix, s.Receiver.Name, s.Function)
}
