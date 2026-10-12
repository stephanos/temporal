# Gomad architecture checks: structural checker plus bounded purity prover

## Goal & Context

Gomad's architecture checks live in one package, `tools/gomad3/internal/gomadtool/architecture`: 6,900 non-test lines plus 7,531 test lines, 14,431 in all. The package is a whole-program effect prover built on `go/types`. It type-checks every standard-library dependency from source on two platforms, resolves function values and interface bindings across the whole program, and carries ownership proofs for canonical writers, `encoding/json` callbacks, `compress/gzip` writers and reflect metadata, each pinned to hashed copies of standard-library source so a Go upgrade invalidates the summary. Of the 36 effect fixtures, 8 describe host effects; the other 28 describe callback and alias resolution.

The property it defends is small: a named set of pure files may not read the host. For wasm that set is `wasi/` minus two transport files, and those files already import nothing impure. For native it is `world`, `record`, `world/mailbox`, two internal packages and three single files. The prover re-proves shared packages along every caller's call path, per platform, when proving them once at their own boundary gives the same guarantee. The wasm purity test lives in gomad3 and never runs when only wasm changes.

This spec splits the checks into two tools of a size that can be read in a sitting. A structural checker, about 500 lines, runs in every module and is the only check the wasm backend needs. A bounded purity checker, at most 2,000 lines, replaces the prover for the native pure set and the shared evidence packages, using heuristics with explicit review marks instead of whole-program resolution. Total budget, tests included, is 4,000 lines. The ownership proofs and the pinned standard-library sources are deleted. [fn-153](fn-153-gomad-retire-canonical-json-the-.md) is already removing the canonical JSON writer those proofs defend. The checker configuration expresses today's layout and then the layout of [fn-164](fn-164-gomad-backend-layout-shared-core-with.md), whose R1 import rule it implements.

## Architecture & Data Models

Two packages replace `internal/gomadtool/architecture`:

```
tools/gomad3/internal/archcheck/        structural checker, no go/types
  config.go     Config: owners, import allowlists, pure sets, review-mark policy
  inventory.go  Discover: module roots, host packages per platform, uncovered sources (kept from today)
  imports.go    package and file import edges against owner and allowlist rules
  syntax.go     per-file AST rules for pure files
  findings.go   Finding{Category, Path, Detail}, stable output ordering
tools/gomad3/internal/puritycheck/      bounded purity checker, go/types from export data
  load.go       go list -export, importer.ForCompiler("gc"), one platform per call
  stdlib.go     allowlist of pure standard-library symbols, keyed by Go minor version
  calls.go      static call classification: stdlib symbol, module function, dynamic
  dynamic.go    dynamic call sites and review marks
  signatures.go PublicSignatures (kept from today's program.go)
```

Both are plain Go with no new module dependencies. `go/importer` with gc export data replaces the source importer, so no standard-library source is parsed and nothing is pinned by hash.

Structural checker rules, all per file or per `go list` edge:

| Rule | Category | Applies to |
| --- | --- | --- |
| Every host package has exactly one owner; owners may import only listed owners | `ownerless`, `owner-edge` | every module |
| Module edges follow the fn-164 direction (shared never imports `backends/*`; backends never import each other) | `module-edge` | every module |
| A pure file imports only packages on the pure allowlist: pure module packages, and a fixed standard-library set | `pure-import` | pure sets |
| A pure file has no `go` statement, no `select` with a `time` channel, no `init` with calls outside the pure set | `pure-syntax` | pure sets |
| `syscall` is imported by a pure file only for `Errno` and `E*` constants | `pure-import` | pure sets |
| Public types of a package reachable from a named external consumer do not expose internal types or aliases | `public-signature` | gomad3 public packages, wasm module |

Purity checker rules, over typed call sites in pure files only:

| Rule | Category |
| --- | --- |
| A call to a standard-library function or method is on the pure symbol allowlist | `host-effect` |
| A call to a module function resolves to a function declared in a pure file | `host-effect` |
| A call through a function value, interface method or method value whose static type is declared outside the pure set is a dynamic call; it needs a review mark on the same line or the line above | `dynamic-call` |
| A method declared in a pure file is checked whether or not anything calls it; this covers `MarshalJSON`, `String`, `Format`, `Write` and `Error` callbacks without callback tracking | (same categories) |
| A review mark that no longer sits on a dynamic call is reported | `stale-review` |

Review mark syntax, one line:

```go
//gomad:dynamic reviewed 2026-10-12 writer is a *bytes.Buffer supplied by compressedGzipPass
```

The checker prints the count and list of review marks per package in its test log so reviewers see the surface. Marks are not a suppression of `host-effect`; only `dynamic-call` can be marked.

Heuristics that replace proofs:

| Today's proof | Replacement | Residual risk and why it is acceptable |
| --- | --- | --- |
| `encoding/json` and `fmt` callback resolution through reflect metadata (json_summary, standard reflect, 2,000 lines) | Import allowlist plus declared-method rule: a value marshaled inside a pure package has a type from a pure package or the standard library, and every method in a pure package is checked directly | A value of interface type `any` holding an impure type injected by a caller; this requires a dynamic entry point, which is the `dynamic-call` rule's domain |
| `compress/gzip` writer ownership (compression_summary, 640 lines) | `gzip.NewWriter`, `gzip.NewReader` are pure symbols; the writer or reader they wrap is a dynamic value only when its static type is `io.Writer` or `io.Reader` from outside the package, which the `dynamic-call` rule marks | None beyond dynamic calls |
| Canonical writer ownership, short-write provenance (canonical_write_*, 1,400 lines) | Deleted; fn-153 removes the writer. Until then `internal/canonicaljson` is in the pure set and checked like any other package | None |
| Pinned standard-library sources and startup inventory (sources files, 800 lines) | The stdlib allowlist file names the Go minor version it was reviewed for; a test fails when `go version` moves to a new minor until the file is re-reviewed and its version bumped | A patch release changing stdlib purity; the toolchain is pinned at 1.27.1 and patch releases do not add host effects to pure helpers |
| Whole-program binding resolution for function values and interface methods | Dynamic-call rule with review marks | Reviewer error on a mark; mitigated by the mark listing in the test log and by the dated reason |
| Two-platform type checking from source | Two-platform `go list` for imports and syntax (cheap); one host-platform typed pass, plus the other qualified platform when `GOMAD3_ARCH_ALL_PLATFORMS=1` or in CI | A platform-specific file with a host effect; it still fails the two-platform import and syntax rules |

Wasm needs only the structural checker. Its pure set is `wasi/` minus `process.go` and `broker_process.go`, its pure import allowlist is the standard-library set plus the gomad3 packages it imports today. The property today's `helper_bridge_boundary_test.go` proves, that a helper callback reply cannot mutate the environment, becomes a unit test in `wasi` that calls the reply path with a hostile reply and asserts the environment is unchanged.

## API Contracts

```go
package archcheck

type Platform struct{ OS, Arch string }
type Finding struct{ Category, Path, Detail string }

type Config struct {
    Module       string
    Owners       map[string][]string           // owner -> package path prefixes
    OwnerEdges   map[string][]string           // owner -> owners it may import
    ModuleEdges  []ModuleEdge                  // forbidden import prefixes
    PureFiles    func(relativePath string) bool
    PureImports  []string                      // standard-library packages pure files may import
}

func Discover(root, goCommand string, platform Platform) (Inventory, []Finding, error) // unchanged
func Check(root, goCommand string, platform Platform, config Config) ([]Finding, error)
```

```go
package puritycheck

func Check(root, goCommand string, platform archcheck.Platform, pure func(path string) bool) ([]Finding, error)
func PublicSignatures(root, goCommand string, platform archcheck.Platform, consumer string) ([]Finding, error)
```

Each module owns its configuration in a test file: `tools/gomad3/architecture_test.go` and a new `tools/gomad_wasm/architecture_test.go`. Finding categories kept from today: `ownerless`, `owner-edge`, `module-edge`, `required-edge`, `uncovered-source`, `package-error`, `unclassified-module`, `public-signature`, `host-effect`. New: `pure-import`, `pure-syntax`, `dynamic-call`, `stale-review`. Removed: `unresolved-effect`.

Standard-library allowlist file format, `puritycheck/stdlib.go`:

```go
// Reviewed against go1.27. A new minor fails TestStdlibAllowlistVersion until re-reviewed.
const reviewedGoMinor = "go1.27"
var pure = map[string][]string{
    "bytes":   {"*", "!NewReader"}, // "*" admits every symbol, "!" removes one
    "time":    {"Duration", "Time.Add", "Time.Before", "Date", "Unix", "UTC", "Parse", "!Now", "!Since", "!Sleep", "!After", "!NewTimer", "!Tick"},
    "encoding/json": {"*"},
    "compress/gzip": {"*"},
    "os": {}, "os/exec": {}, "net": {}, "math/rand": {}, "crypto/rand": {}, "syscall": {"Errno", "E*"},
}
```

The initial table is seeded from today's `memorySymbols` and host classification in `standard.go`, then reduced to what the pure sets reference.

## Edge Cases & Constraints

- **Mixed packages.** `runner/internal/campaign/controller.go`, `target/capability_evaluation.go` and `runner/internal/execution/simulation_progress.go` are pure files inside impure packages. A call from a pure file to a function in the same package but an impure file is a `host-effect` finding, as today.
- **Generated code.** The generated choice wire file is in the pure set; the checker reads it like any other file.
- **`time`.** `time.ParseInLocation` with a non-`UTC` location is a host read. The heuristic admits `ParseInLocation` only when the location argument is the identifier `time.UTC`; any other expression is `host-effect`.
- **Build tags and cgo.** Files excluded by the platform's build constraints are skipped on that platform. Pure files with cgo are rejected outright.
- **Dynamic calls on interfaces declared in the pure set.** A call on an interface declared in a pure package whose every implementation in the module is in the pure set is static, not dynamic. Implementations are found by method-set matching within the module; implementations from outside the module make the call dynamic.
- **Review mark hygiene.** A mark without a date or reason is `stale-review`. A mark older than 365 days is a warning in the log, not a finding.
- **Performance.** Structural checks per module under 5 seconds. Typed pass for the gomad3 pure set under 30 seconds on one platform. Baseline on 2026-10-12, linux/arm64: the three purity and signature tests take 86 seconds wall clock, two platforms each.
- **Fixtures.** The 36 fixtures under `testdata/architecture/effects` are re-triaged. Host-effect fixtures (8) and the pure fixtures keep their expectations. Callback-resolution fixtures become `dynamic-call` fixtures: unmarked they must report, marked they must pass.
- **Error cases.** A module with a `go list` error reports `package-error` and stops. A pure set path that matches no file is a configuration error, reported as a test failure, so a renamed file cannot silently leave the set. A review mark on a non-dynamic call is `stale-review`. A stdlib symbol absent from the table is `host-effect`, never silently admitted.

## Acceptance Criteria

- **R1:** `internal/gomadtool/architecture` is deleted. `internal/archcheck` and `internal/puritycheck` together are at most 2,500 non-test lines and 1,500 test lines, measured by `wc -l` in CI and failed by a test when exceeded.
- **R2:** `tools/gomad_wasm/architecture_test.go` runs the structural checker over the wasm module on both qualified platforms with the pure set `wasi/` minus `process.go` and `broker_process.go`, and fails on a `pure-import`, `pure-syntax`, `module-edge` or `public-signature` finding. gomad3 no longer loads the wasm module. Error case: adding `import "os"` to `wasi/clock.go` fails the wasm module's own test.
- **R3:** The gomad3 structural test enforces the fn-164 import direction with an allowlist of the shared packages that still import backend-bound packages today; removing an entry from the allowlist after the corresponding package moves is the only change needed.
- **R4:** The purity checker reports a `host-effect` for each of the eight host-effect fixtures (clock, environment, filesystem, process, network, entropy, timer, goroutine) and reports nothing for the pure fixtures. Error case: `time.ParseInLocation(layout, value, loc)` with `loc` not the identifier `time.UTC` is a `host-effect`.
- **R5:** Every dynamic call in the gomad3 pure set carries a review mark, the marks are listed in the test log, and a mark on a line without a dynamic call fails the test with `stale-review`.
- **R6:** The standard-library allowlist names its reviewed Go minor and a test fails when the active toolchain's minor differs. Error case: a call to a `time` or `os` symbol absent from the table is `host-effect`.
- **R7:** The helper-bridge property is a unit test in `wasi`: a hostile callback reply leaves the environment byte-identical. Error case: a deliberate mutation in the reply path fails that test.
- **R8:** No standard-library source file is read or hashed by either checker; `go list -export` and `go/importer` supply types. The typed pass over the gomad3 pure set completes in under 30 seconds on the host platform.
- **R9:** `TestHostPackageVet` and `TestPublicPackagesDoNotExportTypeAliases` keep their behavior on the retained `Discover` and `PublicSignatures` entry points.

## Boundaries

- Do not change any pure set membership as part of this work; a file that fails the new checker is fixed in a follow-up or marked, never dropped from the set.
- Do not add `golang.org/x/tools` or any other dependency to gomad3 or gomad_wasm.
- Do not touch CI workflows beyond what R2 needs; wasm CI is fn-164 R9.
- Do not keep the ownership proofs behind a flag for later; they are deleted with their tests.
- Do not change the Finding categories consumed by existing tests except as listed in API Contracts.

## Decision Context

- Applied after fn-164, the former WASM module is `tools/gomad3/backends/wasm` in the single gomad3 module. Its own architecture test and helper-reply unit test live in that subtree and check the same WASI membership/exclusions against the folded module's inventory. References above to the separate module are pre-fold paths, not a requirement to restore it. Root tests must not duplicate the old cross-module WASM purity load.

- The ownership proofs were built under fn-109 R8 when canonical bytes and byte-identical formats were a requirement. fn-109 later dropped byte-for-byte compatibility and fn-153 retires canonical JSON, so the proofs defend a property no longer required.
- The deep prover's distinguishing value for wasm was following calls into shared gomad3 packages; proving those packages pure at their own boundary is a stronger guarantee at lower cost and stops re-checking them per caller and per platform.
- Review marks were chosen over a global suppression list so the reason lives next to the call and shows in diffs; a dated mark gives a reviewer the question to ask.
- Export-data type loading was chosen over source type-checking because it removes the pinned-source apparatus, which is most of the deleted code, and because gomad3 pins its toolchain already.
- The pinned-source design is brittle in practice. Run with a go1.26.0 host toolchain instead of the pinned 1.27.1, the native purity test reports 1,648 `unresolved-effect` findings, all through `record.Uint64String.MarshalJSON`, and zero host effects: the stdlib summaries no longer match, so every JSON call degrades to unknown. A version-keyed allowlist fails with one message naming the version instead.
- Budget is a test, not a guideline, so the package cannot regrow a proof at a time.
