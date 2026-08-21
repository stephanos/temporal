// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ld

import (
	"flag"

	"cmd/internal/gomadcap"
	"cmd/link/internal/loader"
	"cmd/link/internal/sym"
	"internal/buildcfg"
)

var flagGomadCapability = flag.String("gomadcap", "", "emit Gomad live capability metadata for `toolchain-build-key`")

func gomadCapabilityManifest(ctxt *Link) {
	if *flagGomadCapability == "" {
		return
	}
	if buildcfg.GOOS != "darwin" || buildcfg.GOARCH != "arm64" {
		Exitf("-gomadcap is supported only on darwin/arm64")
	}
	if ctxt.LinkMode != LinkInternal || ctxt.BuildMode != BuildModeExe && ctxt.BuildMode != BuildModePIE {
		Exitf("-gomadcap requires an internally linked executable")
	}

	ldr := ctxt.loader
	facts := []gomadcap.Fact{}
	symbols := ldr.NSym()
	for index := 1; index < symbols; index++ {
		owner := loader.Sym(index)
		if !ldr.AttrReachable(owner) {
			continue
		}
		ownerPackage := ldr.SymPkg(owner)
		ownerSymbol := ldr.SymName(owner)
		if ownerPackage == "" || ownerSymbol == "" {
			continue
		}
		relocations := ldr.Relocs(owner)
		for relocationIndex := 0; relocationIndex < relocations.Count(); relocationIndex++ {
			target := relocations.At(relocationIndex).Sym()
			if target == 0 {
				continue
			}
			targetPackage := ldr.SymPkg(target)
			targetSymbol := ldr.SymName(target)
			facts = append(facts, gomadcap.RelocationFacts(ownerPackage, ownerSymbol, targetPackage, targetSymbol, false)...)
		}
	}
	record, err := gomadcap.Encode(gomadcap.Input{
		Facts: facts, GoVersion: buildcfg.Version, GOARCH: buildcfg.GOARCH, GOOS: buildcfg.GOOS,
		ToolchainBuildKey: *flagGomadCapability,
	})
	if err != nil {
		Exitf("-gomadcap: %v", err)
	}
	manifest := ldr.LookupOrCreateSym(gomadcap.ReservedSymbol, 0)
	builder := ldr.MakeSymbolUpdater(manifest)
	if builder.Type() != 0 {
		Exitf("-gomadcap: reserved symbol %s is already defined", gomadcap.ReservedSymbol)
	}
	builder.SetType(sym.SRODATA)
	builder.SetAlign(8)
	builder.SetNotInSymbolTable(false)
	builder.AddBytes(record)
	ldr.SetAttrReachable(manifest, true)
}
