package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCallbackContainerMutations(t *testing.T) {
	for _, test := range []struct {
		name, code string
		effect     bool
	}{
		{"copy-slice", `dst:=[]func(){helper.Clean};copy(dst,[]func(){helper.Dirty});dst[0]()`, true},
		{"copy-alias", `dst:=[]func(){helper.Clean};alias:=dst;copy(dst,[]func(){helper.Dirty});alias[0]()`, true},
		{"insert", `dst:=[]func(){helper.Clean};dst=slices.Insert(dst,0,helper.Dirty);dst[0]()`, true},
		{"insert-ellipsis", `dst:=[]func(){helper.Clean};dst=slices.Insert(dst,0,[]func(){helper.Dirty}...);dst[0]()`, true},
		{"insert-alias", `dst:=make([]func(),1,2);dst[0]=helper.Clean;alias:=dst;dst=slices.Insert(dst,0,helper.Dirty);alias[0]()`, true},
		{"clone-assignment", `dst:=[]func(){helper.Clean};dst=slices.Clone([]func(){helper.Dirty});dst[0]()`, true},
		{"pointer-store", `f:=helper.Clean;p:=&f;*p=helper.Dirty;f()`, true},
		{"pointer-helper", `f:=helper.Clean;helper.Set(&f);f()`, true},
		{"range-function", `for range helper.Iter {}`, true},
		{"range-yield-callback", `for f:=range helper.Callbacks {f()}`, true},
		{"slice-pointer-marshaler", `_,_=json.Marshal([]helper.Value{{}})`, true},
		{"addressable-array-marshaler", `_,_=json.Marshal(&[1]helper.Value{{}})`, true},
		{"addressable-field-marshaler", `_,_=json.Marshal(&struct{V helper.Value}{})`, true},
		{"map-nonaddressable", `_,_=json.Marshal(map[string]helper.Value{"a":{}})`, false},
		{"array-nonaddressable", `_,_=json.Marshal([1]helper.Value{{}})`, false},
		{"field-nonaddressable", `_,_=json.Marshal(struct{V helper.Value}{})`, false},
		{"pure-copy", `dst:=[]func(){helper.Clean};copy(dst,[]func(){helper.Clean});dst[0]()`, false},
		{"pure-insert", `dst:=[]func(){helper.Clean};dst=slices.Insert(dst,0,helper.Clean);dst[0]()`, false},
		{"pure-clone", `dst:=[]func(){helper.Clean};dst=slices.Clone([]func(){helper.Clean});dst[0]()`, false},
		{"pure-pointer", `f:=helper.Clean;helper.SetClean(&f);f()`, false},
		{"pure-iterator", `for n:=range helper.PureIter {_=n}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for name, source := range map[string]string{
				"go.mod":                           "module example.invalid/mutation\n\ngo 1.27.1\n",
				"record/pure.go":                   `package record;import("example.invalid/mutation/internal/canonicaljson";"slices";"encoding/json");var _=slices.Clone[[]int];var _=json.Marshal;func Check(){` + test.code + `}`,
				"internal/canonicaljson/helper.go": `package helper;import "time";func Clean(){};func Dirty(){_=time.Now()};func Set(p *func()){*p=Dirty};func SetClean(p *func()){*p=Clean};func Iter(yield func(int)bool){_=time.Now();yield(1)};func Callbacks(yield func(func())bool){yield(Dirty)};func PureIter(yield func(int)bool){yield(1)};type Value struct{};func(*Value)MarshalJSON()([]byte,error){_=time.Now();return []byte("null"),nil}`,
			} {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
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
						t.Fatalf("pure %s rejected: %v", platform, findings)
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
					t.Fatalf("callback clock escaped %s: %v", platform, findings)
				}
			}
		})
	}
}
