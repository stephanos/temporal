# Task 19 round-three bounded corrective source review

The original existing-variable and field-slot range escapes are fixed. Two actionable assignment edges remain in the correction, reproduced with native Go behavior and both qualified source sets. This bounded source review supplies no formal backend verdict, SHIP decision, or native-acceptance waiver.

## Frozen identity and review scope

Before copying or reviewing source, verified every entry in `round3-final-source.sha256` (978 files; manifest SHA-256 `6f24b2bc9a0d867ae2ee0e8d2c9c5c92f64d32b9a9753bb0a4396d794c5e2921`) and `round3-task-owned-source.sha256` (109 files; manifest SHA-256 `a0b9efadfd12b226b4df417a22bcf5528f09fb87813a71440eeefb50a3858b63`). All entries and the 14 task-18 pins still verified after tests. The owned manifests confirm that only `effects.go` changed and `range_test.go` was added relative to round two.

Read AGENTS/model routing and complete flowctl usage. Judged this distinct assignment once with new `round3-review-state.json`; the result was `no_key`. Explicit Codex reviewer `gpt-6.1-sol`/high retained, actual backend metadata unknown, writer and auditor same-family. Applied the code-review skill's correctness criteria directly under the assignment's single-reviewer restriction. No agent or bridge was dispatched.

Read both changed files and traced the shared assignment helper's three consumers, ordinary assignments, iterator yield calls, and ordinary range values. Source and logs remain in `/tmp/task19-round3-audit.msiFNb`. Frozen `effects.go` SHA-256 is `f2cc79df4d21a556402c3161dffe2b634dfefececbb7f799731aa46cafb5adca`; `range_test.go` is `401ca9b50b3d9ad610ecbd4729bd0b4b06e6a10b859b2be3eae199cb75f53f79`.

## [P1] Unwrap parenthesized assignment destinations

**File:** `tools/gomad3/internal/gomadtool/architecture/effects.go` (lines 65-99)
**Reviewer:** internal (correctness)
**Failure scenario:** A function iterator assigns Dirty through a valid parenthesized destination, then the pure root invokes that destination. Native Go calls time.Now once; the checker accepts the root.

The new helper switches directly on the destination AST type and has no `*ast.ParenExpr` case or fail-closed fallback. Thus `for (f) = range helper.Callbacks {}` performs no abstract assignment. The previously clean callback remains in the original slot. Parenthesized field destinations have the same failure. Minimal root bodies are:

```go
f := helper.Clean
for (f) = range helper.Callbacks {}
f()
```

```go
v := struct{ F func() }{helper.Clean}
for (v.F) = range helper.Callbacks {}
v.F()
```

The permitted internal/canonicaljson helper supplies `Callbacks(yield func(func()) bool) { yield(Dirty) }`, `Clean() {}`, and `Dirty() { _ = time.Now(); Calls++ }`. Both native fixtures report one actual effect and pass owner/edge rules. Both frozen checker source qualifications return `[]`.

## [P1] Analyze effects in assignment index expressions

**File:** `tools/gomad3/internal/gomadtool/architecture/effects.go` (lines 86-90)
**Reviewer:** internal (correctness)
**Failure scenario:** A pure iterator yields Clean, but locating its assigned slice element invokes a clock-reading index helper. Native Go calls time.Now once; the checker accepts the root.

The new shared IndexExpr destination handler evaluates only `target.X`. It never evaluates `target.Index`, although Go evaluates this expression on each yielded assignment. The synthetic yield's assignments are outside its body, so no later AST traversal recovers the omitted call. Minimal root body is:

```go
v := []func(){helper.Clean}
for v[helper.Index()] = range helper.PureCallbacks {}
v[0]()
```

The helper supplies `Index() int { Dirty(); return 0 }` and `PureCallbacks(yield func(func()) bool) { yield(Clean) }`. Here the assigned and subsequently invoked callback is entirely pure; the only effect is in the destination's index. Native effect count is one, owner/edge checks pass, and checker findings are `[]` on linux/amd64 and darwin/arm64. Include a no-yield control when correcting this edge so the checker does not evaluate destinations for uninvoked yield callbacks.

## Exact reproduction and retained evidence

Run from `/tmp/task19-round3-audit.msiFNb`:

```sh
GOWORK=off go test -count=1 -tags test_dep ./architecture \
  -run '^TestRound3AssignmentEdges$' -v
```

This independent three-case test first executes each stock native Go fixture and asserts exactly one Dirty callback, then checks both qualified source platforms. It exits 1 in 1.769s, with all three cases failing because their actual effect escapes. Test file `architecture/round3_assignment_probe_test.go` SHA-256 `beb009277e1d260211caea89342315be8737f57598ba7cc7acb73bd5a5502962`; log `round3-assignment-probes.log` SHA-256 `5390e499ea45e8bd02159d4fe5a16b38b25d4e659c1e2f07a6268c92b17555f4`.

These are remaining task-19 range-summary correctness/spec gaps. The analogous omissions for ordinary assignments already existed before this corrective round, so this review does not attribute them as newly introduced baseline regressions. They directly affect the new shared range-assignment path and the requested complete valid-assignment coverage.

## Confirmed correction behavior

Independently ran the unchanged 13-case original escape/control fixture, the unchanged four-case round-two probe fixture, and the writer's 14 new range cases together. All pass, total 23.307s. The old original fixture passed in 11.27s, writer range cases in 8.56s, and round-two probes in 3.47s. Log `round3-existing-replays.log` SHA-256 `e780eeed9c1d3a4f51a4d55ea6e5d53429224b3b3ee1b2e1f41d034bc257d83d`.

The shared helper correctly mutates existing abstract slots through join, so shallow environment captures retain the callback assignment. The iterator summary now keeps the actual yield signature and assigns yielded arguments before the body. Existing and captured identifiers, fields, indices, pointer slots, two yielded values, declared-variable captures, ordinary slice/map range values, clean callbacks, and uninvoked yields all retain their tested behavior. The blank identifier is discarded explicitly. The original copy/insert/clone/pointer/JSON/init regressions remain fixed. No standard callback source identity or startup boundary code changed in this round.

## Overall Verdict - correctness

**Correctness:** incorrect within the bounded corrective scope.

The ordinary existing-slot correction works, but two valid destination forms still allow a real host clock effect to escape. Neither finding depends on unsupported linux/arm64 native acceptance. Local stock native fixture execution proves Go behavior only. No full-root Quick, broad checker suite, runner suite, native gate, shared source/index/Flow mutation, or prior-artifact edit was performed by this reviewer.
