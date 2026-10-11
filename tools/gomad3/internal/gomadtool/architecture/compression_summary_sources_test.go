package architecture

import (
	"go/build"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type compressionSourceChecker interface {
	checkCompressionSummarySource(*types.Func, token.Pos) bool
}

func compressionSourceCheck(t *testing.T, analysis *effectAnalysis) compressionSourceChecker {
	t.Helper()
	checker, ok := any(analysis).(compressionSourceChecker)
	if !ok {
		t.Fatal("compression source qualification is unavailable")
	}
	return checker
}

func TestCompressionSummarySourceRequiresImportedFunction(t *testing.T) {
	program := &Program{Files: token.NewFileSet(), Metadata: map[string]Package{}, Packages: map[string]*SourcePackage{}, Functions: map[string]*function{}}
	analysis := summarySourceAnalysis(program)
	checker := compressionSourceCheck(t, analysis)
	for _, fn := range []*types.Func{nil, types.NewFunc(token.NoPos, types.NewPackage("compress/gzip", "gzip"), "NewWriter", types.NewSignatureType(nil, nil, nil, nil, nil, false))} {
		if checker.checkCompressionSummarySource(fn, token.NoPos) {
			t.Fatal("compression source accepted without imported native declaration")
		}
	}
	if len(analysis.findings) == 0 {
		t.Fatal("source rejection lost its conservative finding")
	}
}

// Actual selected stock source is imported without a go-list subprocess. These
// fixtures never fabricate native function objects or replace production pins.
func compressionStockFixture(t *testing.T) *Program {
	t.Helper()
	return compressionStockFixturePlatform(t, Platform{"linux", "amd64"})
}

func compressionStockFixturePlatform(t *testing.T, platform Platform) *Program {
	t.Helper()
	return compressionStockFixtureBeforeImport(t, platform, nil)
}

func compressionStockFixtureBeforeImport(t *testing.T, platform Platform, change func(*Program)) *Program {
	t.Helper()
	context := build.Default
	context.GOOS, context.GOARCH = platform.OS, platform.Arch
	context.CgoEnabled = false
	context.BuildTags = []string{"test_dep"}
	program := &Program{Module: "example.invalid/compression", Files: token.NewFileSet(), Metadata: map[string]Package{}, Packages: map[string]*SourcePackage{}, Functions: map[string]*function{}}
	var discover func(string)
	discover = func(path string) {
		if path == "unsafe" {
			return
		}
		if _, found := program.Metadata[path]; found {
			return
		}
		selected, err := context.Import(path, "", 0)
		if err != nil {
			t.Fatalf("select stock %s: %v", path, err)
		}
		if !selected.Goroot || len(selected.CgoFiles) != 0 {
			t.Fatalf("nonstock or cgo source fixture: %s", path)
		}
		metadata := Package{ImportPath: path, Dir: selected.Dir, Standard: true, GoFiles: selected.GoFiles, Imports: selected.Imports}
		// Keep the RED fixture compile-stable before the approved metadata seam
		// exists. Once present, preserve actual selection rather than inventing it.
		for name, files := range map[string][]string{"SFiles": selected.SFiles, "HFiles": selected.HFiles} {
			field := reflect.ValueOf(&metadata).Elem().FieldByName(name)
			if field.IsValid() {
				field.Set(reflect.ValueOf(append([]string(nil), files...)))
			}
		}
		program.Metadata[path] = metadata
		for _, dependency := range selected.Imports {
			discover(dependency)
		}
	}
	discover("compress/gzip")
	if change != nil {
		change(program)
	}
	if _, err := program.Import("compress/gzip"); err != nil {
		t.Fatal(err)
	}
	return program
}

func compressionSourceAssertRejected(t *testing.T, program *Program, fn *types.Func) {
	t.Helper()
	analysis := summarySourceAnalysis(program)
	if compressionSourceCheck(t, analysis).checkCompressionSummarySource(fn, fn.Pos()) || len(analysis.findings) == 0 {
		t.Fatal("changed compression source binding was accepted without a conservative finding")
	}
}

func TestCompressionSummarySourceRejectsSelectedMetadataMutation(t *testing.T) {
	compressionSourceCheck(t, summarySourceAnalysis(&Program{}))
	program := compressionStockFixture(t)
	fn := stockSummaryFunction(t, summarySourceAnalysis(program), "compress/gzip", "", "NewWriter")
	for _, path := range []string{"compress/gzip", "compress/flate", "hash/crc32"} {
		for _, fieldName := range []string{"GoFiles", "CompiledGoFiles", "SFiles", "HFiles"} {
			for _, mutation := range []string{"missing", "extra", "mixed"} {
				t.Run(path+"/"+fieldName+"/"+mutation, func(t *testing.T) {
					original := program.Metadata[path]
					metadata := original
					field := reflect.ValueOf(&metadata).Elem().FieldByName(fieldName)
					if !field.IsValid() {
						t.Fatalf("approved selected-source metadata seam %s is absent", fieldName)
					}
					files := append([]string(nil), field.Interface().([]string)...)
					if mutation == "missing" {
						if len(files) == 0 {
							return // The exact empty manifest is tested by extra/mixed.
						}
						files = files[1:]
					} else {
						files = append(files, "foreign."+map[string]string{"GoFiles": "go", "CompiledGoFiles": "go", "SFiles": "s", "HFiles": "h"}[fieldName])
						if mutation == "mixed" && path == "hash/crc32" && fieldName != "HFiles" {
							files[len(files)-1] = "crc32_arm64." + map[string]string{"GoFiles": "go", "CompiledGoFiles": "go", "SFiles": "s"}[fieldName]
						}
					}
					field.Set(reflect.ValueOf(files))
					program.Metadata[path] = metadata
					defer func() { program.Metadata[path] = original }()
					compressionSourceAssertRejected(t, program, fn)
				})
			}
		}
	}
}

func TestCompressionSummarySourceRejectsImportTypeAndDeclarationMutation(t *testing.T) {
	compressionSourceCheck(t, summarySourceAnalysis(&Program{}))
	program := compressionStockFixture(t)
	fn := stockSummaryFunction(t, summarySourceAnalysis(program), "compress/gzip", "", "NewWriter")
	for _, path := range []string{"compress/gzip", "compress/flate", "hash/crc32", "sync", "sync/atomic", "internal/cpu", "math/bits", "internal/byteorder", "io"} {
		t.Run(path+"/import", func(t *testing.T) {
			original := program.Metadata[path]
			metadata := original
			metadata.ImportMap = map[string]string{"unsafe": "foreign.invalid/unsafe"}
			metadata.Imports = append(append([]string(nil), metadata.Imports...), "foreign.invalid/dependency")
			program.Metadata[path] = metadata
			defer func() { program.Metadata[path] = original }()
			compressionSourceAssertRejected(t, program, fn)
		})
		t.Run(path+"/type-package", func(t *testing.T) {
			pkg := program.Packages[path]
			original := pkg.Types
			pkg.Types = types.NewPackage(path, original.Name())
			defer func() { pkg.Types = original }()
			compressionSourceAssertRejected(t, program, fn)
		})
	}
	key := functionID(fn)
	original := program.Functions[key]
	copy := *original
	declaration := *copy.declaration
	copy.declaration = &declaration
	program.Functions[key] = &copy
	defer func() { program.Functions[key] = original }()
	compressionSourceAssertRejected(t, program, fn)
}

// Copy exact stock bytes BEFORE import; mutation tests must never write the
// installed toolchain. Copy all .go files so existing whole-directory memory
// pins retain their original algorithm, including inactive/test source files.
func compressionCopiedStockFixture(t *testing.T) (*Program, map[string]string) {
	t.Helper()
	root := t.TempDir()
	filenames := map[string]string{}
	program := compressionStockFixtureBeforeImport(t, Platform{"linux", "amd64"}, func(program *Program) {
		for _, path := range []string{"compress/gzip", "compress/flate", "hash/crc32", "sync", "sync/atomic", "internal/cpu", "math/bits", "internal/byteorder", "io", "encoding/binary", "errors", "slices", "time"} {
			metadata := program.Metadata[path]
			directory := filepath.Join(root, "src", filepath.FromSlash(path))
			if err := os.MkdirAll(directory, 0700); err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(metadata.Dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), ".s") && !strings.HasSuffix(entry.Name(), ".h")) {
					continue
				}
				data, err := os.ReadFile(filepath.Join(metadata.Dir, entry.Name()))
				if err != nil {
					t.Fatal(err)
				}
				filename := filepath.Join(directory, entry.Name())
				if err := os.WriteFile(filename, data, 0600); err != nil {
					t.Fatal(err)
				}
				filenames[path+"/"+entry.Name()] = filename
			}
			metadata.Dir = directory
			program.Metadata[path] = metadata
		}
		for _, relative := range []string{"src/runtime/textflag.h", "pkg/include/textflag.h"} {
			data, err := os.ReadFile(filepath.Join(build.Default.GOROOT, filepath.FromSlash(relative)))
			if err != nil {
				t.Fatal(err)
			}
			filename := filepath.Join(root, filepath.FromSlash(relative))
			if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filename, data, 0600); err != nil {
				t.Fatal(err)
			}
			filenames[relative] = filename
		}
	})
	return program, filenames
}

func TestCompressionSummarySourceRejectsTransitiveSourceAndAuxiliaryChanges(t *testing.T) {
	compressionSourceCheck(t, summarySourceAnalysis(&Program{}))
	program, filenames := compressionCopiedStockFixture(t)
	fn := stockSummaryFunction(t, summarySourceAnalysis(program), "compress/gzip", "", "NewWriter")
	baseline := summarySourceAnalysis(program)
	if !compressionSourceCheck(t, baseline).checkCompressionSummarySource(fn, fn.Pos()) || len(baseline.findings) != 0 {
		t.Fatalf("unchanged exact stock source copy was not qualified: %v", baseline.findings)
	}
	for _, relative := range []string{"compress/gzip/gzip.go", "compress/flate/huffman_code.go", "compress/flate/huffman_bit_writer.go", "hash/crc32/crc32_amd64.go", "hash/crc32/crc32_amd64.s", "hash/crc32/crc32_generic.go", "sync/once.go", "sync/oncefunc.go", "sync/atomic/type.go", "internal/cpu/cpu.go", "internal/cpu/cpu_x86.go", "internal/cpu/cpu_x86_other.go", "math/bits/bits.go", "internal/byteorder/byteorder.go", "io/io.go", "io/multi.go", "io/pipe.go", "src/runtime/textflag.h", "pkg/include/textflag.h"} {
		t.Run(relative, func(t *testing.T) {
			filename := filenames[relative]
			original, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.WriteFile(filename, original, 0600); err != nil {
					t.Fatal(err)
				}
			}()
			if err := os.WriteFile(filename, append(append([]byte(nil), original...), []byte("\n// changed witness\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			compressionSourceAssertRejected(t, program, fn)
			// A negative must not populate a successful source cache.
			analysis := summarySourceAnalysis(program)
			checker := compressionSourceCheck(t, analysis)
			if checker.checkCompressionSummarySource(fn, fn.Pos()) || checker.checkCompressionSummarySource(fn, fn.Pos()) {
				t.Fatal("changed source rejection was cached as success")
			}
		})
	}
}

func TestCompressionSummarySourceExpectedPinChangesInvalidateCachedQualification(t *testing.T) {
	compressionSourceCheck(t, summarySourceAnalysis(&Program{}))
	program := compressionStockFixture(t)
	fn := stockSummaryFunction(t, summarySourceAnalysis(program), "compress/gzip", "", "NewWriter")
	for _, path := range compressionMemoryPackages {
		t.Run(path, func(t *testing.T) {
			analysis := summarySourceAnalysis(program)
			checker := compressionSourceCheck(t, analysis)
			if !checker.checkCompressionSummarySource(fn, fn.Pos()) {
				t.Fatal("initial stock profile was rejected")
			}
			original := memorySummarySources[path]
			memorySummarySources[path] = strings.Repeat("0", 64)
			defer func() { memorySummarySources[path] = original }()
			if checker.checkCompressionSummarySource(fn, fn.Pos()) {
				t.Fatal("cached qualification accepted a changed expected memory source pin")
			}
			compressionSourceAssertRejected(t, program, fn)
		})
	}
	analysis := summarySourceAnalysis(program)
	checker := compressionSourceCheck(t, analysis)
	if !checker.checkCompressionSummarySource(fn, fn.Pos()) {
		t.Fatal("initial stock CPU profile was rejected")
	}
	original := startupSources["internal/cpu"]
	startupSources["internal/cpu"] = strings.Repeat("0", 64)
	defer func() { startupSources["internal/cpu"] = original }()
	if checker.checkCompressionSummarySource(fn, fn.Pos()) {
		t.Fatal("cached qualification accepted a changed CPU source witness pin")
	}
	compressionSourceAssertRejected(t, program, fn)
}

func TestCompressionSummarySourceMissingAuxiliaryFileRejects(t *testing.T) {
	compressionSourceCheck(t, summarySourceAnalysis(&Program{}))
	program, filenames := compressionCopiedStockFixture(t)
	fn := stockSummaryFunction(t, summarySourceAnalysis(program), "compress/gzip", "", "NewWriter")
	for _, relative := range []string{"hash/crc32/crc32_amd64.s", "src/runtime/textflag.h", "pkg/include/textflag.h"} {
		t.Run(relative, func(t *testing.T) {
			filename := filenames[relative]
			original, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filename); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.WriteFile(filename, original, 0600); err != nil {
					t.Fatal(err)
				}
			}()
			compressionSourceAssertRejected(t, program, fn)
		})
	}
}

func TestCompressionSummarySourceSupportsOnlyReviewedTargetProfiles(t *testing.T) {
	compressionSourceCheck(t, summarySourceAnalysis(&Program{}))
	for _, platform := range []Platform{{"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}} {
		t.Run(platform.OS+"/"+platform.Arch, func(t *testing.T) {
			program := compressionStockFixturePlatform(t, platform)
			analysis := summarySourceAnalysis(program)
			fn := stockSummaryFunction(t, analysis, "compress/gzip", "", "NewWriter")
			if !compressionSourceCheck(t, analysis).checkCompressionSummarySource(fn, fn.Pos()) || len(analysis.findings) != 0 {
				t.Fatalf("reviewed selected source profile was rejected: %v", analysis.findings)
			}
		})
	}
}

func TestCompressionSummarySourceAcceptsOnlyNativeWriterEntrypoints(t *testing.T) {
	compressionSourceCheck(t, summarySourceAnalysis(&Program{}))
	program := compressionStockFixture(t)
	for _, entry := range []struct{ receiver, name string }{{"", "NewWriter"}, {"*Writer", "Write"}, {"*Writer", "Flush"}, {"*Writer", "Close"}, {"*Writer", "Reset"}} {
		analysis := summarySourceAnalysis(program)
		fn := stockSummaryFunction(t, analysis, "compress/gzip", entry.receiver, entry.name)
		if !compressionSourceCheck(t, analysis).checkCompressionSummarySource(fn, fn.Pos()) || len(analysis.findings) != 0 {
			t.Fatalf("native writer declaration %s rejected: %v", functionID(fn), analysis.findings)
		}
	}
	for _, entry := range []struct{ path, receiver, name string }{{"compress/gzip", "", "NewWriterLevel"}, {"compress/gzip", "", "NewReader"}, {"compress/gzip", "*Reader", "Read"}, {"compress/flate", "", "NewWriter"}} {
		analysis := summarySourceAnalysis(program)
		fn := stockSummaryFunction(t, analysis, entry.path, entry.receiver, entry.name)
		if compressionSourceCheck(t, analysis).checkCompressionSummarySource(fn, fn.Pos()) || len(analysis.findings) == 0 {
			t.Fatalf("unapproved entrypoint %s acquired source qualification", functionID(fn))
		}
	}
}

func TestCompressionSummarySourceRejectsForgedObjects(t *testing.T) {
	compressionSourceCheck(t, summarySourceAnalysis(&Program{}))
	program := compressionStockFixture(t)
	analysis := summarySourceAnalysis(program)
	fn := stockSummaryFunction(t, analysis, "compress/gzip", "", "NewWriter")
	for _, pkg := range []*types.Package{fn.Pkg(), types.NewPackage("compress/gzip", "gzip")} {
		forged := types.NewFunc(fn.Pos(), pkg, fn.Name(), fn.Type().(*types.Signature))
		if compressionSourceCheck(t, analysis).checkCompressionSummarySource(forged, fn.Pos()) {
			t.Fatal("forged native declaration acquired source qualification")
		}
	}
	if len(analysis.findings) == 0 {
		t.Fatal("forged source rejection lost conservative finding")
	}
}
