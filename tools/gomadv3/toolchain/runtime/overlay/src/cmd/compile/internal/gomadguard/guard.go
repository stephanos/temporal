// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gomadguard guards Go entry points in forbidden capability packages.
package gomadguard

import (
	"fmt"
	"strings"

	"cmd/compile/internal/base"
	"cmd/compile/internal/ir"
	"cmd/compile/internal/typecheck"
	"cmd/compile/internal/types"
	"cmd/internal/gomadcap"
	"cmd/internal/src"
)

// Apply installs a guard before each instrumentable non-init function body.
func Apply(pkg *ir.Package) {
	if !*base.GomadGuard || !gomadcap.IsForbiddenImport(types.LocalPkg.Path) {
		return
	}
	guard := guardFunction()
	for _, function := range pkg.Funcs {
		if !guardable(function) {
			continue
		}
		ir.CurFunc = function
		function.Pragma |= ir.Noinline
		function.Body.Prepend(typecheck.Stmt(typecheck.Call(function.Pos(), guard, nil, false)))
		if base.Flag.LowerM != 0 {
			fmt.Printf("gomad guard applied: %s.%s\n", types.LocalPkg.Path, function.Sym().Name)
		}
	}
	ir.CurFunc = nil
}

func guardable(function *ir.Func) bool {
	if function == nil || function.Sym() == nil || len(function.Body) == 0 || function.IsPackageInit() {
		return false
	}
	name := function.Sym().Name
	return name != "init" && !strings.HasPrefix(name, "init.")
}

func guardFunction() *ir.Name {
	name := strings.TrimPrefix(gomadcap.GuardSymbol, "runtime.")
	symbol := ir.Pkgs.Runtime.Lookup(name)
	if symbol.Def != nil {
		guard, ok := symbol.Def.(*ir.Name)
		if !ok {
			base.Fatalf("gomad guard runtime symbol has unexpected definition: %T", symbol.Def)
		}
		return guard
	}
	function := ir.NewFunc(src.NoXPos, src.NoXPos, symbol, types.NewSignature(nil, nil, nil))
	symbol.Def = function.Nname
	return function.Nname
}
