package translate

import (
	"cmp"
	_ "embed"
	"flag"
	"go/token"
	"log"
	"os"
	"path"
	"runtime/pprof"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/modfile"
	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/go/packages"

	"github.com/temporalio/gomad/internal/gomadtool"
	"github.com/temporalio/gomad/internal/translate/cache"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

var PublicExportHacks = map[string][]string{
	"encoding/binary":                  {"littleEndian"},
	"internal/poll":                    {"errNetClosing"},
	"github.com/golang/protobuf/proto": {"enumsByName"},
}

// XXX: replace the package instead
var replacements = map[packageSelector]packageSelector{
	{Pkg: "runtime", Selector: "SetFinalizer"}:               {Pkg: gomadruntimePackage, Selector: "SetFinalizer"},
	{Pkg: "runtime", Selector: "GOOS"}:                       {Pkg: gomadruntimePackage, Selector: "GOOS"},
	{Pkg: "runtime", Selector: "Gosched"}:                    {Pkg: gomadruntimePackage, Selector: "Yield"},
	{Pkg: "internal/runtime/maps", Selector: "Use64BitHash"}: {Pkg: gomadruntimePackage, Selector: "Use64BitHash"},
}

const gomadModPath = gomadtool.Module

const (
	gomadruntimePackage = gomadModPath + "/gomadruntime"
	stdlibHooksPackage  = gomadModPath + "/internal/stdlib/hooks"
	reflectPackage      = gomadModPath + "/internal/reflect"
	simulationPackage   = gomadModPath + "/internal/simulation"
	testingPackage      = gomadModPath + "/internal/testing"
)

var TranslatedRuntimePackages = []string{
	stdlibHooksPackage,
	reflectPackage,
	simulationPackage,
	testingPackage,
}

const (
	// loadDepGraph fairly quickly loads the dependency graph
	loadDepGraph = packages.NeedName | packages.NeedFiles | packages.NeedModule | packages.NeedImports | packages.NeedDeps
	// loadSyntaxAndTypes relatively slowly loads detailed types and syntax information
	loadSyntaxAndTypes = packages.NeedSyntax | packages.NeedName | packages.NeedTypes |
		packages.NeedTypesInfo | packages.NeedFiles | packages.NeedImports
)

func loadPackages(patterns []string, b gomadtool.BuildConfig, mode packages.LoadMode, tests bool) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode:  mode,
		Tests: tests,
		Fset:  token.NewFileSet(),
	}

	// apply build config to packages config
	// TODO: build tags plan (rename files, force GOOS and GOARCH)
	// TODO: if we support varying GOARCH here, we should select the arch-specific hooks at runtime...
	env := append(os.Environ(), "GOOS="+b.GOOS, "GOARCH="+b.GOARCH, "CGO_ENABLED=0")
	cfg.BuildFlags = []string{"-tags", b.PackageTags()}
	cfg.Env = env

	// load packages
	packages, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}
	if len(packages) == 0 {
		log.Println(err)
		log.Fatal("failed to load packages... missing go.mod deps?")
	}

	// check for errors
	for _, pkg := range packages {
		if pkg.Errors != nil {
			// TODO: ignore errors in tests outside of what we care about?
			log.Println("errors for ", pkg.PkgPath)
			for _, err := range pkg.Errors {
				log.Println(err)
			}
			os.Exit(1)
		}
	}

	return packages, nil
}

// reloadUncachedPackages loads the types and AST for the given uncached package
// paths. It takes special care to load tests only where appropriate, because
// broken tests in dependencies are not uncommon.
//
// loadPackages has a binary flag for loading tests, which if enabled loads
// tests for the explicitly listed packages. It does not load tests for
// dependencies.
//
// When we reload packages here we explicitly list all packages we want to load,
// including dependencies whose tests we do not care about. To not load tests
// for those we make to calls to loadPackages, one with and one without tests.
func reloadUncachedPackages(listedPkgs []*packages.Package, uncachedPackages map[string]struct{}, cfg gomadtool.BuildConfig) map[string]*packages.Package {
	// determine packages we want to load tests for based on
	// the original command line arguments
	listedPkgPaths := make(map[string]struct{})
	for _, pkg := range listedPkgs {
		if fromGomad := slices.Contains(TranslatedRuntimePackages, pkg.PkgPath); fromGomad {
			continue
		}
		listedPkgPaths[pkg.PkgPath] = struct{}{}
	}

	// reloadByTests is packages to reload, with tests enabled or not
	reloadByTests := make(map[bool][]string)
	for path := range uncachedPackages {
		_, ok := listedPkgPaths[path]
		reloadByTests[ok] = append(reloadByTests[ok], path)
	}

	// reload packages in parallel
	var mu sync.Mutex
	reloadedByTests := make(map[bool][]*packages.Package)
	var g errgroup.Group
	for test, pkgs := range reloadByTests {
		g.Go(func() error {
			reloaded, err := loadPackages(pkgs, cfg, loadSyntaxAndTypes, test)
			if err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			reloadedByTests[test] = reloaded
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}

	// merge results
	pkgsWithTypesAndAst := make(map[string]*packages.Package)
	for _, pkgs := range reloadedByTests {
		for _, pkg := range pkgs {
			pkgsWithTypesAndAst[pkg.ID] = pkg
		}
	}

	return pkgsWithTypesAndAst
}

type packageKind string

const (
	PackageKindTestBinary = "testbinary"
	PackageKindBase       = "base"
	PackageKindForTest    = "fortest"
	PackageKindTests      = "tests"
)

func classifyPackage(pkg *packages.Package) (packageKind, string) {
	switch {
	case pkg.Name == "main" && strings.HasSuffix(pkg.PkgPath, ".test"):
		// XXX: is this correct?
		return PackageKindTestBinary, strings.TrimSuffix(pkg.PkgPath, ".test")

	case strings.HasSuffix(pkg.Name, "_test"):
		return PackageKindTests, strings.TrimSuffix(pkg.PkgPath, "_test")

	case strings.HasSuffix(pkg.ID, ".test]"):
		return PackageKindForTest, pkg.PkgPath

	case !strings.HasSuffix(pkg.Name, "_test") && pkg.PkgPath == pkg.ID:
		return PackageKindBase, pkg.PkgPath

	default:
		log.Fatal("weird package", pkg.Name, pkg.PkgPath, pkg.ID)
		panic("unreachable")
	}
}

func collectImports(roots []*packages.Package, skip map[string]bool) []*packages.Package {
	seen := make(map[*packages.Package]bool)
	var order []*packages.Package
	var visit func(pkg *packages.Package)
	visit = func(pkg *packages.Package) {
		if skip[pkg.PkgPath] || seen[pkg] {
			return
		}
		seen[pkg] = true
		order = append(order, pkg)
		for _, dep := range pkg.Imports {
			visit(dep)
		}
	}
	for _, root := range roots {
		visit(root)
	}
	slices.SortFunc(order, func(a, b *packages.Package) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return order
}

func writeGoModFile(modDir string, modFile *modfile.File, writer *outputWriter) {
	isGomad := modFile.Module.Mod.Path == gomadModPath
	// take the existing go.mod and make it work for a sub-directory containing
	// a module translated
	if err := modFile.AddModuleStmt("translated"); err != nil {
		log.Fatal(err)
	}

	if isGomad {
		// special case running translate in the gomad module
		if err := modFile.AddRequire(gomadModPath, "v0.0.0"); err != nil {
			log.Fatal(err)
		}
		if err := modFile.AddReplace(gomadModPath, "", "../../../", ""); err != nil {
			log.Fatal(err)
		}
	} else {
		// adjust relative paths for the new module's location
		for _, replace := range modFile.Replace {
			// > Third, filesystem paths found in "replace" directives are
			// represented by a path with an empty version.
			if replace.New.Version == "" && !path.IsAbs(replace.New.Path) {
				// The output module is located three directories deeper than the old module.
				newPath := path.Join("../../../", replace.New.Path)
				if err := modFile.AddReplace(replace.Old.Path, replace.Old.Version, newPath, ""); err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	bytes, err := modFile.Format()
	if err != nil {
		log.Fatal(err)
	}
	if err := writer.stage("go.mod", bytes); err != nil {
		log.Fatal(err)
	}
	goSumBytes, err := os.ReadFile(path.Join(modDir, "go.sum"))
	if err != nil {
		log.Fatal(err)
	}
	if err := writer.stage("go.sum", goSumBytes); err != nil {
		log.Fatal(err)
	}
}

type TranslateInput struct {
	Packages []string
	Cfg      gomadtool.BuildConfig
}

func Translate(input *TranslateInput) (*gomadtool.TranslateOutput, error) {
	// XXX: dedup?
	modDir, err := gomadtool.FindGoModDir()
	if err != nil {
		log.Fatal(err)
	}
	rootOutputDir := path.Join(modDir, gomadtool.OutputDirectory, "translated", input.Cfg.AsDirname())

	cacheDir := path.Join(modDir, gomadtool.OutputDirectory)
	if override := os.Getenv("GOMADCACHE"); override != "" {
		cacheDir = override
	}

	cachePath := path.Join(cacheDir, "cache.sqlite3")
	if err := os.MkdirAll(path.Dir(cachePath), 0o755); err != nil {
		log.Fatal(err)
	}
	db, err := cache.NewDB(cachePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	cache := cache.NewCache(db)
	defer cache.Clean()
	// XXX: cap size at something reasonable? 100x working set?

	if err := os.MkdirAll(rootOutputDir, 0o755); err != nil {
		log.Fatal(err)
	}
	return translatePackages(cache, input.Packages, rootOutputDir, input.Cfg)
}

func buildReplacePackagesAndPackageNames(convertPkgs, allPkgs []*packages.Package) (replacedPkgs map[string]string, packageNames map[string]string) {
	packageNames = make(map[string]string)
	packageNames["golang.org/x/sys/cpu"] = "cpu" // XXX: here because we replace the vendored one with the common one
	for _, pkg := range allPkgs {
		packageNames[pkg.PkgPath] = pkg.Name
	}

	replacedPkgs = make(map[string]string)
	for _, pkg := range convertPkgs {
		inputPackage := pkg.PkgPath
		outputPackage := "translated/" + gomadtool.ReplaceSpecialPackages(pkg.PkgPath)
		replacedPkgs[inputPackage] = outputPackage
		packageNames[outputPackage] = packageNames[inputPackage]
	}

	// override reflect and testing
	replacedPkgs["reflect"] = replacedPkgs[reflectPackage]
	replacedPkgs["internal/reflectlite"] = replacedPkgs[reflectPackage]
	replacedPkgs["testing"] = replacedPkgs[testingPackage]

	// handle the linkname in the os package
	replacedPkgs["net"] = "translated/" + gomadtool.ReplaceSpecialPackages("net")

	// not replaced, but need to know for rewrites
	replacedPkgs[gomadruntimePackage] = gomadruntimePackage
	return
}

func checkGomadDep(modFile *modfile.File) {
	// work in the gomad module
	if isGomad := modFile.Module.Mod.Path == gomadModPath; isGomad {
		return
	}
	for _, req := range modFile.Require {
		// work if there is an explicit dependency
		if req.Mod.Path == gomadModPath {
			return
		}
	}
	// complain otherwise
	log.Fatalf("current module does not depend on %v, try running init", gomadModPath)
}

func checkSingleModule(modPath string, pkgs []*packages.Package) {
	for _, pkg := range pkgs {
		// allow packages from the current module
		if pkg.Module.GoMod == modPath {
			continue
		}
		// allow packages explicitly added by translate
		_, path := classifyPackage(pkg)
		if slices.Contains(TranslatedRuntimePackages, path) {
			continue
		}
		// complain about others
		log.Fatal("packages from outside module modules ", pkg.PkgPath, " ", modPath)
	}
}

func translatePackages(cache *cache.Cache, listPatterns []string, rootOutputDir string, cfg gomadtool.BuildConfig) (*gomadtool.TranslateOutput, error) {
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	listPatterns = append(listPatterns, TranslatedRuntimePackages...)

	listedPkgs, err := loadPackages(listPatterns, cfg, loadDepGraph, true)
	if err != nil {
		log.Fatal(err)
	}
	if len(listedPkgs) == 0 {
		log.Fatal("no packages")
	}

	modPath, modFile, err := gomadtool.FindGoMod()
	if err != nil {
		log.Fatal(err)
	}
	modDir := path.Dir(modPath)

	checkGomadDep(modFile)
	checkSingleModule(modPath, listedPkgs)

	allPkgs := collectImports(listedPkgs, nil)
	convertPkgs := collectImports(listedPkgs, activeStdlibPolicy.skippedPackages)

	packageGraph := newDepGraph()
	basePkgs := make(map[string]*packages.Package)
	pkgById := make(map[string]*packages.Package)
	for _, pkg := range convertPkgs {
		packageGraph.addNode(pkg.ID)
		pkgById[pkg.ID] = pkg

		if kind, path := classifyPackage(pkg); kind == PackageKindBase {
			basePkgs[path] = pkg
		}
	}

	for _, pkg := range convertPkgs {
		for _, dep := range pkg.Imports {
			if _, ok := packageGraph.nodes[dep.ID]; !ok {
				// XXX?
				continue
			}
			packageGraph.addDep(pkg.ID, dep.ID)
		}

		// XXX: add a package from the "for test" to the "main" package
		kind, path := classifyPackage(pkg)
		if kind == PackageKindForTest || kind == PackageKindTests {
			if basePkgs[path] != nil {
				packageGraph.addDep(pkg.ID, basePkgs[path].ID)
			} else {
				log.Println("huh", kind, pkg.PkgPath, path)
				log.Fatal(":(")
			}
		}
	}

	translateToolHash := computeTranslateToolHash(cfg)

	numWorkers := 32

	packageHashes := make(map[string]Hash)

	buildInParallel(packageGraph, numWorkers, packageHashes, func(pkgId string, importHashes map[string]Hash) Hash {
		return computePackageHash(translateToolHash, pkgById[pkgId], importHashes)
	})

	allResults := make(map[string]*TranslatePackageResult)
	cacheHits := make(map[Hash]bool)

	uncachedPackages := make(map[string]struct{})
	for pkgId, hash := range packageHashes {
		pkg := pkgById[pkgId]

		res, err := cacheGet(cache, hash)
		if err != nil {
			log.Fatal(err)
		}
		if res != nil {
			allResults[pkgId] = res
			cacheHits[hash] = true // record cache hits so we don't update the cache for them
		} else {
			uncachedPackages[strings.TrimSuffix(pkg.PkgPath, "_test")] = struct{}{}
		}
	}

	pkgsWithTypesAndAst := reloadUncachedPackages(listedPkgs, uncachedPackages, cfg)

	replacedPkgs, packageNames := buildReplacePackagesAndPackageNames(convertPkgs, allPkgs)

	buildInParallel(packageGraph, numWorkers, allResults, func(pkgId string, localResults map[string]*TranslatePackageResult) *TranslatePackageResult {
		return translatePackage(&translatePackageArgs{
			cfg:                cfg,
			pkg:                pkgById[pkgId],
			replacedPkgs:       replacedPkgs,
			hooksPackage:       stdlibHooksPackage,
			packageNames:       packageNames,
			importResults:      localResults,
			pkgWithTypesAndAst: pkgsWithTypesAndAst[pkgId],
		})
	})

	for pkgId, res := range allResults {
		hash := packageHashes[pkgId]
		if !cacheHits[hash] {
			if err := cachePut(cache, hash, res); err != nil {
				log.Fatal(err)
			}
		}
	}

	writer := newOutputWriter()
	for _, res := range allResults {
		if err := writer.merge(res.TranslatedFiles); err != nil {
			log.Fatal(err)
		}
	}

	writeGoModFile(modDir, modFile, writer)

	if err := writer.writeFiles(rootOutputDir); err != nil {
		log.Fatal(err)
	}
	if err := writer.maybeDeleteGeneratedFiles(rootOutputDir); err != nil {
		log.Fatal(err)
	}

	var out []string
	for _, pkg := range listedPkgs {
		kind, _ := classifyPackage(pkg)
		if kind != PackageKindBase {
			continue
		}
		if fromGomad := slices.Contains(TranslatedRuntimePackages, pkg.PkgPath); fromGomad {
			continue
		}
		out = append(out, replacedPkgs[pkg.PkgPath])
	}

	deps := make(map[string]map[string]time.Time)
	modTimeCache := make(map[string]time.Time)
	for _, pkg := range listedPkgs {
		kind, path := classifyPackage(pkg)
		if kind != PackageKindTestBinary {
			continue
		}

		// TODO: also add deps from the mandatory linked packages?
		files := findAllDepFiles(pkg, modDir)
		times, err := loadModTimes(files, modTimeCache)
		if err != nil {
			log.Fatal(err)
		}
		deps[replacedPkgs[path]] = times
	}

	return &gomadtool.TranslateOutput{
		RootOutputDir: rootOutputDir,
		Packages:      out,
		Deps:          deps,
	}, nil
}

func findAllDepFiles(pkg *packages.Package, root string) []string {
	seen := make(map[*packages.Package]struct{})
	var walk func(*packages.Package)
	var files []string
	walk = func(pkg *packages.Package) {
		if _, ok := seen[pkg]; ok {
			return
		}
		seen[pkg] = struct{}{}
		for _, dep := range pkg.Imports {
			walk(dep)
		}
		for _, file := range pkg.GoFiles {
			if strings.HasPrefix(file, root) {
				files = append(files, file)
			}
		}
	}
	walk(pkg)
	slices.Sort(files)
	return files
}

func loadModTimes(files []string, cache map[string]time.Time) (map[string]time.Time, error) {
	result := make(map[string]time.Time)
	for _, file := range files {
		if t, ok := cache[file]; ok {
			result[file] = t
			continue
		}

		info, err := os.Stat(file)
		if err != nil {
			return nil, err
		}
		t := info.ModTime()
		cache[file] = t
		result[file] = t
	}
	return result, nil
}

// Go version?
// - GOTOOLCHAIN aware?

// Test perf?
// - only run once for all testdata?

// Cmd
// - nicer flags for gomad test

// TODO: hash more singletons?
// - env vars? anything influencing go packages?
// - use go packages export file hash?

// next:
// - verify determinism
// - deal with logging output

// log progress?
// log.Println(results.pkgID, hex.EncodeToString(results.inputHash[:]))
