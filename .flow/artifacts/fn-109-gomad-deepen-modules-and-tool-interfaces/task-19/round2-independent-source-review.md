# Task 19 round-two bounded independent source audit

Outcome: one actionable task-19 checker finding remains, with two real-Go reproducer variants. The earlier nine escapes and four controls now behave correctly on both qualified platforms. This is a bounded source audit, not a formal backend verdict, SHIP, or native-acceptance waiver.

## Frozen identity and routing

Before inspecting or copying the checker, verified every entry in `round2-final-source.sha256` (977 files; manifest SHA-256 `d4652c71a5f87802d1df2ff86ba5b941ece75f89c0642b94d164251733185834`) and `round2-task-owned-source.sha256` (108 files; manifest SHA-256 `33c0e8bc9b252bad81204490308540f84139e1c3f176befa98eaeb95f8d56f21`). All entries still verified after scoped tests; task 18's 14 final-source pins also verified. Shared source, index, Git state, Flow state, task/spec files, and round-one evidence were not changed.

Read AGENTS, flowctl usage and `initialization-boundary.md`. Judged this distinct assignment once using the new `round2-review-state.json`: `no_key`, explicit `gpt-6.1-sol`/high retained, actual backend metadata unknown, same-family writer/auditor. No bridge, new agent, or rerouting.

The copied checker is preserved in `/tmp/task19-round2-audit.9zUZR5/architecture`. Frozen `effects.go` SHA-256: `400ef199460360ed87690fb8a04d6657d7bab4ddf971b6c5affbda52f77495db`; `initialization.go`: `2db7f3747c3f192372837619f4a7e07008b4f9e3a971208239efe0b3a7d5a36c`; `standard.go`: `5c2a524847bcf16b7eb2694255209a7c5a49ea15cda96d539ca3b6d71aef9106`.

## P1: function-range assignments lose yielded callback provenance

Location: `tools/gomad3/internal/gomadtool/architecture/effects.go:1002` (synthetic yield construction), with parameter rebinding at `effects.go:818`.

The new implicit iterator call models `for f := range iterator` as synthetic function parameters, but does not model the assignments required by `for f = range iterator`. For an existing identifier it binds the yielded value only in the synthetic call's cloned environment; the original variable retains its clean callback. A selector/index/dereference LHS is not assigned at all: only identifier expressions are considered when building parameters. A later invocation of the actual assigned callback can therefore pass the purity checker.

Exact minimal allowed-owner reproducer:

```go
// record/pure.go imports its permitted internal/canonicaljson helper.
func Check() {
    f := helper.Clean
    for f = range helper.Callbacks {}
    f()
}

// internal/canonicaljson/helper.go
func Clean() {}
func Dirty() { _ = time.Now(); Calls++ }
func Callbacks(yield func(func()) bool) { yield(Dirty) }
```

Second variant: `v := struct{ F func() }{helper.Clean}; for v.F = range helper.Callbacks {}; v.F()`.

Isolated test `/tmp/task19-round2-audit.9zUZR5/architecture/round2_probe_test.go:12`; execute from that scratch module:

```sh
GOWORK=off go test -count=1 -tags test_dep ./architecture \
  -run '^TestRound2IndependentProbes/(range-existing-slot|range-field-slot)$' -v
```

Each fixture first executes native stock Go and confirms exactly one Dirty/time.Now callback; its owned package edges also pass. Frozen checker findings are `[]` for both linux/amd64 and darwin/arm64, so both negative cases fail. Log: `/tmp/task19-round2-audit.9zUZR5/round2-probes.log`, SHA-256 `e53e6ba2b12f4833b64ad1de7fc89aa4f2ebdd280b8d3659bad165a4e630e117`. Test SHA-256 `840df9d935b2d123df0791f0e7f58b3e273c246657b5a581dc37be3c8567b9f0`. Same test's pointer-field mutation and mixed JSON map/slice addressability controls correctly reject.

This is a task-19 source/spec gap in the new range/yield summary, not a task-13–18 behavior regression or an unsupported-platform finding. Corrective coverage should retain the current declaration/yield-body checks and include assignment to existing identifier, field, index and pointed callback slots, both inside and after the loop.

## Confirmed coverage

The unchanged round-one fixture `independent_audit_test.go` (SHA-256 `d68f33e906c4e2de679290ee4eda683098aeaae6a51afb11822c627e89044f60`) was replayed against copied round-two bytes. All nine old escaping cases reject on both qualified platforms; fresh Clone, direct clock, direct JSON pointer, and pure-copy controls retain their expected behavior. Every case also checks native callback behavior and allowed owner edges. Initial replay passed in 10.647s; retained replay passed in 10.475s, log `round2-original-replay.log` in the scratch root, SHA-256 `58a5a3f8451a03e0bf720e92b77d7d517936859dde1e6ab53b8773c9fc0e9071`.

Scoped source-owned tests passed in 20.820s: `TestCallbackContainerMutations`, `TestDependencyInitialization`, `TestThirdPartyInitialization`, `TestStandardStartupIdentity`. These exercise copy/insert/clone aliases, pointer stores, iterator calls/yield bodies, addressable JSON slices/arrays/fields and nonaddressable map/array/field controls. Initialization cases cover blank/transitive imports, variable expressions, repeated init declarations, third-party sources, and sync.Once effectful/pure callbacks.

An additional real-Go mixed-package probe (`round2_initialization_probe_test.go`) passed in 1.091s on both source qualifications. A selectively pure target capability file causes the package's other source file to initialize: the second init and variable callback each produce their own clock finding. An unused effectful callable sibling with otherwise pure initialization is accepted, confirming initializers do not incorrectly promote every mixed-package callable to a pure root. Native initialization counts are 3 and 1 respectively. Log `round2-initialization-probes.log`, SHA-256 `092eb676644b632edd1c2ab93be4f0d4ac5f5dbce53fd2d88312c7560d45118c`.

Fresh platform-selected `go list -tags test_dep -json ./internal/gomadtool/architecture` metadata for darwin/arm64 and linux/amd64 matches the frozen inventories: the same eight production and five test files, no Error/DepsErrors. All 18 source-owned Test functions collect; scratch-only replay/probe tests also collect. Full collected names are in `round2-collected-tests.log`. This verifies collection and source selection, not target-native execution. Linux/arm64 remains unsupported for native acceptance; the local native commands only prove fixture behavior.

## Standard-process-startup boundary

The new boundary is initialization-only: `startupSources` is consulted solely by `checkStartupSource`, not by the callable graph or standard-call handler. The startup traversal recursively follows the actual selected package metadata import closure before handling each package, including blank/transitive dependencies. It visits a selectively pure package itself, thereby checking every sibling initializer, and builds each init function directly from its declaration rather than the name-colliding global function map.

Standard identity checks hash every immediate `.go` source file in the selected package directory (including inactive/test source, conservatively), in sorted filename order, and fail closed for missing/changed identities or unreadable source. The changed-time-source regression confirms rejection. The C pseudo-import skip is limited to selected standard cgo packages. Nonstandard/module/third-party packages are not exempted: typed source initialization is traversed; missing metadata/typed source is unresolved. Standard calls reached from application/dependency initializers still traverse the ordinary callback/effect graph; the effectful sync.Once fixture confirms this despite accepted stock startup identity.

The separate exact sync/atomic LoadUint32/StoreUint32 memory summaries are source-pinned and carry no callbacks; startup pins do not create a callable function allowlist. No additional actionable startup-boundary or JSON-addressability defect was reproduced in this bounded round.

Parent-reported root Quick success is contextual gate evidence only and was not rerun or treated as proof against the source finding. No broad runner/native suite was run. Round-one findings/report/manifests remain unchanged; the inherited stale D26 simulation pin and native gate qualification remain separate from this round-two checker finding.
