package architecture

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func canonicalNativeWriterFixture(t *testing.T) (*effectAnalysis, map[string]*ast.CallExpr, *SourcePackage) {
	t.Helper()
	program := canonicalWriteFixtureBeforeImport(t, `package record
import("bytes";"io";"crypto/sha256";"go.temporal.io/server/tools/gomad3/internal/canonicaljson")
var _=canonicaljson.WriteCanonicalJSON
func Root()io.Writer{return io.MultiWriter(&bytes.Buffer{},sha256.New())}
`, func(program *Program) {
		record := program.Metadata[program.Module+"/record"]
		record.Imports = []string{"bytes", "io", "crypto/sha256", canonicalWritePackage}
		program.Metadata[program.Module+"/record"] = record
	})
	root := program.Functions[program.Module+"/record.Root"]
	calls := map[string]*ast.CallExpr{}
	ast.Inspect(root.declaration.Body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if fn := calledObject(root.pkg.Info, call.Fun); fn != nil {
				calls[functionID(fn)] = call
			}
		}
		return true
	})
	return effectResolverAnalysis(program), calls, root.pkg
}

func canonicalNativeWriterConstructPair(t *testing.T) (*effectAnalysis, *abstractValue, *abstractValue, *abstractValue, token.Pos) {
	t.Helper()
	analysis, calls, pkg := canonicalNativeWriterFixture(t)
	hashCall, multiCall := calls["crypto/sha256.New"], calls["io.MultiWriter"]
	if hashCall == nil || multiCall == nil {
		t.Fatal("actual native constructor callsites absent")
	}
	hash, handled := analysis.canonicalNativeWriterConstructor(calledObject(pkg.Info, hashCall.Fun), nil, hashCall.Pos())
	if !handled || hash == nil {
		t.Fatalf("SHA constructor not handled: findings=%v", analysis.findings)
	}
	buffer := analysis.eval(pkg, multiCall.Args[0], environment{})
	before := bindingKey([]*abstractValue{buffer, hash})
	multi, handled := analysis.canonicalNativeWriterConstructor(calledObject(pkg.Info, multiCall.Fun), []*abstractValue{buffer, hash}, multiCall.Pos())
	if !handled || multi == nil || len(analysis.findings) != 0 {
		t.Fatalf("native constructor witness rejected: findings=%v", analysis.findings)
	}
	if bindingKey([]*abstractValue{buffer, hash}) != before {
		t.Fatal("constructing MultiWriter mutated or joined its original delegate graphs")
	}
	return analysis, multi, buffer, hash, multiCall.Pos()
}

func TestCanonicalNativeWriterConstructorKeepsExactMembersAndConcreteHash(t *testing.T) {
	analysis, multi, buffer, hash, pos := canonicalNativeWriterConstructPair(t)
	class, children, ok := analysis.canonicalNativeWriterDelegates(multi, pos)
	if !ok || class != "multi" || len(children) != 2 || children[0] != buffer || children[1] != hash || children[0] == children[1] {
		t.Fatalf("separate delegates lost: class=%s children=%v ok=%v", class, children, ok)
	}
	class, children, ok = analysis.canonicalNativeWriterDelegates(hash, pos)
	backend := analysis.program.Packages[canonicalSHA256Backend].Types.Scope().Lookup("Digest")
	if !ok || class != "hash" || len(children) != 0 || hash.builtin || len(hash.types) != 1 ||
		!types.Identical(hash.types[0], types.NewPointer(backend.Type())) {
		t.Fatalf("SHA acquired interface/native-flag authority instead of actual Digest witness: %#v", hash)
	}
	unownedHash := cloneValue(hash)
	unownedHash.nativeWriterOwned = false
	ownedKey, unownedKey := bindingKey([]*abstractValue{hash}), bindingKey([]*abstractValue{unownedHash})
	if ownedKey == unownedKey || strings.Replace(ownedKey, ";native-writer-owned", "", 1) != unownedKey {
		t.Fatal("owned fact must distinguish keys without changing legacy unowned bytes")
	}
	alias := multi
	if _, members, ok := analysis.canonicalNativeWriterDelegates(alias, pos); !ok || members[0] != buffer || members[1] != hash {
		t.Fatal("same-location pointer alias lost original members")
	}
	clone := cloneValue(multi)
	_, members, ok := analysis.canonicalNativeWriterDelegates(clone, pos)
	if !ok || len(members) != 2 || members[0] == members[1] ||
		clone.fields["$pointee"].fields["writers"].fields["0"] != members[0] ||
		clone.fields["$pointee"].fields["writers"].fields["1"] != members[1] {
		t.Fatal("clone did not preserve separate member aliases inside its graph")
	}
}

func TestCanonicalNativeWriterOptionalTrackingPreservesOrdinaryFallback(t *testing.T) {
	analysis, multi, _, _, pos := canonicalNativeWriterConstructPair(t)
	newFunction := analysis.program.Packages["crypto/sha256"].Types.Scope().Lookup("New").(*types.Func)
	for _, test := range []struct {
		name     string
		position token.Pos
		depth    int
	}{
		{"missing-local-callsite", token.NoPos, 0},
		{"binding-resolution", pos, 1},
		{"stdlib-callsite", newFunction.Pos(), 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			analysis.bindingDepth = test.depth
			before := len(analysis.findings)
			if _, handled := analysis.canonicalNativeWriterConstructor(newFunction, nil, test.position); handled || len(analysis.findings) != before {
				t.Fatalf("optional constructor tracking replaced ordinary unsupported-shape semantics: %v", analysis.findings)
			}
		})
	}
	analysis.bindingDepth = 0
	unowned := cloneValue(multi)
	unowned.nativeWriterOwned = false
	method := types.NewMethodSet(multi.types[0]).Lookup(nil, "Write").Obj().(*types.Func)
	before := len(analysis.findings)
	if _, handled := analysis.canonicalNativeWriterOperation(method, unowned, []*abstractValue{valueOf(types.NewSlice(types.Typ[types.Byte]))}, pos); handled || len(analysis.findings) != before {
		t.Fatalf("unowned native operation did not retain ordinary source traversal: %v", analysis.findings)
	}
	program := canonicalWriteFixtureBeforeImport(t, `package record
import("io";"go.temporal.io/server/tools/gomad3/internal/canonicaljson")
var _=canonicaljson.WriteCanonicalJSON
func Root(writers []io.Writer) io.Writer { return io.MultiWriter(writers...) }
`, func(program *Program) {
		record := program.Metadata[program.Module+"/record"]
		record.Imports = []string{"io", canonicalWritePackage}
		program.Metadata[program.Module+"/record"] = record
	})
	root := program.Functions[program.Module+"/record.Root"]
	call := root.declaration.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.CallExpr)
	analysis = effectResolverAnalysis(program)
	if _, handled := analysis.canonicalNativeWriterConstructor(calledObject(root.pkg.Info, call.Fun), []*abstractValue{valueOf(root.pkg.Info.TypeOf(call.Args[0]))}, call.Pos()); handled || len(analysis.findings) != 0 {
		t.Fatalf("expanded constructor acquired optional ownership or new findings: %v", analysis.findings)
	}
	program = canonicalWriterTargetFixture(t, `package record
import "go.temporal.io/server/tools/gomad3/world"
var _=world.Seed(1)
func Root()error{return nil}
`, canonicalWriterWorldBridge)
	choice := program.Functions[canonicalWriterWorldPackage+".choiceRank"]
	var hmacCall *ast.CallExpr
	ast.Inspect(choice.declaration.Body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok && functionID(calledObject(choice.pkg.Info, call.Fun)) == "crypto/hmac.New" {
			hmacCall = call
			return false
		}
		return true
	})
	if hmacCall == nil {
		t.Fatal("actual World higher-order SHA constructor call absent")
	}
	analysis = effectResolverAnalysis(program)
	constructor := analysis.eval(choice.pkg, hmacCall.Args[0], environment{})
	result, handled := analysis.standard(calledObject(choice.pkg.Info, hmacCall.Fun), nil, []*abstractValue{constructor, valueOf(types.NewSlice(types.Typ[types.Byte]))}, hmacCall.Pos())
	if !handled || result == nil || result.elements == nil || result.elements.nativeWriterOwned || len(analysis.findings) != 0 {
		t.Fatalf("actual HMAC higher-order constructor lost ordinary semantics or borrowed owned facts: result=%#v findings=%v", result, analysis.findings)
	}
}

func TestCanonicalNativeWriterExpandedFallbackRetainsHostileMembers(t *testing.T) {
	program := canonicalWriterTargetFixture(t, `package record
import("bytes";"io";"go.temporal.io/server/tools/gomad3/internal/canonicaljson")
type Sink struct{}
func host(){}
func(*Sink)Write(p []byte)(int,error){go host();return len(p),nil}
func Root()error{writers:=[]io.Writer{&bytes.Buffer{},&Sink{}};writer:=io.MultiWriter(writers...);return canonicaljson.WriteCanonicalJSON(writer,7)}
`, "")
	analysis, _ := canonicalWriterAnalyzeRoot(t, program)
	if !compressionHasFinding(analysis, "unresolved-effect", "canonical write writer has no fresh constructor/delegate certificate") ||
		!compressionHasFinding(analysis, "host-effect", "Sink.Write -> goroutine start") {
		t.Fatalf("expanded ordinary writer lost unsupported ownership or hostile member: %v", analysis.findings)
	}
}

func TestCanonicalNativeWriterConstructorRejectsBorrowedOrMutatedWitnesses(t *testing.T) {
	analysis, multi, _, hash, pos := canonicalNativeWriterConstructPair(t)
	for _, test := range []struct {
		name   string
		change func(*abstractValue) *abstractValue
	}{
		{"native-type-and-builtin-only", func(v *abstractValue) *abstractValue {
			forged := valueOf(v.types[0])
			forged.unknown = false
			forged.builtin = true
			return forged
		}},
		{"missing-owner", func(v *abstractValue) *abstractValue { v.nativeWriterOwned = false; return v }},
		{"wrong-location", func(v *abstractValue) *abstractValue { v.origin = token.NoPos; return v }},
		{"mixed-native-type", func(v *abstractValue) *abstractValue { v.types = append(v.types, hash.types[0]); return v }},
		{"delegate-replacement", func(v *abstractValue) *abstractValue {
			v.fields["$pointee"].fields["writers"].fields["0"] = cloneValue(hash)
			return v
		}},
		{"unknown-delegate", func(v *abstractValue) *abstractValue {
			v.fields[canonicalNativeWriterState].fields["1"].unknown = true
			return v
		}},
		{"join-with-unowned-same-type", func(v *abstractValue) *abstractValue { return join(v, zeroValue(v.types[0])) }},
		{"join-with-distinct-owned-state", func(v *abstractValue) *abstractValue { return join(v, cloneValue(v)) }},
		{"struct-value-copy", func(v *abstractValue) *abstractValue { return copyValue(v, v.types[0].(*types.Pointer).Elem()) }},
		{"constructor-name-only", func(v *abstractValue) *abstractValue {
			v.fields[canonicalNativeWriterState].text = "io.NewWriter"
			return v
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, _, ok := analysis.canonicalNativeWriterDelegates(test.change(cloneValue(multi)), pos); ok {
				t.Fatal("uncertified ownership/location/member graph acquired constructor authority")
			}
		})
	}
	actual := analysis.program.Packages["io"].Types.Scope().Lookup("MultiWriter").(*types.Func)
	lookalike := types.NewFunc(actual.Pos(), actual.Pkg(), actual.Name(), actual.Type().(*types.Signature))
	if _, handled := analysis.canonicalNativeWriterConstructor(lookalike, nil, multi.origin); !handled ||
		!compressionHasFinding(analysis, "unresolved-effect", "native declaration/source witness changed") {
		t.Fatal("lookalike function object borrowed actual source declaration identity")
	}
	for _, id := range []string{"io.MultiWriter", "crypto/sha256.New", canonicalSHA256Backend + ".New"} {
		t.Run("in-place-source-"+id, func(t *testing.T) {
			function := analysis.program.Functions[id]
			body := function.declaration.Body
			original := body.List[0]
			body.List[0] = &ast.EmptyStmt{Semicolon: original.Pos(), Implicit: false}
			defer func() { body.List[0] = original }()
			if _, _, ok := analysis.canonicalNativeWriterDelegates(multi, pos); ok && id == "io.MultiWriter" {
				t.Fatal("in-place MultiWriter source mutation retained constructor authority")
			}
			if id != "io.MultiWriter" {
				if _, _, ok := analysis.canonicalNativeWriterDelegates(hash, pos); ok {
					t.Fatal("in-place hash source mutation retained constructor authority")
				}
			}
		})
	}
}

func TestCanonicalNativeMultiWriterOperationsRetainEachCallbackAndConcreteError(t *testing.T) {
	for _, test := range []struct {
		name, declarations, operation, prelude, consume, category, detail string
	}{
		{"second-write-host", `func(*Sink)Write(p []byte)(int,error){go host();return len(p),nil}`, `Write([]byte{1})`, "", "", "host-effect", "Sink.Write -> goroutine start"},
		{"second-string-host", `func(*Sink)Write(p []byte)(int,error){return len(p),nil};func(*Sink)WriteString(s string)(int,error){go host();return len(s),nil}`, `WriteString("x")`, "", "", "host-effect", "Sink.WriteString -> goroutine start"},
		{"short-error-host", `type Failure struct{};func(Failure)Error()string{go host();return "short"};func(*Sink)Write(p []byte)(int,error){return 0,nil}`, `Write([]byte{1})`, `io.ErrShortWrite=Failure{};`, `if err!=nil{_=err.Error()};`, "host-effect", "Failure.Error -> goroutine start"},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := canonicalWriteFixtureBeforeImport(t, `package record
import("bytes";"io";"go.temporal.io/server/tools/gomad3/internal/canonicaljson")
var _=canonicaljson.WriteCanonicalJSON
type Sink struct{}
func host(){}
`+test.declarations+`
func Root()error{`+test.prelude+`writer:=io.MultiWriter(&bytes.Buffer{},&Sink{});_,err:=`+func() string {
				if test.name == "second-string-host" {
					return "writer.(io.StringWriter)."
				}
				return "writer."
			}()+test.operation+`;`+test.consume+`return err}
`, func(program *Program) {
				record := program.Metadata[program.Module+"/record"]
				record.Imports = []string{"bytes", "io", canonicalWritePackage}
				program.Metadata[program.Module+"/record"] = record
			})
			analysis, _ := canonicalWriterAnalyzeRoot(t, program)
			if !compressionHasFinding(analysis, test.category, test.detail) {
				t.Fatalf("indexed member/error callback lost: want=%s findings=%v", test.detail, analysis.findings)
			}
			if test.name == "short-error-host" && !compressionHasFinding(analysis, "unresolved-effect", "short-write error") {
				t.Fatalf("mutable exported error accidentally certified: %v", analysis.findings)
			}
		})
	}
}
