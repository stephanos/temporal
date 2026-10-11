package architecture

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
)

var jsonSummarySourcePins = map[string]map[string]string{
	"encoding/json": {
		"v2_decode.go":  "d82e2e8ac32bc64c967b02769fc1fc6e2e5e9d7aae384673709e6af4669bc3fe",
		"v2_encode.go":  "8d3601b3a4c652b90ece3b9c5f5b1bcb7ad155eb85e625372a158c1dd9e8c414",
		"v2_indent.go":  "f8ac9f3dba216cc5577ca731268db70c7667a1c0d7bb61f436ffd3db92a8f500",
		"v2_inject.go":  "f1fbf89a12eab3f6f6e53d23d801cd4c5f3a2901c0f20aaa3369c9e025560e58",
		"v2_options.go": "253e7ccb9c38463ab47b5c78e1bc1f103b2e95f976dfe0805e23f4922d59ab0a",
		"v2_scanner.go": "d3f720d1a0a001f010c8556dc01507d99d78d6640c953f95c256a6c398fa20ad",
		"v2_stream.go":  "ae20daf462b86bf14884637fc2e8256aaadd9bba91c580fbb6840020e6a12898",
	},
	"encoding/json/v2": {
		"arshal.go":          "54dc9704a72fe6836422f3ca4a216d32c55e5ffdf964c84ccb868e171997d5eb",
		"arshal_any.go":      "bdd67f2105b186fbd39c8de10f6997070a11e69717af69f3d803c8ae5f34ec2e",
		"arshal_default.go":  "c786a1d08a38eb0fee2df1ef68c7d7ce3415d8ad5cbc500a538c1bcec9be7515",
		"arshal_embedded.go": "ee4e745cfdf4fbfbdd776118210de8fbb0764dfa797a6d05f56829b57586f0e2",
		"arshal_funcs.go":    "077d959d3e6de98c0988bc6fadd65b2e8cf88efb7351d57e384bce01facb03b0",
		"arshal_methods.go":  "f4f9841376d866ad8d6eb40a05fddbc2d754d70383ad64b78b0fc0462c16f805",
		"arshal_time.go":     "8d3a8e932babbf2f445ba0f7bf954094b8e984bc50c8a066476cb7fd2c62b428",
		"doc.go":             "ba6bf2c47dd3f875b40ca1216e1c107b50203f7196e431edf11aff591459c450",
		"errors.go":          "029a385fcdca05f9ac1194b49d80202a895c00e569d10f306fd419915e23b784",
		"fields.go":          "963e7621d53f720324005329de0c37079092043a0e41e222b4256f5df277dd14",
		"fold.go":            "eb417126dce6f144bc45ebe5aa36b520242d4f111c9bbfca2c4e3d2a327fc2a0",
		"intern.go":          "69460857d5e78a9c1dc9df31bb8690245e0b2f4f38e2548dfedd4e1b979cacca",
		"options.go":         "c85d7730ddf636f328e82166207a309f93fe6791d789f9c387ec241e9cc32edd",
	},
	"encoding/json/jsontext": {
		"decode.go":  "6978277353d629e9fa02a3a351df55d647a067918c33fff6617ccaf307656fc8",
		"doc.go":     "57da19ab10afee48efbf39f5e8d71bfc95cd732670d989487f5a153731704918",
		"encode.go":  "1ad3b78ff3e407a13a3f3d05f3cc7ecba92fcd7a3edc95c14c0f65d09e932ef8",
		"errors.go":  "782a2da67bce8694e02280f808993f2747747d9154861ca83dd15d308a3a0538",
		"export.go":  "cd5e1f2c6c0642107a9128824154fbd3ddd7a2c72c7329a42e4dc35e42416409",
		"options.go": "4b3fe260c216e5b194f1cf7bb362c8bfdcd9964c63bce9519f3647dcbeb06b47",
		"pools.go":   "d4469d019ee19bde54d2c3b2e1bbceea831acc91290ad82938c62a98b9db3ec2",
		"quote.go":   "3e9445288c8fd79f56ca21112e3bad0ee8e4c097d52c9cf202b66f44514476ee",
		"state.go":   "07930c12fd4a0cd219859f9cd7257cb9e3d71f7581e32f73510c4ddc4c8cc08f",
		"token.go":   "48e3b2b49b0abb6230f4b9083d59e49b07cef6ca98000257ae65507b039c403d",
		"value.go":   "e113bab38d4f93ac5c88ed48560366735298c89d22fa542d16b3310d14c3556e",
	},
	"encoding/json/internal":           {"internal.go": "f7e972ba617bdf0b06d9ff7ceb612ae45fec9fe386b1085b6f746f5d468b2c5d"},
	"encoding/json/internal/jsonflags": {"flags.go": "f53bf09eb0b28ecd9e79b845a1672b9d619570285e88baa1f3c95d14ff0852f2"},
	"encoding/json/internal/jsonopts": {
		"options.go":        "8aaafad353441d2ffd12b4217df815cb202d605c728ba075b0087b55651a4998",
		"options_format.go": "40a274b935daca1c0dc57993e815917ac1d23fc2a621c1a5f2a312209a6ca7a3",
	},
	"encoding/json/internal/jsonwire": {
		"decode.go": "2ec579bfd801743cb8073f89d35c1f5540811e435b53793b2732cf57423e2eaa",
		"encode.go": "b0a9c70c49442959833ba37dd669bedc0fe7a2f1ab16cadbc7f3205dd2c6b934",
		"wire.go":   "f7e0e72cf8624ebb19501b79f5c3be45a10419be52fd027ec3c5a8caf0b8944c",
	},
}

type jsonPackageSourceSnapshot struct {
	pkg       *SourcePackage
	types     *types.Package
	info      *types.Info
	metadata  Package
	files     []*ast.File
	digests   map[string]string
	imports   []*types.Package
	functions map[string]jsonSourceFunction
}

type jsonSourceFunction struct {
	object      *types.Func
	declaration *ast.FuncDecl
}

type jsonSourceQualification struct {
	profile  string
	packages map[string]*jsonPackageSourceSnapshot
}

func bindJSONPackageSource(pkg *SourcePackage, digests map[string]string) *jsonPackageSourceSnapshot {
	metadata := pkg.Metadata
	metadata.GoFiles = slices.Clone(metadata.GoFiles)
	metadata.CgoFiles = slices.Clone(metadata.CgoFiles)
	metadata.CompiledGoFiles = slices.Clone(metadata.CompiledGoFiles)
	metadata.Imports = slices.Clone(metadata.Imports)
	metadata.ImportMap = maps.Clone(metadata.ImportMap)
	functions := map[string]jsonSourceFunction{}
	for _, file := range pkg.Files {
		for _, declaration := range file.Decls {
			if declaration, ok := declaration.(*ast.FuncDecl); ok {
				if object, ok := pkg.Info.Defs[declaration.Name].(*types.Func); ok {
					functions[functionID(object)] = jsonSourceFunction{object: object, declaration: declaration}
				}
			}
		}
	}
	return &jsonPackageSourceSnapshot{pkg: pkg, types: pkg.Types, info: pkg.Info, metadata: metadata, files: slices.Clone(pkg.Files), digests: maps.Clone(digests), imports: slices.Clone(pkg.Types.Imports()), functions: functions}
}

func sameJSONSourceMetadata(current, bound Package) bool {
	return current.Standard && current.ImportPath == bound.ImportPath && current.Dir == bound.Dir &&
		current.Error == nil && len(current.DepsErrors) == 0 &&
		slices.Equal(current.GoFiles, bound.GoFiles) && slices.Equal(current.CgoFiles, bound.CgoFiles) &&
		slices.Equal(current.CompiledGoFiles, bound.CompiledGoFiles) && slices.Equal(current.Imports, bound.Imports) && maps.Equal(current.ImportMap, bound.ImportMap)
}

func (a *effectAnalysis) checkJSONSummarySource(fn *types.Func, pos token.Pos) bool {
	if fn == nil || fn.Pkg() == nil {
		a.report("unresolved-effect", pos, "JSON summary requires an imported function")
		return false
	}
	path := fn.Origin().Pkg().Path()
	if _, pinned := jsonSummarySourcePins[path]; !pinned {
		a.report("unresolved-effect", pos, "JSON summary requires a pinned package declaration")
		return false
	}
	imported := a.program.Packages[path]
	declaration := a.program.Functions[functionID(fn)]
	if imported == nil || imported.Types != fn.Origin().Pkg() || declaration == nil || declaration.pkg != imported || declaration.object.Origin() != fn.Origin() {
		a.report("unresolved-effect", pos, "JSON summary requires native declaration identity")
		return false
	}
	source := a.program.jsonSources[path]
	if source == nil || source.functions[functionID(fn)].object != fn.Origin() || source.functions[functionID(fn)].declaration != declaration.declaration {
		a.report("unresolved-effect", pos, "JSON summary requires parsed declaration identity")
		return false
	}
	if len(jsonSummarySourcePins) != 7 {
		a.report("unresolved-effect", pos, "JSON summary requires a recognized source profile")
		return false
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte("gomad3-jsonv2-callback-source-profile-v1\x00"))
	paths := slices.Sorted(maps.Keys(jsonSummarySourcePins))
	for _, path := range paths {
		pins := jsonSummarySourcePins[path]
		snapshot := a.program.jsonSources[path]
		pkg := a.program.Packages[path]
		if snapshot == nil || pkg == nil || pkg != snapshot.pkg || pkg.Types != snapshot.types || pkg.Info != snapshot.info ||
			!slices.Equal(pkg.Files, snapshot.files) || !slices.Equal(pkg.Types.Imports(), snapshot.imports) ||
			!sameJSONSourceMetadata(a.program.Metadata[path], snapshot.metadata) || !sameJSONSourceMetadata(pkg.Metadata, snapshot.metadata) ||
			snapshot.metadata.ImportPath != path || !snapshot.metadata.Standard || len(snapshot.metadata.CgoFiles) != 0 ||
			len(snapshot.metadata.GoFiles) != len(pins) || len(snapshot.digests) != len(pins) {
			a.report("unresolved-effect", pos, "JSON summary source binding changed: "+path)
			return false
		}
		selected := map[string]bool{}
		for _, name := range snapshot.metadata.GoFiles {
			if selected[name] || filepath.Base(name) != name || pins[name] == "" || snapshot.digests[name] != pins[name] {
				a.report("unresolved-effect", pos, "JSON summary selected source profile changed: "+path)
				return false
			}
			selected[name] = true
		}
		for _, dependency := range snapshot.imports {
			if dependency == types.Unsafe {
				continue
			}
			imported := a.program.Packages[dependency.Path()]
			if imported == nil || imported.Types != dependency || !imported.Metadata.Standard || !a.program.Metadata[dependency.Path()].Standard {
				a.report("unresolved-effect", pos, "JSON summary imported source identity changed: "+dependency.Path())
				return false
			}
		}
		for _, file := range snapshot.files {
			for _, dependency := range file.Imports {
				name, err := strconv.Unquote(dependency.Path.Value)
				resolved := snapshot.metadata.ImportMap[name]
				if err != nil || (resolved != "" && resolved != name) {
					a.report("unresolved-effect", pos, "JSON summary source import redirected: "+path)
					return false
				}
			}
		}
		for _, name := range slices.Sorted(maps.Keys(pins)) {
			_, _ = digest.Write(fmt.Appendf(nil, "%s\x00%s\x00%s\x00", path, name, pins[name]))
		}
	}
	profile := fmt.Sprintf("%x", digest.Sum(nil))
	if cached := a.checkedJSON; cached != nil && cached.profile == profile {
		current := true
		for _, path := range paths {
			current = current && cached.packages[path] == a.program.jsonSources[path]
		}
		if current {
			return true
		}
	}
	for _, path := range paths {
		snapshot := a.program.jsonSources[path]
		for name, expected := range jsonSummarySourcePins[path] {
			data, err := os.ReadFile(filepath.Join(snapshot.metadata.Dir, name))
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != expected {
				a.report("unresolved-effect", pos, "JSON summary pinned source changed: "+path+"/"+name)
				return false
			}
		}
	}
	a.checkedJSON = &jsonSourceQualification{profile: profile, packages: maps.Clone(a.program.jsonSources)}
	return true
}
