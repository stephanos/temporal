package puritycheck

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/archcheck"
)

var host = archcheck.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}

//nolint:staticcheck // Tests execute on the compiling host and require that exact reviewed Go toolchain.
func goCommand() string { return filepath.Join(runtime.GOROOT(), "bin", "go") }

func fixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files["go.mod"] = "module example.invalid/fitness\n\ngo 1.27.1\n"
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func checkFixture(t *testing.T, files map[string]string, pure func(string) bool) []Finding {
	t.Helper()
	findings, err := Check(fixture(t, files), goCommand(), host, pure)
	if err != nil {
		t.Fatal(err)
	}
	return findings
}

func allPure(string) bool { return true }

func requireCategories(t *testing.T, findings []Finding, want ...string) {
	t.Helper()
	got := make([]string, len(findings))
	for i, finding := range findings {
		got[i] = finding.Category
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("categories=%v want=%v findings=%+v", got, want, findings)
	}
}

func TestCalls(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         []string
	}{
		{"clock", `import "time"; func F(){ _ = time.Now() }`, []string{"host-effect"}},
		{"environment", `import "os"; func F(){ _ = os.Getenv("HOME") }`, []string{"host-effect"}},
		{"filesystem", `import "os"; func F(){ _,_ = os.ReadFile("x") }`, []string{"host-effect"}},
		{"process", `import "os/exec"; func F(){ _ = exec.Command("x") }`, []string{"host-effect"}},
		{"network", `import "net"; func F(){ _,_ = net.Dial("tcp","x") }`, []string{"host-effect"}},
		{"entropy", `import "crypto/rand"; func F(){ _,_ = rand.Read(nil) }`, []string{"host-effect"}},
		{"timer", `import "time"; func F(){ _ = time.NewTimer(1) }`, []string{"host-effect"}},
		{"goroutine", `func F(){ go func(){}() }`, []string{"host-effect"}},
		{"uncalled-method", `import "os"; type Value struct{}; func(Value)MarshalJSON()([]byte,error){ return os.ReadFile("x") }`, []string{"host-effect"}},
		{"uncalled-formatter", `import "time"; type Value struct{}; func(Value)String()string{ return time.Now().String() }`, []string{"host-effect", "host-effect"}},
		{"unknown-symbol", `import "time"; func F(){ _,_ = time.LoadLocationFromTZData("x",nil) }`, []string{"host-effect"}},
		{"parse-local", `import "time";func F(){_,_=time.Parse(time.RFC3339,"2026-10-12T00:00:00Z")}`, []string{"host-effect"}},
		{"date-local", `import "time";func F(){_=time.Date(2000,1,1,0,0,0,0,time.Local)}`, []string{"host-effect"}},
		{"format-location", `import "time";func F(value time.Time){_=value.Format(time.RFC3339);_=value.String();_,_=value.MarshalJSON();_=value.UnmarshalJSON(nil)}`, []string{"host-effect", "host-effect", "host-effect", "host-effect"}},
		{"indirect-location", `import("time";"fmt";"encoding/json");func F(value time.Time){_=fmt.Sprint(value);_,_=json.Marshal([]time.Time{value})}`, []string{"host-effect", "host-effect"}},
		{"utc", `import "time"; func F(){ _,_ = time.ParseInLocation("x","x",time.UTC) }`, nil},
		{"location-variable", `import "time"; func F(loc *time.Location){ _,_ = time.ParseInLocation("x","x",loc) }`, []string{"host-effect"}},
		{"utc-alias", `import "time"; func F(){ loc:=time.UTC; _,_ = time.ParseInLocation("x","x",loc) }`, []string{"host-effect"}},
		{"shadow-utc", `import "time"; func F(){ time:=struct{UTC *time.Location}{time.Local}; _ = time; }`, nil},
		{"memory", `import("bytes";"encoding/json";"time";"syscall");func F(){ var b bytes.Buffer;_ = json.NewEncoder(&b).Encode(1);_ = time.Date(2000,1,1,0,0,0,0,time.UTC);_ = syscall.EINVAL }`, nil},
		{"reviewed-memory-adapters", `import("bytes";"bufio";"crypto/hmac";"crypto/sha256";"embed";"reflect");var files embed.FS;func F(){var b bytes.Buffer;w:=bufio.NewWriterSize(&b,8);_,_=w.Write(nil);_,_=w.WriteString("x");_=w.Flush();_=hmac.New(sha256.New,nil);_,_=files.ReadFile("x");v:=reflect.ValueOf(&b);_=v.Elem().Addr();_=v.IsZero();_=reflect.StructTag("x").Get("x");_=reflect.StructField{}.IsExported()}`, nil},
		{"builtin", `func F(){ _ = len(make([]int,2));_ = new(int) }`, nil},
		{"syscall", `import "syscall";func F(){ _,_ = syscall.Open("x",0,0) }`, []string{"host-effect"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireCategories(t, checkFixture(t, map[string]string{"record/pure.go": "package record\n" + tc.source}, allPure), tc.want...)
		})
	}
}

func TestMixedFiles(t *testing.T) {
	findings := checkFixture(t, map[string]string{
		"record/pure.go": "package record\nfunc F(){ impure() }",
		"record/host.go": "package record\nimport \"time\";func impure(){_ = time.Now()}",
	}, func(path string) bool { return path == "record/pure.go" })
	requireCategories(t, findings, "host-effect")
	if !strings.Contains(findings[0].Detail, "impure") {
		t.Fatal(findings)
	}
}

func TestDynamicMarks(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         []string
	}{
		{"unmarked", "func F(f func()){ f() }", []string{"dynamic-call"}},
		{"previous-line", "func F(f func()){\n//gomad:dynamic reviewed 2026-10-12 supplied by the caller\nf()\n}", nil},
		{"same-line", "func F(f func()){ f() //gomad:dynamic reviewed 2026-10-12 supplied by caller\n}", nil},
		{"stale", "func F(){\n//gomad:dynamic reviewed 2026-10-12 no callback remains\n_ = len(\"x\")\n}", []string{"stale-review"}},
		{"undated", "func F(f func()){\n//gomad:dynamic reviewed supplied by caller\nf()\n}", []string{"dynamic-call", "stale-review"}},
		{"no-reason", "func F(f func()){\n//gomad:dynamic reviewed 2026-10-12\nf()\n}", []string{"dynamic-call", "stale-review"}},
		{"bad-date", "func F(f func()){\n//gomad:dynamic reviewed 2026-99-12 supplied by caller\nf()\n}", []string{"dynamic-call", "stale-review"}},
		{"host-mark", "import \"time\";func F(){\n//gomad:dynamic reviewed 2026-10-12 this cannot suppress the host\n_ = time.Now()\n}", []string{"host-effect", "stale-review"}},
		{"time-callback-mark", "import(\"time\";\"encoding/json\");func F(value time.Time){\n//gomad:dynamic reviewed 2026-10-12 known timezone callback cannot be reviewed away\n_,_=json.Marshal(value)\n}", []string{"host-effect", "stale-review"}},
		{"external-interface", "import \"io\";func F(w io.Writer){_,_ = w.Write(nil)}", []string{"dynamic-call"}},
		{"method-value", "import \"bytes\";func F(){var b bytes.Buffer;f:=b.Write;_,_ = f(nil)}", []string{"dynamic-call"}},
		{"pure-interface", "type Writer interface{Write([]byte)(int,error)};type Buffer struct{};func(Buffer)Write(b []byte)(int,error){return len(b),nil};func F(w Writer){_,_ = w.Write(nil)}", nil},
		{"private-interface", "type writer interface{Write([]byte)(int,error)};type buffer struct{};func(buffer)Write(b []byte)(int,error){return len(b),nil};func f(w writer){_,_ = w.Write(nil)}", nil},
		{"empty-interface-implementations", "type Writer interface{Write([]byte)(int,error)};func F(w Writer){_,_ = w.Write(nil)}", []string{"dynamic-call"}},
		{"stdlib-interface-input", "import \"encoding/json\";func F(v any){_,_ = json.Marshal(v)}", []string{"dynamic-call"}},
		{"stdlib-callback-value", "import \"slices\";func F(f func(int,int)int){slices.SortFunc([]int{},f)}", []string{"dynamic-call"}},
		{"stdlib-callback-literal", "import \"slices\";func F(){slices.SortFunc([]int{},func(a,b int)int{return a-b})}", nil},
		{"iterator", "func F(f func(func(int)bool)){for range f{}}", []string{"dynamic-call"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireCategories(t, checkFixture(t, map[string]string{"record/pure.go": "package record\n" + tc.source}, allPure), tc.want...)
		})
	}
}

func TestInterfaceImplementationsOutsidePurePackage(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         []string
	}{
		{"impure", "package host;type Buffer struct{};func(Buffer)Write(b []byte)(int,error){return len(b),nil}", []string{"dynamic-call"}},
		{"private-impure", "package host;type buffer struct{};func(buffer)Write(b []byte)(int,error){return len(b),nil}", []string{"dynamic-call"}},
		{"irrelevant-method", "package host;type Buffer struct{};func(Buffer)Write(string){}", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings := checkFixture(t, map[string]string{
				"record/pure.go": "package record;type Writer interface{Write([]byte)(int,error)};type Buffer struct{};func(Buffer)Write(b []byte)(int,error){return len(b),nil};func F(w Writer){_,_ = w.Write(nil)}",
				"host/host.go":   tc.source,
			}, func(path string) bool { return strings.HasPrefix(path, "record/") })
			requireCategories(t, findings, tc.want...)
		})
	}
}

func TestInterfaceImplementationUsesModuleTypes(t *testing.T) {
	for _, name := range []string{"Writer", "writer"} {
		t.Run(name, func(t *testing.T) {
			findings := checkFixture(t, map[string]string{
				"record/pure.go": strings.ReplaceAll("package record;type Item struct{};type Writer interface{Write(Item)};type Buffer struct{};func(Buffer)Write(Item){};func f(w Writer){w.Write(Item{})}", "Writer", name),
				"host/host.go":   "package host;import \"example.invalid/fitness/record\";type Buffer struct{};func(Buffer)Write(record.Item){}",
			}, func(path string) bool { return strings.HasPrefix(path, "record/") })
			requireCategories(t, findings, "dynamic-call")
		})
	}
}

func TestPackageAndSelectionErrors(t *testing.T) {
	findings, err := Check(fixture(t, map[string]string{"record/pure.go": "package record\nimport _ \"example.invalid/missing\""}), goCommand(), host, allPure)
	if err == nil {
		t.Fatal("missing package must stop loading")
	}
	requireCategories(t, findings, "package-error")
	_, err = Check(fixture(t, map[string]string{"record/pure.go": "package record"}), goCommand(), host, func(string) bool { return false })
	if err == nil || !strings.Contains(err.Error(), "matches no file") {
		t.Fatalf("empty selection: %v", err)
	}
}

func TestPublicSignatures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		leaks        bool
	}{
		{"nested-container", "type Report struct{Values map[string][]*leaf.Value}", true},
		{"alias", "type Report = leaf.Value", true},
		{"defined-rhs", "type Report leaf.Value", true},
		{"interface-result", "type Report interface{Value()leaf.Value}", true},
		{"generic-constraint", "type Report[T interface{~[]leaf.Value}] struct{Value T}", true},
		{"foreign-generic-argument", "type Report struct{Value atomic.Pointer[leaf.Value]}", true},
		{"promoted-method", "type hidden struct{};func(hidden)Value()leaf.Value{return leaf.Value{}};type Report struct{hidden}", true},
		{"private-storage", "type Report struct{value leaf.Value}", false},
		{"private-defined-graph", "type hidden struct{Text string};type Report hidden", false},
		{"recursive-public-graph", "type Report struct{Next *Report;Values []string}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := fixture(t, map[string]string{
				"internal/leaf/value.go": "package leaf\ntype Value struct{Child Child};type Child struct{Text string}",
				"report/value.go":        "package report\nimport(\"example.invalid/fitness/internal/leaf\";\"sync/atomic\");var _ leaf.Value;var _ atomic.Pointer[int]\n" + tc.source,
			})
			findings, err := PublicSignatures(root, goCommand(), host, "example.invalid/outside-consumer")
			if err != nil {
				t.Fatal(err)
			}
			if (len(findings) != 0) != tc.leaks {
				t.Fatalf("leaks=%t findings=%+v", tc.leaks, findings)
			}
			legal, err := PublicSignatures(root, goCommand(), host, "example.invalid/fitness/consumer")
			if err != nil || len(legal) != 0 {
				t.Fatalf("same-parent: findings=%+v err=%v", legal, err)
			}
		})
	}
}
