package canonicaljson

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sync"
	"testing"
)

type cacheEmbedded struct{ Value int }

// This is a narrow source/representation contract, not a complete Effects proof
// and not a claim that the previous cache returned incorrect canonical bytes.
func TestStreamFieldCacheOwnerMemoryContract(t *testing.T) {
	cacheType := reflect.TypeOf(&streamStructCache).Elem()
	mutex, hasMutex := cacheType.FieldByName("mutex")
	entries, hasEntries := cacheType.FieldByName("entries")
	if !hasMutex || mutex.Type != reflect.TypeFor[sync.RWMutex]() || !hasEntries ||
		entries.Type.Kind() != reflect.Map || entries.Type.Key() != reflect.TypeFor[reflect.Type]() ||
		entries.Type.Elem() != reflect.TypeFor[streamFields]() {
		t.Fatal("private metadata cache must use owner-local RWMutex and typed reflect.Type-to-streamFields memory, not sync.Map dispatch")
	}
	file, err := parser.ParseFile(token.NewFileSet(), "streaming.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var body *ast.BlockStmt
	for _, declaration := range file.Decls {
		if fn, ok := declaration.(*ast.FuncDecl); ok && fn.Name.Name == "fieldsForStream" {
			body = fn.Body
		}
	}
	if body == nil {
		t.Fatal("actual fieldsForStream source is absent")
	}
	locked := false
	locks := map[string]int{}
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		method, ok := call.Fun.(*ast.SelectorExpr)
		if ok {
			if receiver, ok := method.X.(*ast.SelectorExpr); ok && receiver.Sel.Name == "mutex" {
				if cache, ok := receiver.X.(*ast.Ident); ok && cache.Name == "streamStructCache" {
					locks[method.Sel.Name]++
					switch method.Sel.Name {
					case "RLock", "Lock":
						if locked {
							t.Fatal("nested metadata cache lock region")
						}
						locked = true
					case "RUnlock", "Unlock":
						if !locked {
							t.Fatal("metadata cache unlock outside lock region")
						}
						locked = false
					default:
						t.Fatal("unsupported metadata cache lock operation")
					}
					return true
				}
			}
		}
		if locked {
			t.Fatal("metadata discovery, sorting, or another call occurred inside cache lock")
		}
		return true
	})
	if locked || locks["RLock"] != 1 || locks["RUnlock"] != 1 || locks["Lock"] != 1 || locks["Unlock"] != 1 {
		t.Fatalf("metadata cache must have one lookup and one publication lock region: %v", locks)
	}
}

func TestStreamFieldCacheCharacterizesMetadata(t *testing.T) {
	for _, test := range []struct {
		name   string
		typ    reflect.Type
		legacy bool
	}{
		{"ordinary", reflect.TypeFor[struct{ A string }](), false},
		{"embedded", reflect.TypeFor[struct{ cacheEmbedded }](), true},
		{"collision", reflect.StructOf([]reflect.StructField{
			{Name: "A", Type: reflect.TypeFor[int](), Tag: `json:"same"`},
			{Name: "B", Type: reflect.TypeFor[int](), Tag: `json:"same"`},
		}), true},
		{"invalid-tag", reflect.TypeFor[struct {
			A int `json:"bad\\name"`
		}](), true},
		{"string", reflect.TypeFor[struct {
			A int `json:"a,string"`
		}](), true},
		{"omitzero", reflect.TypeFor[struct {
			A int `json:"a,omitzero"`
		}](), true},
		{"unknown-option", reflect.TypeFor[struct {
			A int `json:"a,future"`
		}](), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			first := fieldsForStream(test.typ)
			if first.legacy != test.legacy {
				t.Fatalf("legacy=%t, want %t", first.legacy, test.legacy)
			}
			if again := fieldsForStream(test.typ); !reflect.DeepEqual(first, again) {
				t.Fatal("cache hit changed metadata")
			}
		})
	}
	typ := reflect.TypeFor[struct {
		Z      int    `json:"z,omitempty"`
		A      string `json:"a"`
		Hidden string `json:"-"`
		hidden string
	}]()
	want := streamFields{fields: []streamField{{name: "a", index: 1}, {name: "z", index: 0, omitEmpty: true}}}
	if got := fieldsForStream(typ); !reflect.DeepEqual(got, want) {
		t.Fatalf("metadata=%#v, want %#v", got, want)
	}
}

func TestStreamFieldCacheConcurrentColdPublication(t *testing.T) {
	// The first invocation in a test process is cold until workers are released.
	// Repeated -count invocations characterize hits, not another cold publication.
	typ := reflect.TypeFor[struct {
		Z int `json:"cache_cold_z,omitempty"`
		A int `json:"cache_cold_a"`
	}]()
	const workers = 24
	start := make(chan struct{})
	results := make(chan streamFields, workers)
	for range workers {
		go func() {
			<-start
			results <- fieldsForStream(typ)
		}()
	}
	close(start)
	first := <-results
	if len(first.fields) != 2 || first.legacy {
		t.Fatalf("unexpected cold metadata: %#v", first)
	}
	for range workers - 1 {
		got := <-results
		if !reflect.DeepEqual(got, first) || &got.fields[0] != &first.fields[0] {
			t.Fatal("cold callers did not return the winning published immutable slice")
		}
	}
}

func TestStreamFieldCacheConcurrentBytesMatchLegacy(t *testing.T) {
	values := []any{
		struct {
			Z string `json:"z,omitempty"`
			A string `json:"a"`
		}{A: "<>&雪😀\u2028\u2029"},
		struct {
			cacheEmbedded
			Z string
		}{cacheEmbedded{7}, "last"},
		struct {
			A int `json:"a,string"`
		}{12},
		struct {
			A int `json:"a,omitzero"`
		}{},
	}
	const workers = 12
	expected := make([][]byte, len(values))
	for index, value := range values {
		want, err := CanonicalJSON(value)
		if err != nil {
			t.Fatal(err)
		}
		expected[index] = want
	}
	errors := make(chan error, workers)
	for worker := range workers {
		go func() {
			for iteration := range 80 {
				index := (worker + iteration) % len(values)
				var output bytes.Buffer
				if err := WriteCanonicalJSON(&output, values[index]); err != nil {
					errors <- err
					return
				}
				if !bytes.Equal(output.Bytes(), expected[index]) {
					errors <- fmt.Errorf("stream=%q legacy=%q", output.Bytes(), expected[index])
					return
				}
			}
			errors <- nil
		}()
	}
	for range workers {
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
	}
}
