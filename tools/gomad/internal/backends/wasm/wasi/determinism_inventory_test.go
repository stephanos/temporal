package wasi

import (
	"crypto/sha256"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/determinism"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

const wasiModule = "wasi_snapshot_preview1"

// goImportSites derives the imports the Go host accepts from its own tables:
// the WASI operations of transcriptOperations, and the cooperative module and
// import names that Run compares against imported.Module and imported.Name.
func goImportSites(t *testing.T) []string {
	t.Helper()
	var sites []string
	for _, operation := range transcriptOperations {
		if operation.name != "" && operation.name != "unsupported_operation" {
			sites = append(sites, wasiModule+"::"+operation.name)
		}
	}
	modules, names := cooperativeImportLiterals(t)
	for _, module := range modules {
		for _, name := range names[module] {
			sites = append(sites, module+"::"+name)
		}
	}
	return sites
}

// cooperativeImportLiterals reads Run in process.go: every string compared
// with imported.Module is a cooperative module, and the names it accepts are
// the keys of the remaining map (gomad_wasm_v1) or the single name compared
// inside that module's branch.
func cooperativeImportLiterals(t *testing.T) ([]string, map[string][]string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "process.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var run *ast.FuncDecl
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "Run" && function.Recv == nil {
			run = function
		}
	}
	if run == nil {
		t.Fatal("process.go has no Run")
	}
	literal := func(expression ast.Expr) (string, bool) {
		value, ok := expression.(*ast.BasicLit)
		if !ok || value.Kind != token.STRING {
			return "", false
		}
		text, err := strconv.Unquote(value.Value)
		return text, err == nil
	}
	field := func(expression ast.Expr, name string) bool {
		selector, ok := expression.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			return false
		}
		ident, ok := selector.X.(*ast.Ident)
		return ok && ident.Name == "imported"
	}
	compared := func(expression ast.Expr, name string) (string, bool) {
		binary, ok := expression.(*ast.BinaryExpr)
		if !ok || (binary.Op != token.EQL && binary.Op != token.NEQ) {
			return "", false
		}
		if field(binary.X, name) {
			return literal(binary.Y)
		}
		if field(binary.Y, name) {
			return literal(binary.X)
		}
		return "", false
	}
	var modules []string
	names := map[string][]string{}
	var remaining []string
	ast.Inspect(run.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.AssignStmt:
			if len(node.Lhs) == 1 && len(node.Rhs) == 1 {
				if ident, ok := node.Lhs[0].(*ast.Ident); ok && ident.Name == "remaining" {
					if composite, ok := node.Rhs[0].(*ast.CompositeLit); ok {
						for _, element := range composite.Elts {
							if pair, ok := element.(*ast.KeyValueExpr); ok {
								if key, ok := literal(pair.Key); ok {
									remaining = append(remaining, key)
								}
							}
						}
					}
				}
			}
		case *ast.BinaryExpr:
			if module, ok := compared(node, "Module"); ok && !slices.Contains(modules, module) {
				modules = append(modules, module)
			}
		case *ast.IfStmt:
			// if imported.Module == "<module>" { if imported.Name != "<name>" ... }
			module, ok := compared(node.Cond, "Module")
			if !ok || node.Body == nil {
				return true
			}
			ast.Inspect(node.Body, func(inner ast.Node) bool {
				if binary, ok := inner.(*ast.BinaryExpr); ok {
					if name, ok := compared(binary, "Name"); ok {
						names[module] = append(names[module], name)
					}
				}
				return true
			})
		}
		return true
	})
	if len(remaining) == 0 || len(modules) == 0 {
		t.Fatalf("cooperative import tables not found in Run: modules=%v remaining=%v", modules, remaining)
	}
	for _, module := range modules {
		if len(names[module]) == 0 {
			// The module whose names live in the remaining map.
			names[module] = remaining
		}
	}
	return modules, names
}

func TestDeterminismInventoryCoversGoImportTables(t *testing.T) {
	inventory, err := determinism.Load()
	if err != nil {
		t.Fatal(err)
	}
	// The helper's operation table is the full import set (checked in Rust);
	// every import the Go host accepts must be one of its rows.
	sites := goImportSites(t)
	if err := inventory.Covers("import", sites); err != nil {
		t.Fatal(err)
	}
	// Removing a row, or a Go table gaining an operation without a row,
	// fails by name.
	if err := inventory.Without("import:"+wasiModule+"::fd_write").Covers("import", sites); err == nil || !strings.Contains(err.Error(), `"wasi_snapshot_preview1::fd_write" has no inventory row`) {
		t.Fatalf("removed import row not reported: %v", err)
	}
	if err := inventory.Without("import:gomad_wasm_v1::idle").Covers("import", sites); err == nil || !strings.Contains(err.Error(), `"gomad_wasm_v1::idle" has no inventory row`) {
		t.Fatalf("removed cooperative import row not reported: %v", err)
	}
	if err := inventory.Covers("import", append(slices.Clone(sites), wasiModule+"::sock_recv")); err == nil || !strings.Contains(err.Error(), `"wasi_snapshot_preview1::sock_recv" has no inventory row`) {
		t.Fatalf("new import without a row not reported: %v", err)
	}
}

// TestDeterminismInventoryRejectedImportsAreRefused checks that every WASI
// import the inventory marks rejected is refused by the Go namespace as
// unsupported, and that no controlled WASI import is.
func TestDeterminismInventoryRejectedImportsAreRefused(t *testing.T) {
	inventory, err := determinism.Load()
	if err != nil {
		t.Fatal(err)
	}
	inputs := map[string]string{
		"path_readlink": `{"fd":3,"path_base64":"eA==","length":1}`,
		"path_symlink":  `{"old_path_base64":"eA==","fd":3,"path_base64":"eQ=="}`,
		"sock_accept":   `{"fd":3,"flags":0}`,
		"sock_shutdown": `{"fd":3,"how":0}`,
	}
	rejected := 0
	for site, row := range inventory.Sites("import") {
		name, ok := strings.CutPrefix(site, wasiModule+"::")
		if !ok || row.Disposition != "rejected" {
			continue
		}
		rejected++
		input, ok := inputs[name]
		if !ok {
			t.Fatalf("rejected import %s has no refusal probe", name)
		}
		reply := call(t, newTestEnvironment(t, testConfig()), name, input)
		if !reply.Unsupported || reply.Message != "operation outside stock WASI profile: "+name {
			t.Errorf("rejected import %s not refused before reaching the host: %+v", name, reply)
		}
	}
	if rejected != len(inputs) {
		t.Fatalf("inventory rejects %d WASI imports, probes cover %d", rejected, len(inputs))
	}
}

// TestDeterminismInventoryCooperativeImportsRefuseUnlisted checks that Run
// accepts exactly the cooperative imports the inventory lists and refuses an
// unlisted or mistyped one before the helper is launched.
func TestDeterminismInventoryCooperativeImportsRefuseUnlisted(t *testing.T) {
	inventory, err := determinism.Load()
	if err != nil {
		t.Fatal(err)
	}
	var listed []Import
	for site := range inventory.Sites("import") {
		module, name, ok := strings.Cut(site, "::")
		if ok && strings.HasPrefix(module, "gomad_") {
			listed = append(listed, Import{Module: module, Name: name, Params: []string{"i32", "i32"}, Results: []string{"i32"}})
		}
	}
	slices.SortFunc(listed, func(a, b Import) int { return strings.Compare(a.Module+a.Name, b.Module+b.Name) })
	run := func(imports []Import) error {
		config := testConfig()
		config.Profile = CooperativeProfile
		identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
		_, err := Run(t.Context(), Request{
			HelperPath: "/nonexistent/gomad-wasmhost", HelperSHA256: strings.Repeat("0", 64), Engine: PinnedEngine(), Imports: imports,
			InitialMemoryPages: 1, MemoryBytes: 1 << 16, Fuel: 1, Timeout: time.Second, Config: config,
			Runtime: &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, Limit: 8192, ExecutionIdentity: identity}},
		})
		return err
	}
	// The listed set passes the import checks and fails only at the helper.
	if err := run(listed); err == nil || strings.Contains(err.Error(), "import") {
		t.Fatalf("listed cooperative imports refused: %v", err)
	}
	for _, test := range []struct {
		name    string
		imports []Import
	}{
		{"unlisted v1 name", append(slices.Clone(listed), Import{"gomad_wasm_v1", "clock", []string{"i32", "i32"}, []string{"i32"}})},
		{"unlisted v2 name", append(slices.Clone(listed), Import{"gomad_wasm_v2", "datagram", []string{"i32", "i32"}, []string{"i32"}})},
		{"unlisted debug name", append(slices.Clone(listed), Import{"gomad_debug_v1", "peek", []string{"i32", "i32"}, []string{"i32"}})},
		{"mistyped v1", slices.Concat([]Import{{"gomad_wasm_v1", "decision", []string{"i64"}, []string{"i32"}}}, listed)},
		{"missing v1", slices.DeleteFunc(slices.Clone(listed), func(i Import) bool { return i.Name == "idle" })},
	} {
		if err := run(test.imports); err == nil || !strings.Contains(err.Error(), "import") {
			t.Errorf("%s accepted: %v", test.name, err)
		}
	}
	stock := testConfig()
	if _, err := Run(t.Context(), Request{HelperPath: "/nonexistent", Engine: PinnedEngine(), Imports: listed[:1], InitialMemoryPages: 1, MemoryBytes: 1 << 16, Fuel: 1, Timeout: time.Second, Config: stock}); err == nil || !strings.Contains(err.Error(), "cooperative import in stock profile") {
		t.Fatalf("cooperative import admitted in stock profile: %v", err)
	}
}

func TestDeterminismInventoryMapOrderControls(t *testing.T) {
	inventory, err := determinism.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ site, evidence string }{
		{"wasi/cluster.go|ClusterSession.Shutdown|map-range|session.states|1", "TestClusterShutdownControlOrder"},
		{"wasi/cluster.go|ClusterSession.Shutdown|map-range|session.states|1", "TestClusterShutdownLiveNodesEvidenceOrder"},
		{"wasi/cluster.go|ClusterSession.Shutdown|unstable-sort|sort.Slice live|1", "TestClusterShutdownControlOrder"},
		{"wasi/storage_selection.go|ValidateStorageCrashSelection|map-range|request.StorageCrashContext|1", "TestStorageCrashValidationVolumeOrder"},
		{"wasi/storage_selection.go|ValidateStorageCrashSelection|map-range|request.StorageCrashSelection|1", "TestStorageCrashValidationVolumeOrder"},
	} {
		row, exists := inventory.Sites("host-go")[test.site]
		if !exists || row.Disposition != "controlled" || !slices.Contains(row.Evidence, "go:internal/backends/wasm/wasi:"+test.evidence) {
			t.Errorf("%s must be controlled with behavioral evidence %s", test.site, test.evidence)
		}
	}
}
