package architecture

import (
	"go/build"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type compressionEffectSummarizer interface {
	compressionSummary(*types.Func, *abstractValue, []*abstractValue, token.Pos) (*abstractValue, bool)
}

func compressionSemanticFixture(t *testing.T, source string) (*effectAnalysis, compressionEffectSummarizer, *SourcePackage) {
	t.Helper()
	program := &Program{Module: "example.invalid/compression", Files: token.NewFileSet(), Metadata: map[string]Package{}, Packages: map[string]*SourcePackage{}, Functions: map[string]*function{}}
	analysis := effectResolverAnalysis(program)
	_, ok := any(analysis).(compressionEffectSummarizer)
	if !ok {
		t.Fatal("constructor-bound compression effect summary is unavailable")
	}
	var metadata func(string)
	metadata = func(path string) {
		if path == "unsafe" || program.Metadata[path].ImportPath != "" {
			return
		}
		pkg, err := build.Default.Import(path, "", 0)
		if err != nil {
			t.Fatal(err)
		}
		program.Metadata[path] = Package{ImportPath: path, Dir: pkg.Dir, Standard: pkg.Goroot, GoFiles: pkg.GoFiles, Imports: pkg.Imports, SFiles: pkg.SFiles, HFiles: pkg.HFiles}
		for _, dependency := range pkg.Imports {
			metadata(dependency)
		}
	}
	metadata("compress/gzip")
	if strings.Contains(source, "\"fmt\"") {
		metadata("fmt")
	}
	const path = "example.invalid/compression/record"
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "pure.go"), []byte("package record\n"+source), 0600); err != nil {
		t.Fatal(err)
	}
	program.Metadata[path] = Package{ImportPath: path, Dir: directory, GoFiles: []string{"pure.go"}}
	if _, err := program.Import("compress/gzip"); err != nil {
		t.Fatal(err)
	}
	if _, err := program.Import(path); err != nil {
		t.Fatal(err)
	}
	analysis = effectResolverAnalysis(program)
	return analysis, any(analysis).(compressionEffectSummarizer), program.Packages[path]
}

func compressionNativeMethod(t *testing.T, a *effectAnalysis, name string) *types.Func {
	t.Helper()
	typ := a.program.Packages["compress/gzip"].Types.Scope().Lookup("Writer").Type()
	selection := types.NewMethodSet(types.NewPointer(typ)).Lookup(nil, name)
	if selection == nil {
		t.Fatalf("native gzip method %s absent", name)
	}
	return selection.Obj().(*types.Func)
}

func compressionConstruct(t *testing.T, a *effectAnalysis, summary compressionEffectSummarizer, writer *abstractValue) *abstractValue {
	t.Helper()
	fn := a.program.Packages["compress/gzip"].Types.Scope().Lookup("NewWriter").(*types.Func)
	result, handled := summary.compressionSummary(fn, nil, []*abstractValue{writer}, token.NoPos)
	if !handled || result == nil || len(a.findings) != 0 {
		t.Fatalf("native constructor not qualified: handled=%v result=%v findings=%v", handled, result, a.findings)
	}
	return result
}

func compressionHasFinding(a *effectAnalysis, category, detail string) bool {
	for _, finding := range a.findings {
		if finding.Category == category && strings.Contains(finding.Detail, detail) {
			return true
		}
	}
	return false
}

func TestCompressionSummaryPreservesWriterCallbacksForEveryOperation(t *testing.T) {
	for _, operation := range []string{"Write", "Flush", "Close"} {
		t.Run(operation, func(t *testing.T) {
			a, summary, pkg := compressionSemanticFixture(t, "type Sink struct{}\nfunc (Sink) Write([]byte)(int,error){go sideEffect();return 0,nil}\nfunc sideEffect(){}\n")
			writer := compressionConstruct(t, a, summary, valueOf(pkg.Types.Scope().Lookup("Sink").Type()))
			result, handled := summary.compressionSummary(compressionNativeMethod(t, a, operation), writer, []*abstractValue{valueOf(types.NewSlice(types.Typ[types.Byte]))}, token.NoPos)
			if !handled || result == nil || !compressionHasFinding(a, "host-effect", "Sink.Write -> goroutine start") {
				t.Fatalf("%s lost actual writer callback: handled=%v findings=%v", operation, handled, a.findings)
			}
		})
	}
}

func TestCompressionSummaryRetainsStringWriterAndDoesNotCloseUnderlyingWriter(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "type Sink struct{}\nfunc (Sink) Write([]byte)(int,error){return 0,nil}\nfunc (Sink) WriteString(string)(int,error){go stringEffect();return 0,nil}\nfunc (Sink) Close()error{go closeEffect();return nil}\nfunc stringEffect(){}\nfunc closeEffect(){}\n")
	writer := compressionConstruct(t, a, summary, valueOf(pkg.Types.Scope().Lookup("Sink").Type()))
	writer.fields["Header"].fields["Name"].text = "named-header"
	_, handled := summary.compressionSummary(compressionNativeMethod(t, a, "Close"), writer, nil, token.NoPos)
	if !handled || !compressionHasFinding(a, "host-effect", "Sink.WriteString -> goroutine start") || compressionHasFinding(a, "host-effect", "Sink.Close") {
		t.Fatalf("header/underlying-close callback contract changed: %v", a.findings)
	}
}

func TestCompressionSummaryKnownPointerReturnAliasPreservesWriter(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "import \"compress/gzip\"\ntype Sink struct{}\nfunc (Sink) Write([]byte)(int,error){go sideEffect();return 0,nil}\nfunc sideEffect(){}\nfunc Alias(g *gzip.Writer)*gzip.Writer{return g}\n")
	writer := compressionConstruct(t, a, summary, valueOf(pkg.Types.Scope().Lookup("Sink").Type()))
	alias := a.program.Functions["example.invalid/compression/record.Alias"]
	returned := a.call(&boundFunction{fn: alias, pkg: pkg}, []*abstractValue{writer}, token.NoPos)
	_, _ = summary.compressionSummary(compressionNativeMethod(t, a, "Close"), returned, nil, token.NoPos)
	if !compressionHasFinding(a, "host-effect", "Sink.Write -> goroutine start") {
		t.Fatalf("constructor pointer alias lost real writer callback: %v", a.findings)
	}
}

func TestCompressionSummaryDistinctOwnerJoinRejectsAuthority(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "type Sink struct{}\nfunc (Sink) Write([]byte)(int,error){return 0,nil}\n")
	sink := valueOf(pkg.Types.Scope().Lookup("Sink").Type())
	left := compressionConstruct(t, a, summary, sink)
	right := compressionConstruct(t, a, summary, sink)
	_, _ = summary.compressionSummary(compressionNativeMethod(t, a, "Reset"), join(left, right), []*abstractValue{sink}, token.NoPos)
	if !compressionHasFinding(a, "unresolved-effect", "constructor/location ownership") {
		t.Fatalf("joining distinct mutable owners invented alias authority: %v", a.findings)
	}
}

func TestCompressionSummaryPointeeOverwriteViaAliasRejects(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "import \"compress/gzip\"\ntype Sink struct{}\nfunc(Sink)Write([]byte)(int,error){return 0,nil}\nfunc Overwrite(g *gzip.Writer){alias:=g;*alias=gzip.Writer{}}\n")
	writer := compressionConstruct(t, a, summary, valueOf(pkg.Types.Scope().Lookup("Sink").Type()))
	overwrite := a.program.Functions["example.invalid/compression/record.Overwrite"]
	_ = a.call(&boundFunction{fn: overwrite, pkg: pkg}, []*abstractValue{writer}, token.NoPos)
	_, _ = summary.compressionSummary(compressionNativeMethod(t, a, "Close"), writer, nil, token.NoPos)
	if !compressionHasFinding(a, "unresolved-effect", "pointee overwrite") {
		t.Fatalf("whole-pointee assignment retained shell-only authority: %v", a.findings)
	}
}

func TestCompressionSummaryCallbackPointeeOverwriteRejects(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "import \"compress/gzip\"\ntype Sink struct{G *gzip.Writer}\nfunc(s *Sink)Write([]byte)(int,error){*s.G=gzip.Writer{};return 0,nil}\nfunc Link(s *Sink,g *gzip.Writer){s.G=g}\n")
	typ := pkg.Types.Scope().Lookup("Sink").Type()
	sink := zeroValue(typ)
	pointer := addressValue(sink, types.NewPointer(typ), token.NoPos)
	writer := compressionConstruct(t, a, summary, pointer)
	link := a.program.Functions["example.invalid/compression/record.Link"]
	_ = a.call(&boundFunction{fn: link, pkg: pkg}, []*abstractValue{pointer, writer}, token.NoPos)
	_, _ = summary.compressionSummary(compressionNativeMethod(t, a, "Close"), writer, nil, token.NoPos)
	if !compressionHasFinding(a, "unresolved-effect", "pointee overwrite") || !writer.fields["$gzip-state"].unknown {
		t.Fatalf("writer callback overwrite was not revalidated: %v", a.findings)
	}
}

func TestCompressionSummaryExportedErrorMutationRejects(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "import \"io\"\ntype Sink struct{}\nfunc(Sink)Write([]byte)(int,error){return 0,nil}\ntype Failure struct{}\nfunc(Failure)Error()string{go sideEffect();return \"failure\"}\nfunc sideEffect(){}\nfunc Change(){io.ErrShortWrite=Failure{}}\n")
	writer := compressionConstruct(t, a, summary, valueOf(pkg.Types.Scope().Lookup("Sink").Type()))
	returned, _ := summary.compressionSummary(compressionNativeMethod(t, a, "Close"), writer, nil, token.NoPos)
	if !compressionHasFinding(a, "unresolved-effect", "exported short-write error") || !returned.unknown {
		t.Fatalf("exported mutable error became a primitive-only result: %v", a.findings)
	}
}

func TestCompressionSummaryCallbackFreeWriterIsQualified(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "type Sink struct{}\nfunc(Sink)Write([]byte)(int,error){return 0,nil}\n")
	writer := compressionConstruct(t, a, summary, valueOf(pkg.Types.Scope().Lookup("Sink").Type()))
	_, handled := summary.compressionSummary(compressionNativeMethod(t, a, "Close"), writer, nil, token.NoPos)
	if !handled || len(a.findings) != 0 {
		t.Fatalf("known source-backed callback-free writer rejected: %v", a.findings)
	}
}

func TestCompressionSummaryRecursiveAndOpaqueCallbacksReject(t *testing.T) {
	for name, source := range map[string]string{
		"recursive": "type Sink struct{}\nfunc(s *Sink)Write(p []byte)(int,error){return s.Write(p)}\n",
		"opaque":    "type Sink struct{Next func()}\nfunc(s *Sink)Write([]byte)(int,error){s.Next();return 0,nil}\n",
	} {
		t.Run(name, func(t *testing.T) {
			a, summary, pkg := compressionSemanticFixture(t, source)
			typ := pkg.Types.Scope().Lookup("Sink").Type()
			writer := compressionConstruct(t, a, summary, addressValue(valueOf(typ), types.NewPointer(typ), token.NoPos))
			_, _ = summary.compressionSummary(compressionNativeMethod(t, a, "Close"), writer, nil, token.NoPos)
			if !compressionHasFinding(a, "unresolved-effect", "") {
				t.Fatalf("%s callback mutation acquired certainty: %v", name, a.findings)
			}
		})
	}
}

func TestCompressionSummaryStickyErrorPreservesIsAndUnwrapCallbacks(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "type Sink struct{}\nfunc(Sink)Write([]byte)(int,error){return 0,Failure{}}\ntype Failure struct{}\nfunc(Failure)Error()string{return \"failure\"}\nfunc(Failure)Is(error)bool{go isEffect();return false}\nfunc(Failure)Unwrap()error{go unwrapEffect();return nil}\nfunc isEffect(){}\nfunc unwrapEffect(){}\n")
	writer := compressionConstruct(t, a, summary, valueOf(pkg.Types.Scope().Lookup("Sink").Type()))
	returned, _ := summary.compressionSummary(compressionNativeMethod(t, a, "Close"), writer, nil, token.NoPos)
	is := a.program.Packages["errors"].Types.Scope().Lookup("Is").(*types.Func)
	_, _ = a.standard(is, nil, []*abstractValue{returned, returned}, token.NoPos)
	if !compressionHasFinding(a, "host-effect", "Failure.Is -> goroutine start") || !compressionHasFinding(a, "host-effect", "Failure.Unwrap -> goroutine start") {
		t.Fatalf("actual sticky error callbacks were replaced with native errors: %v", a.findings)
	}
}

func TestCompressionSummaryResetPointerAliasRetainsDirtyReplacement(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "type Clean struct{}\nfunc (Clean) Write([]byte)(int,error){return 0,nil}\ntype Dirty struct{}\nfunc (Dirty) Write([]byte)(int,error){go sideEffect();return 0,nil}\nfunc sideEffect(){}\n")
	writer := compressionConstruct(t, a, summary, valueOf(pkg.Types.Scope().Lookup("Clean").Type()))
	alias := writer
	_, handled := summary.compressionSummary(compressionNativeMethod(t, a, "Reset"), alias, []*abstractValue{valueOf(pkg.Types.Scope().Lookup("Dirty").Type())}, token.NoPos)
	if !handled {
		t.Fatal("known pointer alias Reset was not handled")
	}
	_, _ = summary.compressionSummary(compressionNativeMethod(t, a, "Flush"), writer, nil, token.NoPos)
	if !compressionHasFinding(a, "host-effect", "Dirty.Write -> goroutine start") {
		t.Fatalf("Reset replacement lost through alias: %v", a.findings)
	}
}

func TestCompressionSummaryUnknownReceiverRejectsRatherThanAcquiringAuthority(t *testing.T) {
	a, summary, _ := compressionSemanticFixture(t, "func Root(){}\n")
	fn := compressionNativeMethod(t, a, "Write")
	for _, receiver := range []*abstractValue{nil, valueOf(fn.Type().(*types.Signature).Recv().Type()), {unknown: true}} {
		a.findings = map[string]Finding{}
		_, handled := summary.compressionSummary(fn, receiver, nil, token.NoPos)
		if !handled || !compressionHasFinding(a, "unresolved-effect", "compression") {
			t.Fatalf("unknown native receiver acquired ownership: handled=%v findings=%v", handled, a.findings)
		}
	}
}

func TestCompressionSummaryStickyCustomErrorRemainsActual(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "type Sink struct{}\nfunc (Sink) Write([]byte)(int,error){return 0,Failure{}}\ntype Failure struct{}\nfunc (Failure) Error()string{go sideEffect();return \"failed\"}\nfunc sideEffect(){}\n")
	writer := compressionConstruct(t, a, summary, valueOf(pkg.Types.Scope().Lookup("Sink").Type()))
	_, _ = summary.compressionSummary(compressionNativeMethod(t, a, "Write"), writer, []*abstractValue{valueOf(types.NewSlice(types.Typ[types.Byte]))}, token.NoPos)
	if len(a.findings) != 0 {
		t.Fatalf("returning custom error invented Error callback: %v", a.findings)
	}
	returned, _ := summary.compressionSummary(compressionNativeMethod(t, a, "Close"), writer, nil, token.NoPos)
	a.implicit(returned, []string{"Error"}, nil, token.NoPos, map[types.Type]bool{})
	if !compressionHasFinding(a, "host-effect", "Failure.Error -> goroutine start") {
		t.Fatalf("sticky actual custom error was erased: %v", a.findings)
	}
}

func TestCompressionSummaryStickyErrorPreservesFormatCallback(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "import \"fmt\"\ntype Sink struct{}\nfunc(Sink)Write([]byte)(int,error){return 0,Failure{}}\ntype Failure struct{}\nfunc(Failure)Error()string{return \"failure\"}\nfunc(Failure)Format(fmt.State,rune){go formatEffect()}\nfunc formatEffect(){}\n")
	writer := compressionConstruct(t, a, summary, valueOf(pkg.Types.Scope().Lookup("Sink").Type()))
	returned, _ := summary.compressionSummary(compressionNativeMethod(t, a, "Close"), writer, nil, token.NoPos)
	sprintf := a.program.Packages["fmt"].Types.Scope().Lookup("Sprintf").(*types.Func)
	_, _ = a.standard(sprintf, nil, []*abstractValue{valueOf(types.Typ[types.String]), returned}, token.NoPos)
	if !compressionHasFinding(a, "host-effect", "Failure.Format -> goroutine start") {
		t.Fatalf("sticky writer error lost its actual formatter callback: %v", a.findings)
	}
}

func TestCompressionSummaryRepeatedWritesRetainCapturedMutation(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "type Sink struct{Ready bool}\nfunc (s *Sink) Write([]byte)(int,error){if s.Ready{go sideEffect()}else{s.Ready=true};return 0,nil}\nfunc sideEffect(){}\n")
	typ := pkg.Types.Scope().Lookup("Sink").Type()
	sink := zeroValue(typ)
	sink.fields["Ready"].truth = -1
	writer := compressionConstruct(t, a, summary, addressValue(sink, types.NewPointer(typ), token.NoPos))
	_, _ = summary.compressionSummary(compressionNativeMethod(t, a, "Close"), writer, nil, token.NoPos)
	if !compressionHasFinding(a, "host-effect", "*Sink.Write -> goroutine start") {
		t.Fatalf("repeated native writes lost callback mutation: %v", a.findings)
	}
}

func TestCompressionSummaryClosureCaptureMutationCannotAppearPure(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "type Sink struct{F func()}\nfunc(s *Sink)Write([]byte)(int,error){s.F();return 0,nil}\nfunc sideEffect(){}\nfunc Make()*Sink{ready:=false;return &Sink{F:func(){if ready{go sideEffect()}else{ready=true}}}}\n")
	makeSink := a.program.Functions["example.invalid/compression/record.Make"]
	// Isolate invocation-driven callback mutation from eager pure-file closure
	// discovery; this fixture exercises the writer summary's own saturation.
	a.program.Module = "example.invalid/invocation-only"
	sink := a.call(&boundFunction{fn: makeSink, pkg: pkg}, nil, token.NoPos)
	writer := compressionConstruct(t, a, summary, sink)
	_, _ = summary.compressionSummary(compressionNativeMethod(t, a, "Close"), writer, nil, token.NoPos)
	if !compressionHasFinding(a, "host-effect", "goroutine start") && !compressionHasFinding(a, "unresolved-effect", "") {
		t.Fatalf("unpublished captured scalar mutation was accepted as pure: %v", a.findings)
	}
}

func TestCompressionSummaryValueCopyRemainsUnresolved(t *testing.T) {
	a, summary, pkg := compressionSemanticFixture(t, "import \"compress/gzip\"\ntype Sink struct{}\nfunc (Sink) Write([]byte)(int,error){return 0,nil}\nfunc Copy(g *gzip.Writer)*gzip.Writer{copied:=*g;return &copied}\n")
	writer := compressionConstruct(t, a, summary, valueOf(pkg.Types.Scope().Lookup("Sink").Type()))
	_, _ = summary.compressionSummary(compressionNativeMethod(t, a, "Write"), writer, []*abstractValue{valueOf(types.NewSlice(types.Typ[types.Byte]))}, token.NoPos)
	copy := a.program.Functions["example.invalid/compression/record.Copy"]
	returned := a.call(&boundFunction{fn: copy, pkg: pkg}, []*abstractValue{writer}, token.NoPos)
	_, _ = summary.compressionSummary(compressionNativeMethod(t, a, "Reset"), returned, []*abstractValue{valueOf(pkg.Types.Scope().Lookup("Sink").Type())}, token.NoPos)
	if !compressionHasFinding(a, "unresolved-effect", "compression") {
		t.Fatalf("shared-compressor value copy acquired authority: %v", a.findings)
	}
}
