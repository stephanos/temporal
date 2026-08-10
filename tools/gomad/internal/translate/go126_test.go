package translate

import (
	"embed"
	"go/token"
	"io/fs"
	"strings"
	"testing"
	"unsafe"

	"github.com/dave/dst"
	"github.com/temporalio/gomad/gomadruntime"
	stdlibhooks "github.com/temporalio/gomad/internal/stdlib/hooks"
	simtesting "github.com/temporalio/gomad/internal/testing"
)

type go126NamedMap gomadruntime.Map[string, int]

func acceptGo126MapType[M gomadruntime.MapType[string, int]](M) {}

func TestGo126FIPSPackagesKeepAssembly(t *testing.T) {
	packages := []string{
		"crypto/internal/fips140/aes",
		"crypto/internal/fips140/aes/gcm",
		"crypto/internal/fips140/bigmod",
		"crypto/internal/fips140/edwards25519/field",
		"crypto/internal/fips140/nistec",
		"crypto/internal/fips140/sha256",
		"crypto/internal/fips140/sha3",
		"crypto/internal/fips140/sha512",
	}
	for _, pkg := range packages {
		if !go126KeepAsmPackages[pkg] {
			t.Errorf("Go 1.26 FIPS package %q does not retain its assembly implementation", pkg)
		}
	}
}

func TestGo126ThirdPartyPackagesKeepAssembly(t *testing.T) {
	packages := []string{
		"github.com/golang/snappy",
		"github.com/klauspost/compress/zstd/internal/xxhash",
		"golang.org/x/crypto/chacha20",
		"golang.org/x/crypto/chacha20poly1305",
		"golang.org/x/crypto/internal/poly1305",
	}
	for _, pkg := range packages {
		if !go126KeepAsmPackages[pkg] {
			t.Errorf("Go 1.26 package %q does not retain its assembly implementation", pkg)
		}
	}
}

func TestGo126EmbedAndFSRemainCompatible(t *testing.T) {
	var readFileFS fs.ReadFileFS = embed.FS{}
	_ = readFileFS
	for _, pkg := range []string{"embed", "io/fs"} {
		if !go126SkippedPackages[pkg] {
			t.Errorf("Go 1.26 package %q is translated across the embed.FS interface boundary", pkg)
		}
	}
}

func TestGo126SimulatedTestingContext(t *testing.T) {
	simulated := &simtesting.T{}
	if simulated.Context() == nil {
		t.Fatal("Context returned nil")
	}
}

func TestGo126MapTypeAcceptsNamedMaps(t *testing.T) {
	acceptGo126MapType(go126NamedMap{})
}

func TestGo126HookSelectorCanReuseAdapter(t *testing.T) {
	source := packageSelector{Pkg: "internal/sync", Selector: "runtime_nanotime"}
	target := packageSelector{Pkg: stdlibHooksPackage, Selector: "Sync_runtime_nanotime"}
	if got := hookSelector(source, target); got != target.Selector {
		t.Fatalf("hook selector = %q, want %q", got, target.Selector)
	}
}

func TestGo126InternalSyncHooksReuseSyncAdapters(t *testing.T) {
	expected := map[string]string{
		"runtime_rand":            "HashMaphash_runtime_rand",
		"runtime_SemacquireMutex": "Sync_runtime_SemacquireMutex",
		"runtime_Semrelease":      "Sync_runtime_Semrelease",
		"runtime_canSpin":         "Sync_runtime_canSpin",
		"runtime_doSpin":          "Sync_runtime_doSpin",
		"runtime_nanotime":        "Sync_runtime_nanotime",
		"throw":                   "Sync_throw",
		"fatal":                   "Sync_fatal",
	}
	for source, target := range expected {
		got, ok := go126Hooks[packageSelector{Pkg: "internal/sync", Selector: source}]
		if !ok {
			t.Errorf("Go 1.26 internal/sync hook %q is missing", source)
			continue
		}
		if got != (packageSelector{Pkg: stdlibHooksPackage, Selector: target}) {
			t.Errorf("Go 1.26 internal/sync hook %q = %#v, want adapter %q", source, got, target)
		}
	}
}

func TestGo126RuntimeHooks(t *testing.T) {
	expected := map[packageSelector]string{
		{Pkg: "internal/synctest", Selector: "Run"}:                    "InternalSynctest_Run",
		{Pkg: "internal/synctest", Selector: "Wait"}:                   "InternalSynctest_Wait",
		{Pkg: "internal/synctest", Selector: "IsInBubble"}:             "InternalSynctest_IsInBubble",
		{Pkg: "internal/synctest", Selector: "associate"}:              "InternalSynctest_associate",
		{Pkg: "internal/synctest", Selector: "disassociate"}:           "InternalSynctest_disassociate",
		{Pkg: "internal/synctest", Selector: "isAssociated"}:           "InternalSynctest_isAssociated",
		{Pkg: "internal/synctest", Selector: "acquire"}:                "InternalSynctest_acquire",
		{Pkg: "internal/synctest", Selector: "release"}:                "InternalSynctest_release",
		{Pkg: "internal/synctest", Selector: "inBubble"}:               "InternalSynctest_inBubble",
		{Pkg: "internal/race", Selector: "Acquire"}:                    "InternalRace_Acquire",
		{Pkg: "internal/race", Selector: "Release"}:                    "InternalRace_Release",
		{Pkg: "internal/race", Selector: "ReleaseMerge"}:               "InternalRace_ReleaseMerge",
		{Pkg: "internal/race", Selector: "Disable"}:                    "InternalRace_Disable",
		{Pkg: "internal/race", Selector: "Enable"}:                     "InternalRace_Enable",
		{Pkg: "internal/race", Selector: "Read"}:                       "InternalRace_Read",
		{Pkg: "internal/race", Selector: "ReadPC"}:                     "InternalRace_ReadPC",
		{Pkg: "internal/race", Selector: "ReadObjectPC"}:               "InternalRace_ReadObjectPC",
		{Pkg: "internal/race", Selector: "Write"}:                      "InternalRace_Write",
		{Pkg: "internal/race", Selector: "WritePC"}:                    "InternalRace_WritePC",
		{Pkg: "internal/race", Selector: "WriteObjectPC"}:              "InternalRace_WriteObjectPC",
		{Pkg: "internal/race", Selector: "ReadRange"}:                  "InternalRace_ReadRange",
		{Pkg: "internal/race", Selector: "WriteRange"}:                 "InternalRace_WriteRange",
		{Pkg: "internal/race", Selector: "Errors"}:                     "InternalRace_Errors",
		{Pkg: "crypto/internal/fips140", Selector: "fatal"}:            "CryptoInternalFips140_fatal",
		{Pkg: "crypto/internal/fips140", Selector: "getIndicator"}:     "CryptoInternalFips140_getIndicator",
		{Pkg: "crypto/internal/fips140", Selector: "setIndicator"}:     "CryptoInternalFips140_setIndicator",
		{Pkg: "crypto/internal/constanttime", Selector: "boolToUint8"}: "CryptoInternalConstanttime_boolToUint8",
		{Pkg: "crypto/subtle", Selector: "setDITEnabled"}:              "InternalRuntimeSys_EnableDIT",
		{Pkg: "crypto/subtle", Selector: "setDITDisabled"}:             "InternalRuntimeSys_DisableDIT",
		{Pkg: "crypto/fips140", Selector: "setBypass"}:                 "CryptoFips140_setBypass",
		{Pkg: "crypto/fips140", Selector: "isBypassed"}:                "CryptoFips140_isBypassed",
		{Pkg: "crypto/fips140", Selector: "unsetBypass"}:               "CryptoFips140_unsetBypass",
		{Pkg: "crypto/internal/sysrand", Selector: "fatal"}:            "Sync_fatal",
		{Pkg: "crypto/rand", Selector: "fatal"}:                        "Sync_fatal",
		{Pkg: "time", Selector: "runtimeNow"}:                          "Time_now",
		{Pkg: "time", Selector: "runtimeIsBubbled"}:                    "InternalSynctest_IsInBubble",
		{Pkg: "weak", Selector: "runtime_registerWeakPointer"}:         "Weak_runtime_registerWeakPointer",
		{Pkg: "weak", Selector: "runtime_makeStrongFromWeak"}:          "Weak_runtime_makeStrongFromWeak",
		{Pkg: "internal/syscall/unix", Selector: "vgetrandom"}:         "InternalSyscallUnix_vgetrandom",
		{Pkg: "golang.org/x/sys/unix", Selector: "vgetrandom"}:         "InternalSyscallUnix_vgetrandom",
	}
	for source, target := range expected {
		got, ok := go126Hooks[source]
		if !ok {
			t.Errorf("Go 1.26 runtime hook %#v is missing", source)
			continue
		}
		if got != (packageSelector{Pkg: stdlibHooksPackage, Selector: target}) {
			t.Errorf("Go 1.26 runtime hook %#v = %#v, want adapter %q", source, got, target)
		}
	}
}

func TestGo126HooksDoNotReplaceMethods(t *testing.T) {
	translator := packageTranslator{
		pkgPath: "internal/synctest",
		hooks: map[packageSelector]packageSelector{
			{Pkg: "internal/synctest", Selector: "Run"}: {Pkg: stdlibHooksPackage},
		},
	}
	method := &dst.FuncDecl{
		Name: dst.NewIdent("Run"),
		Recv: &dst.FieldList{List: []*dst.Field{{Type: dst.NewIdent("Bubble")}}},
	}
	if hook, ok := translator.hookForFunc(method); ok {
		t.Fatalf("method unexpectedly matched hook %#v", hook)
	}
}

func TestGo126AssemblyGlobalsRemainPackageSymbols(t *testing.T) {
	globals := []packageSelector{
		{Pkg: "crypto/internal/fips140/sha256", Selector: "_K"},
		{Pkg: "crypto/internal/fips140/sha512", Selector: "_K"},
	}
	for _, global := range globals {
		if !go126GlobalsDontTranslate[global] {
			t.Errorf("Go 1.26 assembly global %#v is not shared", global)
		}
	}
}

func TestGo126WeakPointerAdaptersRetainIdentity(t *testing.T) {
	value := 42
	pointer := unsafe.Pointer(&value)
	weak := stdlibhooks.Weak_runtime_registerWeakPointer(pointer)
	if weak != pointer {
		t.Fatalf("registered weak pointer = %p, want %p", weak, pointer)
	}
	if strong := stdlibhooks.Weak_runtime_makeStrongFromWeak(weak); strong != pointer {
		t.Fatalf("strong pointer = %p, want %p", strong, pointer)
	}
}

func TestGo126RuntimeMapConstantUsesSimulationRuntime(t *testing.T) {
	source := packageSelector{Pkg: "internal/runtime/maps", Selector: "Use64BitHash"}
	want := packageSelector{Pkg: gomadruntimePackage, Selector: "Use64BitHash"}
	if got := replacements[source]; got != want {
		t.Fatalf("Go 1.26 runtime map replacement = %#v, want %#v", got, want)
	}
}

func TestGo126ConstantTimeBoolConversion(t *testing.T) {
	if got := stdlibhooks.CryptoInternalConstanttime_boolToUint8(false); got != 0 {
		t.Fatalf("false converted to %d, want 0", got)
	}
	if got := stdlibhooks.CryptoInternalConstanttime_boolToUint8(true); got != 1 {
		t.Fatalf("true converted to %d, want 1", got)
	}
}

func TestGo126HashTrieMapUsesSimulationTypeOperations(t *testing.T) {
	simulatedMap := gomadruntime.NewMap[string, any]()
	hasher := gomadruntime.MapHasher(simulatedMap)
	if got := hasher(nil, 17); got != 17 {
		t.Fatalf("map hash = %d, want seed 17", got)
	}
	equal := gomadruntime.MapValueEqual(simulatedMap)
	left, right := any("value"), any("value")
	if !equal(unsafe.Pointer(&left), unsafe.Pointer(&right)) {
		t.Fatal("equal map values were reported unequal")
	}
}

func TestGo126HashTrieMapSelectorsAreRewritten(t *testing.T) {
	translator := packageTranslator{
		pkgPath:      "internal/sync",
		replacedPkgs: map[string]string{gomadruntimePackage: "translated/gomadruntime"},
	}
	tests := []struct {
		name string
		expr dst.Expr
		want string
	}{
		{
			name: "hasher",
			expr: &dst.SelectorExpr{X: dst.NewIdent("mapType"), Sel: dst.NewIdent("Hasher")},
			want: "MapHasher",
		},
		{
			name: "equal",
			expr: &dst.SelectorExpr{
				X:   &dst.SelectorExpr{X: dst.NewIdent("mapType"), Sel: dst.NewIdent("Elem")},
				Sel: dst.NewIdent("Equal"),
			},
			want: "MapValueEqual",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			replacement, ok := translator.go126HashTrieMapReplacement(test.expr)
			if !ok {
				t.Fatal("selector was not rewritten")
			}
			call, ok := replacement.(*dst.CallExpr)
			if !ok {
				t.Fatalf("replacement type = %T, want *dst.CallExpr", replacement)
			}
			fun, ok := call.Fun.(*dst.Ident)
			if !ok || fun.Name != test.want || fun.Path != "translated/gomadruntime" {
				t.Fatalf("replacement function = %#v, want %s", call.Fun, test.want)
			}
		})
	}
}

func TestGo126HashTrieMapTypeProbeIsRemoved(t *testing.T) {
	translator := packageTranslator{pkgPath: "internal/sync"}
	assignment := &dst.AssignStmt{
		Lhs: []dst.Expr{dst.NewIdent("mapType")},
		Tok: token.DEFINE,
		Rhs: []dst.Expr{dst.NewIdent("probe")},
	}
	if !translator.isGo126HashTrieMapTypeAssignment(assignment) {
		t.Fatal("mapType probe was not recognized")
	}
}

func TestGo126InternalSyncFunctionsAreNoRace(t *testing.T) {
	if !shouldMarkSyncFuncsNorace("internal/sync") {
		t.Fatal("Go 1.26 internal/sync functions remain race-instrumented")
	}
}

func TestGo126AcceptedRuntimeLinknames(t *testing.T) {
	noBody := []packageSelector{
		{Pkg: "crypto/internal/fips140hash", Selector: "sha3Unwrap"},
		{Pkg: "internal/runtime/maps", Selector: "typeString"},
		{Pkg: "internal/runtime/maps", Selector: "fatal"},
		{Pkg: "internal/runtime/maps", Selector: "rand"},
		{Pkg: "internal/runtime/maps", Selector: "typedmemmove"},
		{Pkg: "internal/runtime/maps", Selector: "typedmemclr"},
		{Pkg: "internal/runtime/maps", Selector: "newarray"},
		{Pkg: "internal/runtime/maps", Selector: "newobject"},
	}
	for _, linkname := range noBody {
		if !go126AcceptedNoBodyLinknames[linkname] {
			t.Errorf("Go 1.26 runtime-provided linkname %#v is not accepted", linkname)
		}
	}

	runtimeMaps := []string{
		"runtime_mapaccess1",
		"runtime_mapaccess2",
		"runtime_mapassign",
		"runtime_mapaccess1_fast32",
		"runtime_mapaccess2_fast32",
		"runtime_mapassign_fast32",
		"runtime_mapassign_fast32ptr",
		"runtime_mapdelete_fast32",
		"runtime_mapaccess1_fast64",
		"runtime_mapaccess2_fast64",
		"runtime_mapassign_fast64",
		"runtime_mapassign_fast64ptr",
		"runtime_mapdelete_fast64",
		"runtime_mapaccess1_faststr",
		"runtime_mapaccess2_faststr",
		"runtime_mapassign_faststr",
		"runtime_mapdelete_faststr",
	}
	for _, name := range runtimeMaps {
		want := packageSelector{Pkg: "runtime", Selector: strings.TrimPrefix(name, "runtime_")}
		got := go126AcceptedLinknames[packageSelector{Pkg: "internal/runtime/maps", Selector: name}]
		if got != want {
			t.Errorf("Go 1.26 runtime map linkname %q = %#v, want %#v", name, got, want)
		}
	}
}
