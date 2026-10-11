package architecture

import (
	"crypto/sha256"
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// This profile qualifies source, never callback purity. Only the separate
// constructor-owned gzip writer summary may consume its successful result.
var compressionSummarySourcePins = map[string]map[string]string{
	"compress/gzip": {
		"gzip.go":   "15564f213c583a22e79da0611c173c28ead54fa6651ad9bc7797938048e77ba0",
		"gunzip.go": "5886344b7fd61358d430872ab735653b16a314c87dab1f0bde4d8d9db0381ec4",
	},
	"compress/flate": {
		"deflate.go":            "50d6c83cdff9ccd5cce90ce1dc9d827de0648dd01a78032d3ad4a8932b642d04",
		"deflatefast.go":        "ca01c6426266a5e2b5154cf1123a357be37f016247f14409c97a8d744bdc6518",
		"dict_decoder.go":       "a0862076d1c99a910b8b46f81f0d55c4bc6f09ade8307a2cedfd3f69ee28310d",
		"huffman_bit_writer.go": "ce22b308bb43d26661d79dd68b15dff0526212b3f2ff4ecaffb12f7bbed45fcc",
		"huffman_code.go":       "1098a9d935c86863d49b8fc9e81481a6ac3bca41686c4350be120ef10c68fb85",
		"inflate.go":            "28b241ba6b66cc14ac93cc17a49756cd78daaade60afb47d7cd6aed2489dc5cb",
		"level1.go":             "aeb270b53a189c2f1a971cb891d7c98b3e03d8f9edd981169c36e1a1b49eb290",
		"level2.go":             "3a89117710b3832a80e89ff1ae098d65425cebefd09702f4dd530ec7957f2d99",
		"level3.go":             "e10dec76189c92563eb8cac21659187ff3fdebc58c1eca8f4b79890eea87c935",
		"level4.go":             "4994a4be40a0912cdd374ec3b301fa551d5d6768b0e26c7baf53d42eee78dbb4",
		"level5.go":             "745661dad9d18ef228de51d0aa92f89c08884be053acb7fca6fce06231a6eb09",
		"level6.go":             "0bbd86f2ec1b856d293c5e2cf2f09bd7e82cbb4d99f5aa7a2959b719b330b26c",
		"load_store.go":         "5ecf768f2ee78d075a5161e7bf3ab946e0ae4c7dc8142e99ac66a1c0254007e9",
		"token.go":              "19c2dabd2e911839dc74c9f63131eb092cdc053c4bddf6654310d6b5da70e3be",
		"regmask_amd64.go":      "470ff9906e97252aafc87c6fb7c63c1b9dd59518220503ab6dfe851406d57330",
		"regmask_other.go":      "7196ac23e798b05acb411b5c611e873894e91757bcac56ed00cf14b1a97fadc0",
	},
	"hash/crc32": {
		"crc32.go":         "85f9b1484d31d449fda4c70b6df4e8c78455b5384d06fecd5f7a359638de2f76",
		"crc32_generic.go": "ab955ded37b50660ad89f448fa2d6129e73cee97e4755770c0c695d666fb065d",
		"gen.go":           "b6f5dd452079fe27e1789a41dee81b24b2b933a655fcaa4ca530be8bafb2fe63",
		"crc32_amd64.go":   "548dc603730f2b153b1568bacb155ac57568e8b474d5be371971e4f785ab2865",
		"crc32_arm64.go":   "e3c97e14322a0dd605f21797bf1aff2cbfac580ed41b109cabab909d35b7f320",
	},
	"internal/cpu": {
		"cpu.go":       "9af4310772f84d92f2a98e9f16e6553fdbefc4ce6286e0f508121e6c8819faf7",
		"cpu_x86.go":   "395ce0d3004df63df376585519bc1b46303b206415c1915ab80b7bead155e26a",
		"cpu_arm64.go": "5b156522fb27d47992d4ab013779c6d24c87e002a1a83550042cbd1d1d641538",
	},
	"internal/byteorder": {"byteorder.go": "c31dade373ba9a6970fdec0943cdf95b2a8ae1f0e057fdd581e50165aca26fbb"},
	"math/bits": {
		"bits.go":        "694e0114bda396589e8c993a3d7880a0b461da23ee64c272929a3a5b69c2acf2",
		"bits_errors.go": "1341283973609b16798176456337e3a0cbff9bfb32f79673a45197223471d7c6",
		"bits_tables.go": "c4cb4a05dab6d395d9d51720893a42f0a9d1cdf8a0c2383d87268e49acb73f7f",
	},
	"io": {
		"io.go":    "3945d328a9072e4f5970b7d4bc0eb834eec29fd22889a1ed9f3fb76ad41c3274",
		"multi.go": "acb61a128d4e49dcc8b5dcd268ee1bd61008fd97af0e459f839166e3ae51384e",
		"pipe.go":  "75ef56f505592fcffb188bf7660c6ea6abede3f2701b500d6946fa519c37e233",
	},
}

var compressionMemoryPackages = []string{"encoding/binary", "errors", "slices", "time", "sync", "sync/atomic"}

var compressionCommonGoFiles = map[string][]string{
	"compress/gzip":      {"gunzip.go", "gzip.go"},
	"compress/flate":     {"deflate.go", "deflatefast.go", "dict_decoder.go", "huffman_bit_writer.go", "huffman_code.go", "inflate.go", "level1.go", "level2.go", "level3.go", "level4.go", "level5.go", "level6.go", "load_store.go", "token.go"},
	"hash/crc32":         {"crc32.go", "crc32_generic.go", "gen.go"},
	"encoding/binary":    {"binary.go", "native_endian_little.go", "varint.go"},
	"errors":             {"errors.go", "join.go", "wrap.go"},
	"slices":             {"iter.go", "slices.go", "sort.go", "zsortanyfunc.go", "zsortordered.go"},
	"time":               {"format.go", "format_rfc3339.go", "sleep.go", "sys_unix.go", "tick.go", "time.go", "zoneinfo.go", "zoneinfo_goroot.go", "zoneinfo_read.go", "zoneinfo_unix.go"},
	"sync":               {"cond.go", "map.go", "mutex.go", "once.go", "oncefunc.go", "pool.go", "poolqueue.go", "runtime.go", "runtime2.go", "rwmutex.go", "waitgroup.go"},
	"sync/atomic":        {"doc.go", "doc_64.go", "type.go", "value.go"},
	"internal/byteorder": {"byteorder.go"},
	"math/bits":          {"bits.go", "bits_errors.go", "bits_tables.go"},
	"io":                 {"io.go", "multi.go", "pipe.go"},
}

const compressionTextflagSHA256 = "a9d761d03aecdcc117691b6a55eacd43eb7d47d9ad9ca7510b8669ac403b380e"

var compressionAssemblyPins = map[string]string{
	"crc32_amd64.s": "a72c2ee64b5035a506864ce6ecba6823ee7260862353970c05a96cadda053861",
	"crc32_arm64.s": "aedb6699f5f65086add1e0b86c8824fcf5f29cfb08c390c19b55441b01bac1b5",
}

type compressionPackageSourceSnapshot struct {
	native      *jsonPackageSourceSnapshot
	auxiliary   map[string]string
	definitions map[*types.Var]types.Type
	memoryPin   string
	startupPin  string
}

type compressionSourceQualification struct {
	profile  string
	packages map[string]*compressionPackageSourceSnapshot
}

func compressionSourcePackage(path string) bool {
	_, pinned := compressionSummarySourcePins[path]
	return pinned || slices.Contains(compressionMemoryPackages, path)
}

func bindCompressionPackageSource(pkg *SourcePackage, digests, auxiliary map[string]string) *compressionPackageSourceSnapshot {
	native := bindJSONPackageSource(pkg, digests)
	native.metadata.SFiles = slices.Clone(pkg.Metadata.SFiles)
	native.metadata.HFiles = slices.Clone(pkg.Metadata.HFiles)
	definitions := map[*types.Var]types.Type{}
	for _, object := range pkg.Info.Defs {
		if object, ok := object.(*types.Var); ok {
			definitions[object] = object.Type()
		}
	}
	return &compressionPackageSourceSnapshot{native: native, auxiliary: maps.Clone(auxiliary), definitions: definitions, memoryPin: memorySummarySources[pkg.Metadata.ImportPath], startupPin: startupSources[pkg.Metadata.ImportPath]}
}

// Auxiliary bytes are not Go ASTs. The generated go_asm.h offset is qualified by
// native Go type/constant identity below, never by an invented disk input.
func compressionAuxiliaryPaths(metadata Package) map[string]string {
	paths := map[string]string{}
	if metadata.ImportPath != "hash/crc32" {
		return paths
	}
	for _, name := range append(slices.Clone(metadata.SFiles), metadata.HFiles...) {
		paths[name] = filepath.Join(metadata.Dir, name)
	}
	root := filepath.Clean(filepath.Join(metadata.Dir, "..", "..", ".."))
	for _, name := range []string{"src/runtime/textflag.h", "pkg/include/textflag.h"} {
		paths[name] = filepath.Join(root, filepath.FromSlash(name))
	}
	return paths
}

func captureCompressionAuxiliary(metadata Package) (map[string]string, error) {
	digests := map[string]string{}
	if metadata.ImportPath == "internal/cpu" {
		digest, err := compressionWholeGoDigest(metadata)
		if err != nil {
			return nil, err
		}
		digests["startup-go-source"] = digest
	}
	for name, filename := range compressionAuxiliaryPaths(metadata) {
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		digests[name] = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	return digests, nil
}

// Reuse the existing initialization owner's exact aggregate algorithm/pin as
// a source witness only. This does not classify CPU initialization as pure.
func compressionWholeGoDigest(metadata Package) (string, error) {
	entries, err := os.ReadDir(metadata.Dir)
	if err != nil {
		return "", err
	}
	digest := sha256.New()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(metadata.Dir, entry.Name()))
		if err != nil {
			return "", err
		}
		_, _ = digest.Write(fmt.Appendf(nil, "%x  %s\n", sha256.Sum256(data), entry.Name()))
	}
	return fmt.Sprintf("%x", digest.Sum(nil)), nil
}

func compressionSameNames(actual, expected []string) bool {
	actual, expected = slices.Clone(actual), slices.Clone(expected)
	slices.Sort(actual)
	slices.Sort(expected)
	return slices.Equal(actual, expected)
}

func compressionCompiledSelection(metadata Package) bool {
	if len(metadata.CompiledGoFiles) == 0 {
		return true
	}
	names := make([]string, 0, len(metadata.CompiledGoFiles))
	for _, filename := range metadata.CompiledGoFiles {
		name := filepath.Base(filename)
		if filepath.IsAbs(filename) && filepath.Clean(filename) != filepath.Clean(filepath.Join(metadata.Dir, name)) {
			return false
		}
		if !filepath.IsAbs(filename) && filename != name {
			return false
		}
		names = append(names, name)
	}
	return compressionSameNames(names, metadata.GoFiles)
}

func compressionSelectedProfile(sources map[string]*compressionPackageSourceSnapshot) (map[string][]string, string, bool) {
	crc := sources["hash/crc32"]
	if crc == nil {
		return nil, "", false
	}
	assembly := crc.native.metadata.SFiles
	arch := ""
	if slices.Equal(assembly, []string{"crc32_amd64.s"}) {
		arch = "amd64"
	} else if slices.Equal(assembly, []string{"crc32_arm64.s"}) {
		arch = "arm64"
	} else {
		return nil, "", false
	}
	expected := map[string][]string{}
	for path, names := range compressionCommonGoFiles {
		expected[path] = slices.Clone(names)
	}
	expected["hash/crc32"] = append(expected["hash/crc32"], "crc32_"+arch+".go")
	if arch == "amd64" {
		expected["compress/flate"] = append(expected["compress/flate"], "regmask_amd64.go")
		expected["internal/cpu"] = []string{"cpu.go", "cpu_x86.go", "cpu_x86_other.go", "datacache_x86.go"}
	} else {
		expected["compress/flate"] = append(expected["compress/flate"], "regmask_other.go")
		cpu := sources["internal/cpu"]
		if cpu == nil {
			return nil, "", false
		}
		expected["internal/cpu"] = []string{"cpu.go", "cpu_arm64.go", "cpu_no_name.go", "datacache_unsupported.go"}
		if slices.Contains(cpu.native.metadata.GoFiles, "cpu_arm64_darwin.go") {
			expected["internal/cpu"] = append(expected["internal/cpu"], "cpu_arm64_darwin.go", "cpu_darwin.go")
		} else {
			expected["internal/cpu"] = append(expected["internal/cpu"], "cpu_arm64_hwcap.go", "cpu_arm64_linux.go")
		}
	}
	return expected, arch, true
}

func compressionCPUOffsets(pkg *SourcePackage, arch string) bool {
	constantObject, ok := pkg.Types.Scope().Lookup("CacheLinePadSize").(*types.Const)
	if !ok {
		return false
	}
	padding, ok := constant.Int64Val(constantObject.Val())
	if !ok || (arch == "arm64" && padding != 128) || (arch == "amd64" && padding != 64) {
		return false
	}
	if arch != "amd64" {
		return true
	}
	object, ok := pkg.Types.Scope().Lookup("X86").(*types.Var)
	if !ok {
		return false
	}
	structure, ok := object.Type().Underlying().(*types.Struct)
	if !ok {
		return false
	}
	fields := make([]*types.Var, structure.NumFields())
	for i := range fields {
		fields[i] = structure.Field(i)
	}
	offsets := types.SizesFor("gc", "amd64").Offsetsof(fields)
	for i, field := range fields {
		if field.Name() == "HasAVX512VPCLMULQDQ" {
			return field.Type() == types.Typ[types.Bool] && offsets[i] == 82
		}
	}
	return false
}

func (a *effectAnalysis) checkCompressionSummarySource(fn *types.Func, pos token.Pos) bool {
	reject := func(detail string) bool {
		a.report("unresolved-effect", pos, "compression source qualification: "+detail)
		return false
	}
	if fn == nil || fn.Pkg() == nil {
		return reject("requires an imported native function")
	}
	fn = fn.Origin()
	if !slices.Contains([]string{"compress/gzip.NewWriter", "compress/gzip.*Writer.Write", "compress/gzip.*Writer.Flush", "compress/gzip.*Writer.Close", "compress/gzip.*Writer.Reset"}, functionID(fn)) {
		return reject("unapproved writer entrypoint")
	}
	declaration := a.program.Functions[functionID(fn)]
	pkg := a.program.Packages["compress/gzip"]
	if pkg == nil || pkg.Types != fn.Pkg() || declaration == nil || declaration.pkg != pkg || declaration.object != fn {
		return reject("native function identity changed")
	}
	expected, arch, valid := compressionSelectedProfile(a.program.compressionSources)
	if !valid || len(a.program.compressionSources) != len(expected) {
		return reject("unknown selected source profile")
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte("gomad3-constructor-owned-gzip-source-v1\x00"))
	paths := slices.Sorted(maps.Keys(expected))
	for _, path := range paths {
		snapshot := a.program.compressionSources[path]
		pkg := a.program.Packages[path]
		if snapshot == nil || pkg == nil {
			return reject("missing native source " + path)
		}
		if slices.Contains(compressionMemoryPackages, path) {
			if snapshot.memoryPin == "" || memorySummarySources[path] != snapshot.memoryPin {
				return reject("expected memory source contract changed: " + path)
			}
			_, _ = digest.Write(fmt.Appendf(nil, "memory\x00%s\x00%s\x00", path, snapshot.memoryPin))
		}
		if path == "internal/cpu" {
			if snapshot.startupPin == "" || startupSources[path] != snapshot.startupPin || !maps.Equal(snapshot.auxiliary, map[string]string{"startup-go-source": snapshot.startupPin}) {
				return reject("expected CPU startup source witness changed")
			}
			_, _ = digest.Write(fmt.Appendf(nil, "cpu-source\x00%s\x00", snapshot.startupPin))
		} else if path != "hash/crc32" && len(snapshot.auxiliary) != 0 {
			return reject("unexpected auxiliary source witness: " + path)
		}
		source := snapshot.native
		if pkg != source.pkg || pkg.Types != source.types || pkg.Info != source.info || !slices.Equal(pkg.Files, source.files) || !slices.Equal(pkg.Types.Imports(), source.imports) ||
			!sameJSONSourceMetadata(a.program.Metadata[path], source.metadata) || !sameJSONSourceMetadata(pkg.Metadata, source.metadata) ||
			!slices.Equal(a.program.Metadata[path].SFiles, source.metadata.SFiles) || !slices.Equal(a.program.Metadata[path].HFiles, source.metadata.HFiles) ||
			!slices.Equal(pkg.Metadata.SFiles, source.metadata.SFiles) || !slices.Equal(pkg.Metadata.HFiles, source.metadata.HFiles) ||
			source.metadata.ImportPath != path || len(source.metadata.CgoFiles) != 0 || !compressionCompiledSelection(source.metadata) || !compressionSameNames(source.metadata.GoFiles, expected[path]) || len(source.digests) != len(expected[path]) {
			return reject("parsed source binding changed: " + path)
		}
		if len(source.metadata.HFiles) != 0 {
			return reject("unexpected selected headers: " + path)
		}
		expectedAssembly := []string(nil)
		switch path {
		case "hash/crc32":
			expectedAssembly = []string{"crc32_" + arch + ".s"}
		case "sync/atomic":
			expectedAssembly = []string{"asm.s"} // Existing atomic contract, not new asm purity.
		case "internal/cpu":
			expectedAssembly = []string{"cpu.s", "cpu_" + map[string]string{"amd64": "x86", "arm64": "arm64"}[arch] + ".s"}
		}
		if !compressionSameNames(source.metadata.SFiles, expectedAssembly) {
			return reject("unexpected selected assembly: " + path)
		}
		for _, name := range expected[path] {
			if filepath.Base(name) != name || source.digests[name] == "" {
				return reject("missing parsed bytes: " + path + "/" + name)
			}
			if pin := compressionSummarySourcePins[path][name]; pin != "" && source.digests[name] != pin {
				return reject("computational source changed: " + path + "/" + name)
			}
			_, _ = digest.Write(fmt.Appendf(nil, "%s\x00%s\x00%s\x00", path, name, source.digests[name]))
		}
		for id, bound := range source.functions {
			current := a.program.Functions[id]
			if current == nil || current.pkg != pkg || current.object.Origin() != bound.object || current.declaration != bound.declaration || pkg.Info.Defs[bound.declaration.Name] != bound.object {
				return reject("native declaration changed: " + id)
			}
		}
		for object, typ := range snapshot.definitions {
			if object.Type() != typ {
				return reject("native type changed: " + path)
			}
		}
		for _, imported := range source.imports {
			if imported == types.Unsafe {
				continue
			}
			dependency := a.program.Packages[imported.Path()]
			if dependency == nil || dependency.Types != imported || !dependency.Metadata.Standard || !a.program.Metadata[imported.Path()].Standard {
				return reject("native import identity changed: " + imported.Path())
			}
		}
		for _, file := range source.files {
			for _, imported := range file.Imports {
				name, err := strconv.Unquote(imported.Path.Value)
				if redirect := source.metadata.ImportMap[name]; err != nil || (redirect != "" && redirect != name) {
					return reject("native import redirected: " + path)
				}
			}
		}
	}
	if !compressionCPUOffsets(a.program.Packages["internal/cpu"], arch) {
		return reject("native CPU layout changed")
	}
	crc := a.program.compressionSources["hash/crc32"]
	auxiliaryPins := map[string]string{"crc32_" + arch + ".s": compressionAssemblyPins["crc32_"+arch+".s"], "src/runtime/textflag.h": compressionTextflagSHA256, "pkg/include/textflag.h": compressionTextflagSHA256}
	if !maps.Equal(crc.auxiliary, auxiliaryPins) {
		return reject("native CRC auxiliary bytes changed")
	}
	profile := fmt.Sprintf("%x", digest.Sum(nil))
	if cached := a.checkedCompression; cached != nil && cached.profile == profile && maps.Equal(cached.packages, a.program.compressionSources) {
		return true
	}
	for _, path := range compressionMemoryPackages {
		if !a.checkMemorySource(path, pos) {
			return false
		}
	}
	for _, path := range paths {
		source := a.program.compressionSources[path].native
		for name, bound := range source.digests {
			data, err := os.ReadFile(filepath.Join(source.metadata.Dir, name))
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != bound {
				return reject("current disk differs from parsed bytes: " + path + "/" + name)
			}
		}
	}
	auxiliary, err := captureCompressionAuxiliary(crc.native.metadata)
	if err != nil || !maps.Equal(auxiliary, auxiliaryPins) {
		return reject("current CRC auxiliary bytes changed")
	}
	cpu := a.program.compressionSources["internal/cpu"]
	currentCPU, err := compressionWholeGoDigest(cpu.native.metadata)
	if err != nil || currentCPU != cpu.startupPin {
		return reject("current CPU startup source witness changed")
	}
	a.checkedCompression = &compressionSourceQualification{profile: profile, packages: maps.Clone(a.program.compressionSources)}
	return true
}
