package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEffectThroughHelper(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{"go.mod": "module example.invalid/architecture\n\ngo 1.27.1\n", "record/pure.go": `package record; import "time"; func Validate() { helper() };func helper(){_ = time.Now()}`} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	p, err := Load(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	findings := p.Effects()
	if len(findings) == 0 || findings[0].Category != "host-effect" {
		t.Fatalf("transitive host clock escaped: %v", findings)
	}
}

func TestMemorySummarySourceIdentity(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "record"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"go.mod":         "module example.invalid/architecture\n\ngo 1.27.1\n",
		"record/pure.go": `package record;import "strings";func Check()bool{return strings.Contains("owned","own")}`,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	program, err := Load(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	metadata := program.Metadata["strings"]
	if findings := program.Effects(); len(findings) != 0 {
		t.Fatalf("stock source rejected: %v", findings)
	}
	for _, test := range []struct {
		name     string
		metadata Package
		detail   string
	}{
		{"foreign-inspected-body", func() Package { changed := metadata; changed.Standard = false; return changed }(), "bodyless call internal/bytealg."},
		{"changed-source", func() Package {
			changed := metadata
			changed.Dir = t.TempDir()
			if err := os.WriteFile(filepath.Join(changed.Dir, "strings.go"), []byte("package strings\n"), 0600); err != nil {
				t.Fatal(err)
			}
			return changed
		}(), "pinned memory summary source changed: strings"},
	} {
		t.Run(test.name, func(t *testing.T) {
			program.Metadata["strings"] = test.metadata
			findings := program.Effects()
			for _, finding := range findings {
				if finding.Category == "unresolved-effect" && strings.Contains(finding.Detail, test.detail) {
					return
				}
			}
			t.Fatal("unqualified memory summary escaped")
		})
	}
}

func TestPureRootStoredClosureIsChecked(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{"go.mod": "module example.invalid/architecture\n\ngo 1.27.1\n", "record/pure.go": `package record;import "time";func Callback()func(){return func(){_ = time.Now()}}`} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	program, err := Load(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range program.Effects() {
		if finding.Category == "host-effect" && strings.Contains(finding.Detail, "time.Now") {
			return
		}
	}
	t.Fatal("stored pure-root closure escaped")
}

func TestEffectCallbackContextsAndUnwrapReturns(t *testing.T) {
	for _, test := range []struct{ name, root, helper string }{
		{"argument-order", `helper.First(clean,dirty);helper.First(dirty,clean)`, `func First(a,b func()){a()}`},
		{"captured-order", `makeCall:=func(f func())func(){return func(){f()}};helper.First(makeCall(clean),clean);helper.First(makeCall(dirty),clean)`, `func First(a,b func()){a()}`},
		{"single-unwrap", `_ = errors.Is(helper.Outer{},errors.New("target"))`, `type Outer struct{};func(Outer)Error()string{return "outer"};func(Outer)Unwrap()error{return Leaf{}};type Leaf struct{};func(Leaf)Error()string{return "leaf"};func(Leaf)Is(error)bool{_ = time.Now();return false}`},
		{"joined-unwrap", `_ = errors.Is(errors.Join(helper.Outer{}),errors.New("target"))`, `type Outer struct{};func(Outer)Error()string{return "outer"};func(Outer)Unwrap()error{return Leaf{}};type Leaf struct{};func(Leaf)Error()string{return "leaf"};func(Leaf)Is(error)bool{_ = time.Now();return false}`},
		{"wrapped-is", `_ = errors.Is(fmt.Errorf("%w",helper.Leaf{}),errors.New("target"))`, `type Leaf struct{};func(Leaf)Error()string{return "leaf"};func(Leaf)Is(error)bool{_ = time.Now();return false}`},
		{"interface-return", `var source helper.Supplier=helper.Source{};source.Get().Error()`, `type Supplier interface{Get()error};type Source struct{};func(Source)Get()error{return Leaf{}};type Leaf struct{};func(Leaf)Error()string{_ = time.Now();return "leaf"}`},
		{"multi-unwrap", `_ = errors.Is(helper.Outer{},errors.New("target"))`, `type Outer struct{};func(Outer)Error()string{return "outer"};func(Outer)Unwrap()[]error{return []error{Leaf{}}};type Leaf struct{};func(Leaf)Error()string{return "leaf"};func(Leaf)Is(error)bool{_ = time.Now();return false}`},
		{"unwrap-return", `errors.Unwrap(helper.Outer{}).Error()`, `type Outer struct{};func(Outer)Error()string{return "outer"};func(Outer)Unwrap()error{return Leaf{}};type Leaf struct{};func(Leaf)Error()string{_ = time.Now();return "leaf"}`},
		{"recursive-wrapper", `helper.Wrap(1,helper.Leaf{}).Error()`, `type Outer struct{Child error};func(o Outer)Error()string{return o.Child.Error()};func Wrap(n int,e error)error{if n==0{return e};return Outer{Child:Wrap(n-1,e)}};type Leaf struct{};func(Leaf)Error()string{_ = time.Now();return "leaf"}`},
		{"recursive-returned-callback", `helper.Wrap(1,dirty)()`, `func Wrap(n int,f func())func(){if n==0{return f};return Wrap(n-1,func(){f()})}`},
		{"global-callback-tuple", `helper.First()`, `func Dirty(){_ = time.Now()};func Clean(){};func Values()(func(),func()){return Dirty,Clean};var first,second=Values();func First(){first()}`},
		{"cross-argument-alias", `helper.Probe(&helper.Box{F:clean},&helper.Box{F:clean});c:=&helper.Box{F:clean};helper.Probe(c,c)`, `type Box struct{F func()};func Dirty(){_ = time.Now()};func Probe(x,y *Box){x.F=Dirty;y.F()}`},
		{"cached-mutation", `a:=&helper.Box{F:clean};b:=&helper.Box{F:clean};helper.Set(a);helper.Set(b);b.F()`, `type Box struct{F func()};func Dirty(){_ = time.Now()};func Set(x *Box){x.F=Dirty}`},
		{"json-marshaler-error-return", `_,err:=json.Marshal(helper.Value{});_ = err.Error()`, `type Value struct{};func(Value)MarshalJSON()([]byte,error){return nil,Leaf{}};type Leaf struct{};func(Leaf)Error()string{_ = time.Now();return "leaf"}`},
		{"json-writer-error-return", `err:=json.NewEncoder(helper.Writer{}).Encode(1);_ = err.Error()`, `type Writer struct{};func(Writer)Write([]byte)(int,error){return 0,Leaf{}};type Leaf struct{};func(Leaf)Error()string{_ = time.Now();return "leaf"}`},
		{"json-reader-error-return", `var v int;err:=json.NewDecoder(helper.Reader{}).Decode(&v);_ = err.Error()`, `type Reader struct{};func(Reader)Read([]byte)(int,error){return 0,Leaf{}};type Leaf struct{};func(Leaf)Error()string{_ = time.Now();return "leaf"}`},
		{"json-distinct-receivers", `_,_=json.Marshal(helper.Holder{Left:helper.Box{F:clean},Right:helper.Box{F:dirty}})`, `type Holder struct{Left,Right Box};type Box struct{F func()};func(b Box)MarshalJSON()([]byte,error){b.F();return []byte("null"),nil}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for name, source := range map[string]string{"go.mod": "module example.invalid/architecture\n\ngo 1.27.1\n", "record/pure.go": `package record;import("example.invalid/architecture/helper";"time";"fmt";"errors";"encoding/json");var _ = errors.New;var _ = fmt.Sprintf;var _=json.Marshal;func clean(){};func dirty(){_ = time.Now()};func Check(){` + test.root + `}`, "helper/helper.go": `package helper;import "time";var _ = time.Now;` + test.helper} {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			p, err := Load(root, "go", Platform{"linux", "amd64"})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, finding := range p.Effects() {
				if finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.Check") && strings.Contains(finding.Detail, "time.Now") {
					found = true
				}
			}
			if !found {
				t.Fatalf("callback clock escaped actual root: %v", p.Effects())
			}
		})
	}
}

func TestPureMemoryFormattingAndJSON(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{"go.mod": "module example.invalid/architecture\n\ngo 1.27.1\n", "record/pure.go": `package record; import("bytes";"fmt";"encoding/json"); type Value struct { N int }; func Encode() {var b bytes.Buffer;_ = json.NewEncoder(&b).Encode(Value{N:1});_ = fmt.Sprintf("%d",1)}`} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	p, err := Load(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if findings := p.Effects(); len(findings) != 0 {
		t.Fatalf("pure memory encoding rejected: %v", findings)
	}
}

func TestPureJSONHelperRetainsConcreteArgument(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{"go.mod": "module example.invalid/architecture\n\ngo 1.27.1\n", "record/pure.go": `package record;import "example.invalid/architecture/helper";type Value struct{N int};func Encode(){helper.Encode(Value{N:1})}`, "helper/helper.go": `package helper;import("bytes";"encoding/json");func Encode(v any){var b bytes.Buffer;_ = json.NewEncoder(&b).Encode(v)}`} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	p, err := Load(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if findings := p.Effects(); len(findings) != 0 {
		t.Fatalf("concrete JSON argument lost: %v", findings)
	}
}

func TestImplicitCallbackPrecedence(t *testing.T) {
	for _, test := range []struct{ name, call, types string }{
		{"formatter-precedence", `_ = fmt.Sprintf("%v",helper.Value{})`, `type Value struct{};func(Value)Format(fmt.State,rune){};func(Value)String()string{_ = time.Now();return "dirty"}`},
		{"gostring-not-v", `_ = fmt.Sprintf("%v",helper.Value{})`, `type Value struct{};func(Value)String()string{return "pure"};func(Value)GoString()string{_ = time.Now();return "dirty"}`},
		{"marshal-precedence", `_,_ = json.Marshal(helper.Value{Inner:helper.Inner{}})`, `type Value struct{Inner Inner};func(Value)MarshalJSON()([]byte,error){return []byte("null"),nil};type Inner struct{};func(Inner)MarshalJSON()([]byte,error){_ = time.Now();return []byte("null"),nil}`},
		{"recursive-owned-copy", `_ = helper.Copy(&helper.Node{})`, `type Node struct{Next *Node};func Copy(n *Node)*Node{if n==nil{return nil};result:=*n;result.Next=Copy(n.Next);return &result}`},
		{"recursive-returned-closure", `helper.Wrap(1,func(){})()`, `func Wrap(n int,f func())func(){if n==0{return f};return Wrap(n-1,func(){f()})}`},
		{"known-pure-closure-factory", `makeCall:=func(f func())func(){return func(){f()}};helper.Call(makeCall(func(){}))`, `func Call(f func()){f()}`},
		{"unwrap-does-not-call-multi", `_ = errors.Unwrap(helper.Value{})`, `type Value struct{};func(Value)Error()string{return "value"};func(Value)Unwrap()[]error{_ = time.Now();return nil}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for name, source := range map[string]string{"go.mod": "module example.invalid/architecture\n\ngo 1.27.1\n", "record/pure.go": `package record;import("example.invalid/architecture/helper";"fmt";"encoding/json";"errors");var _=fmt.Sprintf;var _=json.Marshal;var _=errors.Unwrap;func Check(){` + test.call + `}`, "helper/helper.go": `package helper;import("time";"fmt");var _=time.Now;var _ fmt.State;` + test.types} {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			program, err := Load(root, "go", Platform{"linux", "amd64"})
			if err != nil {
				t.Fatal(err)
			}
			if findings := program.Effects(); len(findings) != 0 {
				t.Fatalf("unused callback incorrectly invoked: %v", findings)
			}
		})
	}
}

func TestPureCanonicalJSONConcreteArgument(t *testing.T) {
	source, err := os.ReadFile("../../canonicaljson/canonical.go")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for name, content := range map[string]string{"go.mod": "module example.invalid/architecture\n\ngo 1.27.1\n", "record/pure.go": `package record;import "example.invalid/architecture/helper";type Value struct{N int};func Encode(){_,_ = helper.CanonicalJSON(Value{N:1})}`, "helper/helper.go": strings.Replace(string(source), "package canonicaljson", "package helper", 1)} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	p, err := Load(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if findings := p.Effects(); len(findings) != 0 {
		t.Fatalf("concrete recursive canonical JSON argument lost: %v", findings)
	}
}
