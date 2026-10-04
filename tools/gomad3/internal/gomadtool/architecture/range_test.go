package architecture

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRangeAssignmentSlots(t *testing.T) {
	for _, test := range []struct {
		name, code string
		effect     bool
	}{
		{"existing", `f:=helper.Clean;for f=range helper.Callbacks {};f()`, true},
		{"field", `v:=struct{F func()}{helper.Clean};for v.F=range helper.Callbacks {};v.F()`, true},
		{"index", `v:=[]func(){helper.Clean};for v[0]=range helper.Callbacks {};v[0]()`, true},
		{"dereference", `f:=helper.Clean;p:=&f;for *p=range helper.Callbacks {};f()`, true},
		{"captured-existing", `f:=helper.Clean;g:=func(){f()};for f=range helper.Callbacks {};g()`, true},
		{"declared-capture", `g:=helper.Clean;for f:=range helper.Callbacks {g=func(){f()}};g()`, true},
		{"two-values", `f:=helper.Clean;for _,f=range helper.Pair {};f()`, true},
		{"slice-existing-capture", `f:=helper.Clean;g:=func(){f()};for _,f=range []func(){helper.Dirty} {};g()`, true},
		{"slice-field", `v:=struct{F func()}{helper.Clean};for _,v.F=range []func(){helper.Dirty} {};v.F()`, true},
		{"map-index", `v:=[]func(){helper.Clean};for _,v[0]=range map[int]func(){0:helper.Dirty} {};v[0]()`, true},
		{"clean-existing", `f:=helper.Clean;for f=range helper.PureCallbacks {};f()`, false},
		{"clean-field", `v:=struct{F func()}{helper.Clean};for v.F=range helper.PureCallbacks {};v.F()`, false},
		{"clean-capture", `g:=helper.Clean;for f:=range helper.PureCallbacks {g=func(){f()}};g()`, false},
		{"noninvoked", `f:=helper.Clean;for f=range helper.None {helper.Dirty()};f()`, false},
		{"paren-existing", `f:=helper.Clean;for ((f))=range helper.Callbacks {};f()`, true},
		{"paren-field", `v:=struct{F func()}{helper.Clean};for ((v.F))=range helper.Callbacks {};v.F()`, true},
		{"paren-index", `v:=[]func(){helper.Clean};for ((v[0]))=range helper.Callbacks {};v[0]()`, true},
		{"paren-dereference", `f:=helper.Clean;p:=&f;for ((*p))=range helper.Callbacks {};f()`, true},
		{"effect-index", `v:=[]func(){helper.Clean};for v[helper.Index()]=range helper.PureCallbacks {};v[0]()`, true},
		{"ordinary-paren", `f:=helper.Clean;((f))=helper.Dirty;f()`, true},
		{"ordinary-index", `v:=[]func(){helper.Clean};v[helper.Index()]=helper.Clean;v[0]()`, true},
		{"slice-max", `v:=[]func(){helper.Clean};v=v[:1:helper.Index()+1];v[0]()`, true},
		{"map-key", `v:=map[int]func(){helper.Index():helper.Clean};v[0]()`, true},
		{"clean-paren", `f:=helper.Clean;for ((f))=range helper.PureCallbacks {};f()`, false},
		{"clean-index", `v:=[]func(){helper.Clean};for v[helper.PureIndex()]=range helper.PureCallbacks {};v[0]()`, false},
		{"clean-slice-max", `v:=[]func(){helper.Clean};v=v[:1:helper.PureIndex()+1];v[0]()`, false},
		{"clean-map-key", `v:=map[int]func(){helper.PureIndex():helper.Clean};v[0]()`, false},
		{"noninvoked-map-key", `v:=map[int]func(){helper.IgnoreIndex(helper.Dirty):helper.Clean};v[0]()`, false},
		{"noninvoked-slice-max", `v:=[]func(){helper.Clean};v=v[:1:helper.IgnoreIndex(helper.Dirty)+1];v[0]()`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			want := 0
			if test.effect {
				want = 1
			}
			for name, source := range map[string]string{
				"go.mod":                           "module example.invalid/rangeslots\n\ngo 1.27.1\n",
				"record/pure.go":                   `package record;import "example.invalid/rangeslots/internal/canonicaljson";func Check(){` + test.code + `}`,
				"internal/canonicaljson/helper.go": `package helper;import "time";var Calls int;func Clean(){};func Dirty(){Calls++;_=time.Now()};func Index()int{Dirty();return 0};func PureIndex()int{return 0};func IgnoreIndex(fn func())int{return 0};func Callbacks(yield func(func())bool){yield(Dirty)};func PureCallbacks(yield func(func())bool){yield(Clean)};func None(yield func(func())bool){};func Pair(yield func(int,func())bool){yield(1,Dirty)}`,
				"record/pure_test.go":              `package record;import("testing";"example.invalid/rangeslots/internal/canonicaljson");func TestBehavior(t *testing.T){Check();if helper.Calls!=` + strconv.Itoa(want) + `{t.Fatalf("actual effects=%d",helper.Calls)};t.Logf("actual effects=%d",helper.Calls)}`,
			} {
				path := filepath.Join(root, name)
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
				t.Fatalf("invalid native fixture: %v\n%s", err, output)
			}
			t.Logf("stock native causal fixture: %s", output)
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
				if !test.effect {
					if len(findings) != 0 {
						t.Fatalf("pure iteration rejected: %v", findings)
					}
					continue
				}
				found := false
				for _, finding := range findings {
					if finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.Check") && strings.Contains(finding.Detail, "time.Now") {
						found = true
					}
				}
				if !found {
					t.Errorf("yielded effect escaped %s: %v", platform, findings)
				}
			}
		})
	}
}
