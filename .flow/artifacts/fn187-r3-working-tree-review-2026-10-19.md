# R3 working-tree review

These fresh source-only audits cover the actual uncommitted worktree/index delta
and new consumers in `/home/agent/wt/wasm-fn187-functional`, not the empty committed
range examined by the canonical wrapper. R3 remains in progress. Correctness
findings below are blocking until repaired and verified. Reports are verbatim.

### Correctness axis

## Quality Audit — Correctness axis: fn187 R3 actual worktree

### Summary

203 tracked paths plus task-relevant untracked sources/fixtures/tools inspected. Critical 0 · Should Fix 3 · Consider 0 · Ship: ⚠️ Fix first.

### Should Fix

- **tools/gomad/internal/backends/wasm/wasi/broker_lifecycle_corpus_test.go:135** (Conf 100): Replacement discards `idle`’s returned time, losing deleted `TestBrokerRevocationReleasesSurvivorProgressAfterReap`’s `now == 1000` assertion. `observe` checks the separate broker clock — record/assert the actual return in both crash/stop cases. Evidence: walked.
- **tools/gomad/internal/backends/wasm/wasi/broker_lifecycle_corpus_test.go:57** (Conf 100): Constructor failures retain only error text, losing deleted compressed-profile admission’s `errors.Is(err, world.ErrCapacity)` assertion — record/check creation error classification or retain that typed assertion. Evidence: walked.
- **tools/gomad/internal/backends/wasm/wasi/network_cluster_corpus_test.go:113** (Conf 100): Group-rejection replacements compare error messages, losing deleted atomic-group/reservation tests’ exact `err == netmodel.ErrResourceExhausted` checks — preserve sentinel identity in the observations and assert it for live/scheduled and reservation cases. Evidence: walked.

### Evidence and limits

- Executed: all ten phase source/tool manifests match current bytes; accepted staged-diff hash unchanged; all nine owner-ledger hashes match.
- Cited: five-mode Rust conservation reports and Go statement conservation contain zero losses/gaps. These establish coverage conservation, not the missing assertions above.
- Walked: broader helper literals/feature expectations, namespace/volume/network/broker adapters, Go provenance and fail-closed selector/object/source/LLVM controls.
- No builds, tests, helpers or LLVM executed. Generated fixture semantics were sampled; exhaustive assertion equivalence and primary integration are not certified.

### What's Good

- Frozen phases, explicit selections and narrow LLVM diagnostic exceptions make evidence provenance reviewable.

### Standards axis

## Quality Audit — Standards axis: fn187 R3 actual worktree

### Summary

- Scope: 203 tracked files versus accepted index, plus task-relevant untracked source/fixtures and coverage tooling. Should Fix 0 · Consider 1 · Blocking: none possible.

### Consider

- **tools/gomad/internal/backends/wasm/wasi/broker_lifecycle_corpus_test.go:75** (Conf 75): `observe` freezes private `slot/request/arrivals/pending` bookkeeping into every golden snapshot, including unrelated profile-limit cases; these assertions exceed the accepted broker tests and existing outputs/World/network corpus schema, coupling R4 refactors to fixture rewrites — restrict snapshots to required observations and preserve necessary private assertions as focused invariants. Evidence: walked.

### Scope and limitations

- Walked broader Rust/Go helper, broker/network, namespace/volume adapters, oracle additions, regeneration ownership, private bindings and coverage guards; inspected frozen feature-tool copies and sampled generated fixtures.
- Read-only review. No builds/tests/helpers/LLVM executed. Retained five-mode conservation, ten pin-check and six-mode Clippy results were cited from logs; generated JSON was sampled, and this report does not certify every changed file.

### What's Good

- Source/tool/object pins, exact selections and narrow LLVM exception guards make evidence provenance explicit; historical limitations remain recorded.
- Mechanical size check found no Rust/Go/Python file newly crossing 1,000 lines.

Correctness: 3 findings; worst tier Should Fix.
Standards: 1 finding; worst tier Consider.

## Conductor disposition

All three correctness findings are accepted after tracing actual adapters and
the deleted assertions retained in primary `wasi/broker_test.go`. `idle` returns
`response.Time - world.InitialTime`, independently of the observed broker clock;
error strings do not preserve `errors.Is` or exact sentinel equality.
Prepare failing consumer checks before fixing adapters, and retain same-text
wrong-error negative controls. No production semantics or original corpus bytes
may change. New Go-authored observations must preserve literal original
assertions and pass independent review before regeneration is accepted.

The standards Consider is retained, not a blocker. Do not rewrite golden
bookkeeping expectations during correctness repair or the subsequent zero-diff
module refactor; reconsider projection breadth as a separate deliberate change.

All five passing conservation phases remain evidence for their exact pre-repair
source. New assertions and changed corpus inputs require applicable fresh
verification; do not relabel the old manifests as repaired-source results.

## Assertion repair verification

All three findings are repaired in the Go corpus consumers. Intended RED and
focused GREEN logs are retained under
`/home/agent/wt/evidence/fn187/r3-current/typed-assertion-controls-{red,green}-2026-10-19.log`.
Four production-source overlay mutations reject wrong idle returns, same-text
untyped capacity errors, and same-text unrelated/wrapped resource errors;
actual source hashes before/after match. Receipt:
`typed-assertion-mutations-2026-10-19/receipt.json`.
All eight normal corpus/control roots pass with no skips. Two regenerations
agree byte-for-byte. The delta proof preserves all prior fields in61 added cases
and introduces only the restored Go assertion metadata. Fresh checks preserve
all526 original contracts and215 original helper cases.

Correctness source rereview resolves all three findings with no new findings;
fresh full conservation verification remains required.

## Replay provenance rereview

Initial bounded review found two Should Fix guard gaps: whitelisted Go paths
were not bound to reviewed hashes, and copied executed tool roots were captured
without comparison to reviewed maps. Both are fixed. Reviewer report:

Both guard findings are resolved in source. No new findings in this bounded rereview.

- Walked: `build()` checks exact reviewed per-mode tool maps, fixture-proof digest and all three Go source hashes before capture.
- Executed: manifest SHA matches `a2a0fce819a1761d392aba7fe443de7e140065a240bb80199f7815b70ddbe38d`; proof, three Go files and all 35 tool entries in each mode match.
- Walked: `build-origin.json` records the reviewed manifest digest and explicitly identifies object reuse.

No tests, launcher, helpers, LLVM or compiler executed. Replay results remain pending.

Conductor preflight additionally rejects altered manifest/tool maps before phase
creation. Owned session25445 terminated exit0: all five fresh replay modes pass
full/corpus/invariant, cursor and conservation checks. Preserved baseline-covered
production lines are15626/14971/14963/15103/14276, each with zero loss/union gaps.
Default Go full/corpus/invariants and statement conservation also pass.
Session36566 terminated exit0 after independently verifying all ten phase
source/tool/object pins against current bytes. The accepted R1/R2 index is unchanged.

## Final correctness acceptance

## Quality Audit — Correctness axis: fn187 R3 actual worktree

Critical 0 · Should Fix 0 · Consider 0 · Ship: ✅ SHIP for R3.

All three assertion-conservation findings and both replay-provenance findings are resolved. No remaining correctness blocker identified for task .3.

- **Executed:** all ten fresh phase manifests, archived source/tool copies, object pins and 90 independently owned executables match their hashes; zero source/tool/object drift. Accepted staged-diff hash remains unchanged.
- **Cited:** fresh five-mode full/corpus/invariant execution, cursor proofs and conservation commands exit 0. Baseline-covered Rust lines preserved: 15,626 / 14,971 / 14,963 / 15,103 / 14,276; every loss/gap list is empty. Per-module reports cover nine modules.
- **Cited:** fresh Go full/corpus/invariant arms pass with 67/27/9 roots; all four new assertion controls pass without skips. Statement conservation has zero losses/gaps.
- **Walked:** authenticated binary reuse is explicitly disclosed and restricted to unchanged Rust compiler inputs; execution and profiles are fresh.

This accepts the reviewed R3 worktree through the documented fallback. No compiler, runtime, LLVM or tests were executed by this reviewer. Primary integration and overall spec completion remain separate.

Final correctness:0 findings. Standards:0 Should Fix/1 Consider, retained as
nonblocking. Reviewer and writer are the same model family, explicitly disclosed.
