package architecture

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type errorProvenanceFixture struct {
	name, code, helper, external, callback string
	calls, count                           int
	unresolved                             bool
}

func TestErrorProvenanceUnwrap(t *testing.T) {
	leaf := `type Leaf []int;func(Leaf)Error()string{return "leaf"};func(Leaf)Is(error)bool{Calls++;_=time.Now();return false}`
	outer := `type Outer struct{};func(Outer)Error()string{return "outer"};func(Outer)Unwrap()error{return Leaf{1}};`
	for _, test := range []errorProvenanceFixture{
		{name: "single-named-slice", code: `_ = errors.Is(helper.Outer{},errors.New("target"))`, helper: outer + leaf, calls: 1, callback: "canonicaljson.Leaf.Is"},
		{name: "joined-named-slice", code: `_ = errors.Is(errors.Join(helper.Outer{}),errors.New("target"))`, helper: outer + leaf, calls: 1, callback: "canonicaljson.Leaf.Is"},
		{name: "wrapped-named-slice", code: `_ = errors.Is(fmt.Errorf("%w",helper.Outer{}),errors.New("target"))`, helper: outer + leaf, calls: 1, callback: "canonicaljson.Leaf.Is"},
		{name: "as-named-slice", code: `var target interface{Wanted()};_ = errors.As(helper.Outer{},&target)`, helper: outer + `type Leaf []int;func(Leaf)Error()string{return "leaf"};func(Leaf)As(any)bool{Calls++;_=time.Now();return false}`, calls: 1, callback: "canonicaljson.Leaf.As"},
		{name: "direct-named-slice", code: `_ = errors.Is(helper.Leaf{1},errors.New("target"))`, helper: leaf, calls: 1, callback: "canonicaljson.Leaf.Is"},
		{name: "true-multi", code: `_ = errors.Is(helper.Outer{},errors.New("target"))`, helper: `type Outer struct{};func(Outer)Error()string{return "outer"};func(Outer)Unwrap()[]error{return []error{Leaf{1}}};` + leaf, calls: 1, callback: "canonicaljson.Leaf.Is"},
		{name: "single-only", code: `_ = errors.Unwrap(helper.Outer{})`, helper: `type Outer struct{};func(Outer)Error()string{return "outer"};func(Outer)Unwrap()[]error{Calls++;_=time.Now();return []error{Leaf{1}}};` + leaf},
		{name: "clean-slice", code: `_ = errors.Is(helper.Outer{},errors.New("target"))`, helper: outer + `type Leaf []int;func(Leaf)Error()string{return "leaf"};func(Leaf)Is(error)bool{return false}`},
		{name: "nil-child", code: `_ = errors.Is(helper.Outer{},errors.New("target"))`, helper: `type Outer struct{};func(Outer)Error()string{return "outer"};func(Outer)Unwrap()error{return nil};` + leaf},
		{name: "empty-multi", code: `_ = errors.Is(helper.Outer{},errors.New("target"))`, helper: `type Outer struct{};func(Outer)Error()string{return "outer"};func(Outer)Unwrap()[]error{return []error{}};` + leaf},
		{name: "nil-slice-node", code: `_ = errors.Is(helper.Outer{},errors.New("target"))`, helper: strings.Replace(outer, "Leaf{1}", "Leaf(nil)", 1) + leaf, calls: 1, callback: "canonicaljson.Leaf.Is"},
	} {
		t.Run(test.name, func(t *testing.T) { checkErrorProvenance(t, test) })
	}
}

func TestErrorProvenanceConversion(t *testing.T) {
	leaf := `type Leaf int;func(Leaf)Error()string{Calls++;_=time.Now();return "leaf"};func Make(n int)error{return Leaf(n)}`
	for _, test := range []errorProvenanceFixture{
		{name: "typed-source", code: `_ = helper.Make(1).Error()`, helper: leaf, calls: 1, callback: "canonicaljson.Leaf.Error"},
		{name: "supplier", code: `var source helper.Supplier=helper.Source{N:1};_ = source.Get().Error()`, helper: leaf + `;type Supplier interface{Get()error};type Source struct{N int};func(s Source)Get()error{return Leaf(s.N)}`, calls: 1, callback: "canonicaljson.Leaf.Error"},
		{name: "interface-dynamic-type", code: `_ = helper.Make(1).Error()`, helper: strings.Replace(leaf, "return Leaf(n)", "return error(Leaf(n))", 1), calls: 1, callback: "canonicaljson.Leaf.Error"},
		{name: "external-typed-source", code: `_ = helper.Make(1).Error()`, helper: `func Make(n int)error{return dependency.Make(n)}`, external: `type Leaf int;func(Leaf)Error()string{Calls++;_=time.Now();return "leaf"};func Make(n int)error{return Leaf(n)}`, calls: 1, callback: "dependency.Leaf.Error"},
		{name: "clean-return", code: `_ = helper.Make(1).Error()`, helper: strings.Replace(leaf, "Calls++;_=time.Now();", "", 1)},
		{name: "noninvoked-return", code: `_ = helper.Make(1)`, helper: leaf},
		{name: "unrelated-caller", code: `v:=helper.Source(1);helper.Ignore(v);_ = error(v).Error()`, helper: leaf + `;type Source int;func(Source)Error()string{return "clean"};func Ignore(n Source){_=Leaf(n)}`},
		{name: "function-payload", code: `helper.Make(helper.Dirty)()`, helper: `func Dirty(){Calls++;_=time.Now()};type Callback func();func Make(f func())Callback{return Callback(f)}`, calls: 1, callback: "canonicaljson.Dirty"},
		{name: "slice-alias", code: `v:=[]func(){helper.Clean};helper.Set(helper.Callbacks(v));v[0]()`, helper: `func Clean(){};func Dirty(){Calls++;_=time.Now()};type Callbacks []func();func Set(v Callbacks){v[0]=Dirty}`, calls: 1, callback: "canonicaljson.Dirty"},
		{name: "pointer-alias", code: `v:=&helper.Box{F:helper.Clean};helper.Set((*helper.Other)(v));v.F()`, helper: `func Clean(){};func Dirty(){Calls++;_=time.Now()};type Box struct{F func()};type Other Box;func Set(v *Other){v.F=Dirty}`, calls: 1, callback: "canonicaljson.Dirty"},
	} {
		t.Run(test.name, func(t *testing.T) { checkErrorProvenance(t, test) })
	}
}

func TestErrorProvenanceWriter(t *testing.T) {
	leaf := `type Leaf struct{};func(Leaf)Error()string{Calls++;_=time.Now();return "leaf"};type Writer struct{};func(Writer)Write([]byte)(int,error){return 7,Leaf{}}`
	for _, test := range []errorProvenanceFixture{
		{name: "fprint", code: `n,err:=fmt.Fprint(helper.Writer{},"x");helper.Count=n;_ = err.Error()`, helper: leaf, calls: 1, count: 7, callback: "canonicaljson.Leaf.Error"},
		{name: "fprintf", code: `n,err:=fmt.Fprintf(helper.Writer{},"%s","x");helper.Count=n;_ = err.Error()`, helper: leaf, calls: 1, count: 7, callback: "canonicaljson.Leaf.Error"},
		{name: "fprintln", code: `n,err:=fmt.Fprintln(helper.Writer{},"x");helper.Count=n;_ = err.Error()`, helper: leaf, calls: 1, count: 7, callback: "canonicaljson.Leaf.Error"},
		{name: "wrapped-is", code: `_,err:=fmt.Fprint(helper.Writer{},"x");_ = errors.Is(fmt.Errorf("%w",err),errors.New("target"))`, helper: `type Leaf struct{};func(Leaf)Error()string{return "leaf"};func(Leaf)Is(error)bool{Calls++;_=time.Now();return false};type Writer struct{};func(Writer)Write([]byte)(int,error){return 7,Leaf{}}`, calls: 1, callback: "canonicaljson.Leaf.Is"},
		{name: "nil-return", code: `n,err:=fmt.Fprint(helper.Writer{},"x");helper.Count=n;if err!=nil {_ = err.Error()}`, helper: strings.Replace(leaf, "return 7,Leaf{}", "return 7,nil", 1), count: 7},
		{name: "clean-return", code: `_,err:=fmt.Fprint(helper.Writer{},"x");_ = err.Error()`, helper: strings.Replace(leaf, "Calls++;_=time.Now();", "", 1)},
		{name: "noninvoked-return", code: `_,_=fmt.Fprint(helper.Writer{},"x")`, helper: leaf},
		{name: "unknown-return", code: `_,err:=fmt.Fprint(helper.Writer{},"x");if err!=nil {_ = err.Error()}`, helper: `type Writer struct{Err error};func(w Writer)Write([]byte)(int,error){return 7,w.Err}`, unresolved: true},
	} {
		t.Run(test.name, func(t *testing.T) { checkErrorProvenance(t, test) })
	}
}

func checkErrorProvenance(t *testing.T, test errorProvenanceFixture) {
	t.Helper()
	root := t.TempDir()
	module := "module example.invalid/errorprovenance\n\ngo 1.27.1\n"
	helper := `package helper;import "time";var _=time.Now;var Calls,Count int;` + test.helper
	counter := "helper.Calls"
	imports := `"testing";"example.invalid/errorprovenance/internal/canonicaljson"`
	files := map[string]string{}
	if test.external != "" {
		dependency := t.TempDir()
		module += "require example.invalid/dependency v0.0.0\nreplace example.invalid/dependency => " + dependency + "\n"
		helper = `package helper;import "example.invalid/dependency";var Calls,Count int;` + test.helper
		counter = "dependency.Calls"
		imports += `;"example.invalid/dependency"`
		files[filepath.Join(dependency, "go.mod")] = "module example.invalid/dependency\n\ngo 1.27.1\n"
		files[filepath.Join(dependency, "dependency.go")] = `package dependency;import "time";var Calls int;` + test.external
	}
	files[filepath.Join(root, "go.mod")] = module
	files[filepath.Join(root, "record/pure.go")] = `package record;import("example.invalid/errorprovenance/internal/canonicaljson";"errors";"fmt");var _=errors.New;var _=fmt.Sprintf;func Check(){` + test.code + `}`
	files[filepath.Join(root, "internal/canonicaljson/helper.go")] = helper
	files[filepath.Join(root, "record/pure_test.go")] = `package record;import(` + imports + `);func TestBehavior(t *testing.T){Check();if ` + counter + `!=` + strconv.Itoa(test.calls) + `{t.Fatalf("actual callbacks=%d",` + counter + `)};if helper.Count!=` + strconv.Itoa(test.count) + `{t.Fatalf("actual writer count=%d",helper.Count)};t.Logf("actual callbacks=%d writer count=%d",` + counter + `,helper.Count)}`
	for path, source := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command("go", "test", "-count=1", "-tags", "test_dep", "-v", "./record")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("invalid stock-host fixture: %v\n%s", err, output)
	}
	t.Logf("stock-host causal fixture: %s", output)
	for _, platform := range []Platform{{"linux", "amd64"}, {"darwin", "arm64"}} {
		program, err := Load(root, "go", platform)
		if err != nil {
			t.Fatal(err)
		}
		var packages []Package
		for _, metadata := range program.Metadata {
			packages = append(packages, metadata)
		}
		if findings := PackageEdges(program.Module, packages); len(findings) != 0 {
			t.Fatalf("invalid fixture edges: %v", findings)
		}
		findings := program.Effects()
		t.Logf("metadata %s package edges=0 effects=%v", platform, findings)
		if test.calls == 0 && !test.unresolved {
			if len(findings) != 0 {
				t.Errorf("pure callback rejected %s: %v", platform, findings)
			}
			continue
		}
		found := false
		for _, finding := range findings {
			if test.unresolved {
				if finding.Category == "unresolved-effect" && strings.Contains(finding.Detail, "record.Check") && strings.Contains(finding.Detail, "unknown dynamic callback receiver") {
					found = true
				}
			} else if finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.Check") && strings.Contains(finding.Detail, test.callback) && strings.Contains(finding.Detail, "time.Now") {
				found = true
			}
		}
		if !found {
			t.Errorf("returned callback escaped %s (%s): %v", platform, test.callback, findings)
		}
	}
}

func TestErrorProvenanceEmptySliceAlias(t *testing.T) {
	for _, test := range []errorProvenanceFixture{
		{name: "dirty", code: `v:=make([]func(),1);helper.Set(helper.Callbacks(v));v[0]()`, helper: `func Dirty(){Calls++;_=time.Now()};type Callbacks []func();func Set(v Callbacks){v[0]=Dirty}`, calls: 1, callback: "canonicaljson.Dirty"},
		{name: "clean", code: `v:=make([]func(),1);helper.Set(helper.Callbacks(v));v[0]()`, helper: `func Clean(){};type Callbacks []func();func Set(v Callbacks){v[0]=Clean}`},
		{name: "concrete-zero-element", code: `v:=make([]helper.Leaf,1);_ = fmt.Sprint(helper.Leaves(v))`, helper: `type Leaf struct{};func(Leaf)String()string{Calls++;_=time.Now();return "leaf"};type Leaves []Leaf`, calls: 1, callback: "canonicaljson.Leaf.String"},
		{name: "clean-concrete-zero-element", code: `v:=make([]helper.Leaf,1);_ = fmt.Sprint(helper.Leaves(v))`, helper: `type Leaf struct{};func(Leaf)String()string{return "leaf"};type Leaves []Leaf`},
		{name: "nil-interface-elements", code: `v:=make([]error,1);_ = fmt.Sprint(helper.Errors(v))`, helper: `type Errors []error`},
		{name: "unknown-interface-elements", code: `v:=helper.Box{}.Values;_ = fmt.Sprint(helper.Errors(v))`, helper: `type Box struct{Values []error};type Errors []error`, unresolved: true},
	} {
		t.Run(test.name, func(t *testing.T) { checkErrorProvenance(t, test) })
	}
}
