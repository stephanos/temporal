package architecture

import (
	"go/build"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func compressionEffectsFixture(t *testing.T, root, callbacks string, platform Platform) *Program {
	t.Helper()
	program := compressionStockFixturePlatform(t, platform)
	context := build.Default
	context.GOOS, context.GOARCH = platform.OS, platform.Arch
	context.CgoEnabled = false
	var discover func(string)
	discover = func(path string) {
		if program.Metadata[path].ImportPath != "" {
			return
		}
		selected, err := context.Import(path, "", 0)
		if err != nil {
			t.Fatal(err)
		}
		program.Metadata[path] = Package{ImportPath: path, Dir: selected.Dir, Standard: selected.Goroot, GoFiles: selected.GoFiles, Imports: selected.Imports, SFiles: selected.SFiles, HFiles: selected.HFiles}
		for _, dependency := range selected.Imports {
			discover(dependency)
		}
	}
	for _, path := range []string{"bytes", "fmt", "unsafe"} {
		discover(path)
	}
	directory := t.TempDir()
	for name, source := range map[string]string{
		"runner": "package runner\n" + callbacks,
		"record": "package record\n" + root,
	} {
		dir := filepath.Join(directory, name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "pure.go"), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		path := program.Module + "/" + name
		program.Metadata[path] = Package{ImportPath: path, Dir: dir, GoFiles: []string{"pure.go"}}
	}
	if _, err := program.Import(program.Module + "/record"); err != nil {
		t.Fatal(err)
	}
	return program
}

const compressionEffectsImports = `import ("compress/gzip"; "example.invalid/compression/runner")
`

func TestCompressionEffectsConstructorBoundedWriter(t *testing.T) {
	for _, platform := range []Platform{{"darwin", "arm64"}, {"linux", "amd64"}} {
		t.Run(platform.OS+"/"+platform.Arch, func(t *testing.T) {
			program := compressionEffectsFixture(t, compressionEffectsImports+`func Root() error {
sink := &runner.Bounded{Limit:1024}; writer:=gzip.NewWriter(sink)
writer.Name="record"; writer.Comment="bounded"; writer.Extra=[]byte("header")
if _,err:=writer.Write([]byte("record"));err!=nil{return err}
if err:=writer.Flush();err!=nil{return err}
alias:=runner.Alias(writer); alias.Reset(sink); return writer.Close()
}`, `import ("bytes"; "compress/gzip"; "errors")
type Bounded struct { Buffer bytes.Buffer; Used,Limit uint64 }
func(s *Bounded)Write(data []byte)(int,error){
if uint64(len(data))>s.Limit-s.Used{return 0,errors.New("capacity")}
n,err:=s.Buffer.Write(data);s.Used+=uint64(n);return n,err
}
func Alias(g *gzip.Writer)*gzip.Writer{return g}
`, platform)
			analysis := effectResolverAnalysis(program)
			constructor := stockSummaryFunction(t, analysis, "compress/gzip", "", "NewWriter")
			if _, handled := analysis.standard(constructor, nil, nil, token.NoPos); !handled {
				t.Fatal("native constructor summary is absent from production standard routing")
			}
			if findings := program.Effects(); len(findings) != 0 {
				t.Fatalf("constructor-backed bounded output rejected: %v", findings)
			}
		})
	}
}

func TestCompressionEffectsRetainsHostileBindings(t *testing.T) {
	for _, test := range []struct{ name, root, callbacks, category, detail string }{
		{"write", `func Root(){g:=gzip.NewWriter(runner.Dirty{});_,_=g.Write(nil)}`, `type Dirty struct{};func(Dirty)Write([]byte)(int,error){go effect();return 0,nil};func effect(){}`, "host-effect", "Dirty.Write -> goroutine start"},
		{"flush", `func Root(){g:=gzip.NewWriter(runner.Dirty{});_=g.Flush()}`, `type Dirty struct{};func(Dirty)Write([]byte)(int,error){go effect();return 0,nil};func effect(){}`, "host-effect", "Dirty.Write -> goroutine start"},
		{"close", `func Root(){g:=gzip.NewWriter(runner.Dirty{});_=g.Close()}`, `type Dirty struct{};func(Dirty)Write([]byte)(int,error){go effect();return 0,nil};func effect(){}`, "host-effect", "Dirty.Write -> goroutine start"},
		{"string-writer", `func Root(){g:=gzip.NewWriter(runner.Sink{});g.Name="name";_=g.Close()}`, `type Sink struct{};func(Sink)Write([]byte)(int,error){return 0,nil};func(Sink)WriteString(string)(int,error){go effect();return 0,nil};func effect(){}`, "host-effect", "Sink.WriteString -> goroutine start"},
		{"reset-alias", `func Root(){g:=gzip.NewWriter(runner.Clean{});alias:=g;alias.Reset(runner.Dirty{});_=g.Close()}`, `type Clean struct{};func(Clean)Write([]byte)(int,error){return 0,nil};type Dirty struct{};func(Dirty)Write([]byte)(int,error){go effect();return 0,nil};func effect(){}`, "host-effect", "Dirty.Write -> goroutine start"},
		{"repeated-callback", `func Root(){g:=gzip.NewWriter(&runner.Sink{});_=g.Close()}`, `type Sink struct{Ready bool};func(s *Sink)Write([]byte)(int,error){if s.Ready{go effect()}else{s.Ready=true};return 0,nil};func effect(){}`, "host-effect", "*Sink.Write -> goroutine start"},
		{"sticky-error", `func Root()string{g:=gzip.NewWriter(runner.Sink{});_,_=g.Write(nil);return g.Close().Error()}`, `type Sink struct{};func(Sink)Write([]byte)(int,error){return 0,Failure{}};type Failure struct{};func(Failure)Error()string{go effect();return "failure"};func effect(){}`, "host-effect", "Failure.Error -> goroutine start"},
		{"pointee-alias", `func Root(){g:=gzip.NewWriter(runner.Clean{});alias:=runner.Alias(g);*alias=gzip.Writer{};_=g.Close()}`, `import "compress/gzip";type Clean struct{};func(Clean)Write([]byte)(int,error){return 0,nil};func Alias(g *gzip.Writer)*gzip.Writer{return g}`, "unresolved-effect", "pointee overwrite"},
		{"callback-overwrite", `func Root(){s:=&runner.Sink{};g:=gzip.NewWriter(s);s.G=g;_=g.Close()}`, `import "compress/gzip";type Sink struct{G *gzip.Writer};func(s *Sink)Write([]byte)(int,error){*s.G=gzip.Writer{};return 0,nil}`, "unresolved-effect", "pointee overwrite"},
		{"value-copy", `func Root(){g:=gzip.NewWriter(runner.Clean{});copy:=*g;_=copy.Close()}`, `type Clean struct{};func(Clean)Write([]byte)(int,error){return 0,nil}`, "unresolved-effect", "constructor/location ownership"},
		{"unconstructed", `func Root(g *gzip.Writer){_ = runner.Unused{};_=g.Close()}`, `type Unused struct{}`, "unresolved-effect", "constructor/location ownership"},
		{"opaque-writer", `func Root(){g:=gzip.NewWriter(runner.Unknown());_=g.Close()}`, `import "io";func Unknown()io.Writer{return nil}`, "unresolved-effect", "unproven dynamic Writer binding"},
		{"recursive-callback", `func Root(){g:=gzip.NewWriter(&runner.Sink{});_=g.Close()}`, `type Sink struct{};func(s *Sink)Write(p []byte)(int,error){return s.Write(p)}`, "unresolved-effect", "compression callback recursion"},
		{"dot-import-mutable-error", `func Root(){runner.Change();g:=gzip.NewWriter(runner.Sink{});_=g.Close()}`, `import . "io";type Sink struct{};func(Sink)Write([]byte)(int,error){return 0,nil};type Failure struct{};func(Failure)Error()string{return "failure"};func Change(){ErrShortWrite=Failure{}}`, "unresolved-effect", "exported short-write error"},
		{"dot-import-error-address", `func Root(){runner.Change();g:=gzip.NewWriter(runner.Sink{});_=g.Close()}`, `import . "io";type Sink struct{};func(Sink)Write([]byte)(int,error){return 0,nil};func Change(){_ = &ErrShortWrite}`, "unresolved-effect", "exported short-write error"},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := compressionEffectsFixture(t, compressionEffectsImports+test.root, test.callbacks, Platform{"linux", "amd64"})
			findings := program.Effects()
			for _, finding := range findings {
				if finding.Category == test.category && strings.Contains(finding.Detail, "/record.Root ->") && strings.Contains(finding.Detail, test.detail) {
					return
				}
			}
			t.Fatalf("Root lost %s %q: %v", test.category, test.detail, findings)
		})
	}
}
