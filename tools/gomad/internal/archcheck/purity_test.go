package archcheck

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

var host = Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}

//nolint:staticcheck // Tests execute on the compiling host and require that exact reviewed Go toolchain.
func goCommand() string { return filepath.Join(runtime.GOROOT(), "bin", "go") }

func purityFixture(t *testing.T, files map[string]string) string {
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
	findings, err := CheckPurity(purityFixture(t, files), goCommand(), host, pure)
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

func TestEprefixedSyscallIsNotAConstant(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("EpollCreate1 is Linux-specific")
	}
	requireCategories(t, checkFixture(t, map[string]string{"record/pure.go": `package record;import "syscall";func F(){_,_=syscall.EpollCreate1(0)}`}, allPure), "host-effect")
}

func TestConcreteStdlibInterfaceRequiresPureMethods(t *testing.T) {
	findings := checkFixture(t, map[string]string{
		"record/pure.go": `package record;import "container/heap";type Queue []int;func F(q *Queue){heap.Push(q,1)}`,
		"record/host.go": `package record;import "time";func(q Queue)Len()int{return len(q)};func(q Queue)Less(i,j int)bool{return q[i]<q[j]};func(q Queue)Swap(i,j int){q[i],q[j]=q[j],q[i]};func(q *Queue)Push(value any){_=time.Now()};func(q *Queue)Pop()any{return 0}`,
	}, func(path string) bool { return path == "record/pure.go" })
	requireCategories(t, findings, "dynamic-call")
	findings = checkFixture(t, map[string]string{
		"record/pure.go": `package record;import "io";type Buffer struct{};func(*Buffer)Write(data []byte)(int,error){return len(data),nil};func F(){_,_=io.WriteString(&Buffer{},"x")}`,
		"record/host.go": `package record;import "time";func(*Buffer)WriteString(data string)(int,error){_=time.Now();return len(data),nil}`,
	}, func(path string) bool { return path == "record/pure.go" })
	requireCategories(t, findings, "dynamic-call")
	for _, method := range []string{"UnmarshalJSON(data []byte)error", "UnmarshalText(data []byte)error", "AppendText(data []byte)([]byte,error)", "IsZero()bool"} {
		findings = checkFixture(t, map[string]string{
			"record/pure.go": `package record;import "encoding/json";type Value struct{};func F(data []byte)error{var v Value;return json.Unmarshal(data,&v)}`,
			"record/host.go": `package record;import "os";func(*Value)` + method + `{_,_=os.ReadFile("x");panic(0)}`,
		}, func(path string) bool { return path == "record/pure.go" })
		requireCategories(t, findings, "dynamic-call")
	}
	findings = checkFixture(t, map[string]string{
		"record/pure.go": `package record;import "encoding/json";type Writer interface{Write([]byte)(int,error)};type Buffer struct{};func(Buffer)Write(data []byte)(int,error){return len(data),nil};func F(w Writer){_,_=json.Marshal(w)}`,
		"record/host.go": `package record;import "time";func(Buffer)MarshalJSON()([]byte,error){_=time.Now();return nil,nil}`,
	}, func(path string) bool { return path == "record/pure.go" })
	requireCategories(t, findings, "dynamic-call")
}

func TestStreamHelperRequiresPureCallers(t *testing.T) {
	findings := checkFixture(t, map[string]string{
		"record/pure.go": `package record;import("bytes";"encoding/json");func F(data []byte)bool{return next(json.NewDecoder(bytes.NewReader(data)))};func next(d *json.Decoder)bool{return d.More()}`,
		"record/host.go": `package record;import("encoding/json";"os");func G()bool{return next(json.NewDecoder(os.Stdin))}`,
	}, func(path string) bool { return path == "record/pure.go" })
	requireCategories(t, findings, "dynamic-call")
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
		{"pure-interface", "type Item struct{};type Writer interface{Write(Item)};type Buffer struct{};func(Buffer)Write(Item){};func F(w Writer){w.Write(Item{})}", nil},
		{"private-interface", "type writer interface{write([]byte)(int,error)};type buffer struct{};func(buffer)write(b []byte)(int,error){return len(b),nil};func f(w writer){_,_ = w.write(nil)}", nil},
		{"stdlib-implementable-interface", "type Writer interface{Write([]byte)(int,error)};type Buffer struct{};func(Buffer)Write(b []byte)(int,error){return len(b),nil};func F(w Writer){_,_ = w.Write(nil)}", []string{"dynamic-call"}},
		{"promoted-interface", "type Env interface{Get(string)string};type Box struct{Env};func F(b Box)string{return b.Get(\"HOME\")}", []string{"dynamic-call"}},
		{"promoted-pure-interface", "type Item struct{};type Writer interface{Write(Item)};type Buffer struct{};func(Buffer)Write(Item){};type Box struct{Writer};func F(b *Box){b.Write(Item{})}", nil},
		{"stream-parameters", `import("bufio";"encoding/json");func F(d *json.Decoder,w *bufio.Writer,e *json.Encoder){var x int;_ = d.Decode(&x);_ = w.Flush();_ = e.Encode(1)}`, []string{"dynamic-call", "dynamic-call", "dynamic-call"}},
		{"stream-argument", `import("bytes";"compress/gzip";"io");func F(r *gzip.Reader){_,_ = io.Copy(&bytes.Buffer{},r)}`, []string{"dynamic-call"}},
		{"stream-reassigned", `import("bytes";"encoding/json");func F(p *json.Decoder){d:=json.NewDecoder(bytes.NewReader(nil));d=p;_ = d.More()}`, []string{"dynamic-call"}},
		{"stream-helper", `import("bytes";"encoding/json");func F(data []byte)error{return next(json.NewDecoder(bytes.NewReader(data)))};func next(d *json.Decoder)error{if d.More(){return next(d)};_,err:=d.Token();return err}`, nil},
		{"stream-helper-value", `import("bytes";"encoding/json");var _ = next;func F(data []byte)error{return next(json.NewDecoder(bytes.NewReader(data)))};func next(d *json.Decoder)error{_,err:=d.Token();return err}`, []string{"dynamic-call"}},
		{"stream-exported-helper", `import "encoding/json";func Next(d *json.Decoder)bool{return d.More()}`, []string{"dynamic-call"}},
		{"stream-local", `import("bytes";"compress/gzip";"encoding/json";"io");func F(data []byte)error{var x int;d:=json.NewDecoder(bytes.NewReader(data));d.UseNumber();z,err:=gzip.NewReader(bytes.NewReader(data));if err!=nil{return err};_,_ = io.ReadAll(z);return d.Decode(&x)}`, nil},
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
	const writer = "package record;type Item struct{};type Writer interface{Write(Item)};type Buffer struct{};func(Buffer)Write(Item){};func F(w Writer){w.Write(Item{})}"
	for _, tc := range []struct {
		name, record, host string
		want               []string
	}{
		{"impure", writer, "type Buffer struct{};func(Buffer)Write(record.Item){}", []string{"dynamic-call"}},
		{"private-impure", strings.ReplaceAll(writer, "Writer", "writer"), "type buffer struct{};func(buffer)Write(record.Item){}", []string{"dynamic-call"}},
		{"irrelevant-method", writer, "type Buffer struct{};func(Buffer)Write(string){};var _ record.Item", nil},
		{"stdlib-file", "package record;type Source interface{Read([]byte)(int,error)};type Buffer struct{};func(Buffer)Read([]byte)(int,error){return 0,nil};func F(s Source){_,_ = s.Read(nil)}", "import \"os\";func G(){record.F(os.Stdin)}", []string{"dynamic-call"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings := checkFixture(t, map[string]string{
				"record/pure.go": tc.record,
				"host/host.go":   "package host;import \"example.invalid/fitness/record\"\n" + tc.host,
			}, func(path string) bool { return strings.HasPrefix(path, "record/") })
			requireCategories(t, findings, tc.want...)
		})
	}
}

func TestPackageAndSelectionErrors(t *testing.T) {
	findings, err := CheckPurity(purityFixture(t, map[string]string{"record/pure.go": "package record\nimport _ \"example.invalid/missing\""}), goCommand(), host, allPure)
	if err == nil {
		t.Fatal("missing package must stop loading")
	}
	requireCategories(t, findings, "package-error")
	_, err = CheckPurity(purityFixture(t, map[string]string{"record/pure.go": "package record"}), goCommand(), host, func(string) bool { return false })
	if err == nil || !strings.Contains(err.Error(), "matches no file") {
		t.Fatalf("empty selection: %v", err)
	}
}
