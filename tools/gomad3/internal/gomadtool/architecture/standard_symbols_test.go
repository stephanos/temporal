package architecture

import (
	"go/token"
	"go/types"
	"testing"
)

func TestStandardHostSymbolClassification(t *testing.T) {
	for _, group := range []struct {
		path, receiver string
		names          []string
	}{
		{"net", "", []string{"Dial", "DialTimeout", "Listen", "ListenPacket", "ListenIP", "ListenTCP", "ListenUDP", "ListenUnix", "ListenUnixgram", "DialIP", "DialTCP", "DialUDP", "DialUnix", "LookupAddr", "LookupCNAME", "LookupHost", "LookupIP", "LookupMX", "LookupNS", "LookupSRV", "LookupTXT"}},
		{"net", "*Dialer", []string{"Dial", "DialContext"}},
		{"net", "*ListenConfig", []string{"Listen", "ListenPacket"}},
		{"os", "", []string{"Open", "OpenFile", "Create", "ReadFile", "WriteFile", "Stat", "Lstat", "ReadDir", "Mkdir", "MkdirAll", "Remove", "RemoveAll", "Rename", "Getenv", "LookupEnv", "Environ", "Setenv", "Unsetenv", "Getwd", "Chdir", "Exit", "FindProcess", "StartProcess", "Executable", "UserHomeDir", "TempDir", "Pipe"}},
		{"time", "", []string{"Now", "Parse", "Sleep", "After", "AfterFunc", "Tick", "NewTimer", "NewTicker", "LoadLocation", "LoadLocationFromTZData"}},
		{"runtime", "", []string{"nanotime", "walltime", "fastrand", "rand", "startTheWorld", "newproc"}},
	} {
		for _, name := range group.names {
			id := group.path + "."
			if group.receiver != "" {
				id += group.receiver + "."
			}
			id += name
			t.Run(id, func(t *testing.T) {
				fn := reflectSummaryFunction(group.path, group.receiver, name)
				for _, standard := range []bool{true, false} {
					a := &effectAnalysis{
						program:  &Program{Files: token.NewFileSet(), Metadata: map[string]Package{group.path: {Standard: standard}}},
						findings: map[string]Finding{}, root: "record.Root", chain: []string{"helper.Call"},
					}
					_, handled := a.standardUnchecked(fn, nil, nil, token.NoPos)
					wantFindings := 0
					if standard {
						wantFindings = 1
					}
					if handled != standard || len(a.findings) != wantFindings {
						t.Fatalf("standard=%v: handled=%v findings=%v", standard, handled, a.findings)
					}
					for _, finding := range a.findings {
						if finding.Category != "host-effect" || finding.Detail != "record.Root -> helper.Call -> "+id {
							t.Fatalf("host classification or provenance changed: %v", finding)
						}
					}
				}
			})
		}
	}
}

func TestStandardUnknownSymbolLookupAllocations(t *testing.T) {
	for _, path := range []string{"net", "fmt", "encoding/json", "sort", "slices", "reflect", "example.invalid/unmodeled"} {
		t.Run(path, func(t *testing.T) {
			pkg := types.NewPackage(path, "unmodeled")
			fn := types.NewFunc(token.NoPos, pkg, "Unmodeled", types.NewSignatureType(nil, nil, nil, nil, nil, false))
			a := &effectAnalysis{
				program:  &Program{Files: token.NewFileSet(), Metadata: map[string]Package{path: {Standard: true}}},
				findings: map[string]Finding{},
			}
			receiver := &abstractValue{builtin: true}
			var result *abstractValue
			var handled bool
			allocations := testing.AllocsPerRun(100, func() {
				result, handled = a.standardUnchecked(fn, receiver, nil, token.NoPos)
			})
			if result != nil || handled || len(a.findings) != 0 {
				t.Fatalf("unmodeled symbol was admitted: result=%v handled=%v findings=%v", result, handled, a.findings)
			}
			if allocations > 2 {
				t.Fatalf("unmatched lookup rebuilt symbol sets: %.0f allocations, want at most 2 for function ID", allocations)
			}
		})
	}
}
