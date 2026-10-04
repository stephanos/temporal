# Task 32 independent source review

Verdict is NEEDS_WORK. One introduced conversion-alias defect must be repaired before root commits source progress. This bounded source review provides no formal SHIP or supported-native qualification.

Base and HEAD are f931c9879e3017b346562667b9f6fbcc4db458ec on gomad. The reviewed candidate is the uncommitted effects.go, standard.go and error_provenance_test.go captured by writer-stage.json and every review receipt. Root retains sole ownership of Git/index/commits, Flow, parent requirements, MILESTONES and admission. Reviewer changed no product source or those states.

Review instructions were read from AGENTS.md, Gomad README, MILESTONES, task32, original R8/R18/R19, task19, the source owner plan, requesting-code-review's code-reviewer.md template and verification-before-completion. The applicable Codex reviewer request is gpt-6.1-sol/high. Actual execution-model metadata is unavailable; writer and reviewer use the same Codex family. Existing routing remains Tier: session (jev-unavailable(no_key)); no judge, bridge, download or formal dispatch was repeated.

## Strengths

The Unwrap repair at standard.go:617 distinguishes exactly Unwrap() error from Unwrap() []error before visiting children. A named slice returned as error retains its own Is/As method; actual multi-error children still receive traversal. Invalid signatures remain uncalled by this traversal. Existing single-only errors.Unwrap, recursive wrapper/returned callback, argument/capture contexts, formatting precedence and JSON callback summaries pass fresh checks.

The fmt repair at standard.go:240 reuses callbackError and joins only the concrete Writer.Write error into result slot 1. The original count slot and tuple shape remain intact. Fprint/Fprintf/Fprintln, wrapped-Is, nil/clean/noninvoked return and unknown dynamic-return fixtures pass. Unknown callback receivers remain unresolved and fail closed.

The 29 new fixtures execute real temporary modules, keep only record.Check as a pure production root, and independently inspect linux/amd64 and darwin/arm64 metadata. All final dirty fixtures name their exact leaf callback and time.Now in one record.Check finding; clean/nil/unreachable controls have empty effects and the unknown writer-return case is unresolved. The external-return fixture uses a fresh local replacement module. Historical architecture tests and protected inputs retain original bytes.

## Issues

### Critical

None found.

### Important

1. Preserve the first element mutation through converted slice aliases at effects.go:750.

The conversion copies the abstractValue header and keeps its current elements pointer. When elements is nil, an assignment through the converted slice installs a new elements value only on that copy. The caller's original slice retains nil elements even though both Go slice values share one backing array. The existing new slice-alias test at error_provenance_test.go:49 starts with helper.Clean, so join mutates an already shared nonnil element abstraction and hides the defect.

The exact dirty reproduction is `v:=make([]func(),1);helper.Set(helper.Callbacks(v));v[0]()` with `type Callbacks []func();func Set(v Callbacks){v[0]=Dirty}` and Dirty incrementing Calls before time.Now. Both candidate and baseline execute the callback exactly once. Saved baseline production reports record.Check -> canonicaljson.Dirty -> time.Now on each supported metadata source set. Candidate production reports only unresolved callback on both, losing the known concrete causal path.

The clean companion replaces Dirty with `func Clean(){}` and assigns Clean through the same conversion. Its stock callback counter is zero. Baseline effects are empty on both metadata source sets; candidate rejects the pure call with unresolved callback. This is an introduced false positive as well as lost dirty provenance. A zero-initialized pointer conversion control still passes, so the finding concerns slice element ownership specifically.

Evidence is review-alias-full-baseline.json/log (exit 0, three fixtures) and review-alias-full-candidate.json/log (exit 1, dirty/clean slice failures, pointer control passing). Earlier two-fixture diagnosis receipts remain immutable under review-alias-baseline/candidate. review-alias-probe.py runs temporary Go source overlays, retaining exact supplemental fixture source and its hash in each raw log. Baseline overlays use the admitted immutable sources/baseline production files, previously verified against Git BASE bytes; candidate overlays replace only the new test file. Both full receipts bind the probe and runner hashes, baseline production hashes, exact outer argv/cwd/environment/tool/config identities and unchanged checkout source/protected maps. These overlays do not edit checkout or runtime inputs and are solely diagnostic source probes.

Repair within the admitted conversion code by sharing the writable slice-element provenance across the original and converted value even when the element abstraction starts empty. Retain the destination concrete method set without retagging the caller or replacing function/alias payloads. Add both zero-initialized dirty/clean slice cases to the admitted new regression file, retaining literal counters and exact causal/empty findings. Existing typed/interface/unrelated-caller/function/pointer cases must keep their behavior. Root must authorize the same task32 writer to apply and verify the repair, then return it for source review.

### Minor

No additional actionable source defect found.

## Verification and evidence

All requested fresh checks are terminal. Whole architecture and focused new/summary/initialization/mutation/range/precedence controls pass; the 29 new fixtures yield 58 supported-source metadata observations. The five actual root tests each ran once and passed, including both external compilation tests. Whole cmd/gomadtool passes. make validate executes every check without regeneration; diff/gofmt checks pass; errortype is empty with exit 0.

Fresh actual unfiltered pinned lint exits 1 with four diagnostics. Its raw log equals worker baseline and worker final byte-for-byte. Residuals are initialization.go:123 errcheck, standard.go:222 and :304 QF1003, and standard.go:390 errcheck. Introduced diagnostics are zero; resolved diagnostics are zero. Historical 419 findings were not rerun or represented as current.

The read-only review-audit.py verifies all 17 worker receipts and 13 review receipts, timestamp/elapsed consistency, exact supplied tool/config hashes, environments, architecture maps, protected aggregate, snapshot bindings, literal causal counters and actual RUN/PASS inventory. The worker audit independently exits 0. Baseline inventory is 14 Go files and new-test stages/final are 15; protected 1042 inputs retain aggregate b4ca4bdd63eb9426d63446b41042ccc0f825c8402b33faf6197f020d855f1e61. Admission's original task19, AGENTS, README, config and prior task19 evidence hashes match.

The new test file is not byte-identical from test-first to final. Its sole difference is the metadata t.Logf at final line121. Removing that exact logging line recovers the original test-first fixture recipes, counter assertions and analyzer assertions in every retained stage. Historical tests remain byte-identical. This corrects the overstrong initial dispatch wording without changing causal RED evidence.

Generator inputs were inspected in toolchain/version/descriptor.go and internal/gomadtool/generation/{protocol,boundary}; the admitted source paths are outside those input lists. All generator inputs and outputs remain protected. Full native gates require patched Go on darwin/arm64 and linux/amd64. The patched executable is absent here. Stock Go1.27.1 linux/arm64 execution and supported-platform Load/vet metadata are developmental only.

## Recommendations

Route the causal slice repair through root to the same admitted writer. Keep current review and worker receipts immutable, use new receipt names for the repaired candidate, and rerun the affected provenance suite, whole architecture package and applicable boundaries/consumer/static/lint/errortype checks against stable source. Return that repair to this reviewer before staging.

## Assessment

Ready to merge? No. Source-progress commit? No, pending the one Important repair. The intended Unwrap and writer-error repairs have passing causal evidence, while the conversion change introduces a proven clean-alias rejection and loses a known dirty callback path. Source gate is NEEDS_WORK; all commands and delegates are terminal.

Original R8/R18/R19/task19/fn105D4/predecessors/task21, original first-baseline fixed identities, full/completion/formal review and both native-platform/affected-consumer acceptance remain open. Nothing in this bounded review waives those requirements.
