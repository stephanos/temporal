# Task 32 source handover

Architecture checks now retain the concrete callbacks of named slice errors returned by single Unwrap, typed named conversions returned through interfaces, and fmt writer errors. The production changes occupy effects.go and standard.go; error_provenance_test.go adds 29 causal fixtures with only record.Check as a pure production root.

Status remains in_progress. The conductor owns independent source review, Git/index/commits, Flow and acceptance. Base is f931c9879e3017b346562667b9f6fbcc4db458ec; this worker made no commits or delegates. Actual execution model metadata is unavailable.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)

## Evidence

evidence.json references 17 immutable command receipts, their raw logs and five bounded source snapshots. Each command receipt records argv, cwd, relevant environment, timestamps, elapsed time, child exit/timeout, tool/config/log hashes, complete architecture maps and the protected aggregate. Baseline has 14 architecture Go files; every test-first and later stage has 15. audit.py reads these records and writes no files.

The prechange causal suite failed 14 named cases. All 29 stock-host fixtures compiled and their literal callback/count assertions passed. Thirteen failures missed known reachable callbacks on both supported metadata source sets; the fourteenth lost unknown writer-return provenance. The nil named-slice case required the conversion repair as well as traversal. Existing direct slice, actual multi-Unwrap, single-only errors.Unwrap, clean/unreachable and function/slice/pointer alias controls already passed and retained their expectations.

Unwrap-stage tests passed 10 fixtures; conversion-stage tests passed 21; writer-stage tests passed 8. Final causal tests passed all 29, with 58 explicit metadata observations. Dirty fixtures retain one record.Check -> exact Leaf callback -> time.Now path; clean/nil/unreachable fixtures retain no effects. The writer unknown-return fixture remains unresolved and fail-closed.

Final whole architecture passed all 22 top-level tests, including unchanged callback-summary, initialization, range, mutation, recursive, JSON and source-identity tests. The required five actual public/consumer boundaries each ran once across final-boundaries and final-public-consumer-boundaries. Broader pure-effect, host-vet and exact-edge checks also passed. All gomadtool command tests passed; source import inspection found the module-root architecture tests are the checker package's direct consumer. Generator input inspection found no changed file in version, protocol, boundary or compatibility input sets; make validate passed all checks without regeneration. Diff and gofmt checks passed.

Actual unfiltered pinned lint was red before edits and remains red with the same four diagnostics, byte-identical logs. Residuals are initialization.go:123 and standard.go:390 unchecked fmt.Fprintf returns (errcheck), plus standard.go:222 and :304 tagged-switch suggestions (QF1003). Introduced findings 0; resolved findings 0. Baseline and final errortype both exited 0 with empty output. No diagnostic, exclusion, config or pin was waived or changed.

All 1042 protected tracked inputs retain aggregate b4ca4bdd63eb9426d63446b41042ccc0f825c8402b33faf6197f020d855f1e61. Original tests, public APIs, task19/README/AGENTS contracts, fixtures, seeds, fixed-input identities, runtime, dependencies and other production files remain byte-identical. Saved baseline production bytes also match Git admission BASE.

Stock execution used cached Go1.27.1 on linux/arm64. Load inspected linux/amd64 and darwin/arm64 metadata; host-vet additionally inspected linux/arm64. These are developmental checks. Patched Go is absent; supported native execution, broader/full/formal qualification, original R8/R18/R19/task19/fn105D4/predecessors/task21 and first-baseline fixed-identity acceptance remain open. The historical whole-module 419-finding lint receipt was not rerun or represented as a current count.

Defect route:
- prior fixes: the source note identified earlier repaired graph-key, struct/interface/Unwrap-return and %w paths; Git shows task19 owns those changes. Root admitted this sole bounded source writer. Memory search found no matching error-provenance fix. GitHub/tracker checks were unavailable because gh auth reports an invalid token.
- diagnosis: the real stock counters exclude unreachable callbacks and fixture errors; direct/multi controls isolate the named-slice traversal omission; typed parameter/Supplier/external cases isolate conversion type loss; writer tuple cases isolate discarded returned error provenance. Stage gates confirm each correction.
- introduced by: skipped; no known-good revision for these surviving cases was supplied.
- base: causal-red.json binds the unmodified admitted production plus test-first fixture. Head candidate: final-causal.json binds the repaired source with all fixtures green.
- live: no live application surface; real fixture execution and analyzer Load supply the proof.

Run the read-only audit from the repository root with `python3 .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-32/audit.py`. Every owned command is terminal; no delegates were spawned.
