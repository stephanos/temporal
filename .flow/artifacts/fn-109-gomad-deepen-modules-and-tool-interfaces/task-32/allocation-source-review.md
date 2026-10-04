# Task 32 corrective allocation source review

SOURCE_PROGRESS_COMMIT_ONLY. No actionable introduced defects remain in the reviewed candidate. The accepted fresh-slice, fresh-map and empty-map-literal findings are closed, including the six admitted allocation alias families. Root may commit verified source progress after auditing these new artifacts. Original acceptance remains open.

## Strengths

`effects.go:122` distinguishes an empty aggregate bottom cell from a statically typed zero element. Fresh array, slice, map and `new(T)` origins retain existing shared element or `$pointee` cells through named reference conversions. Zero construction descends through value aggregates and stops at reference types, so the recursive Node control terminates. Unknown non-allocation producers retain the existing fail-closed path.

`effects.go:143` copies array/struct value cells at the conversion boundary while retaining nested reference storage. The two value-copy controls now keep an unrelated caller clean, and the nested-reference control still records `record.Check -> canonicaljson.Dirty -> time.Now`. This conversion-local correction follows the retained origin-stage failures and leaves assignment/parameter handling, storage representation and graph/memo logic unchanged.

The original exact Unwrap signature distinction and concrete fmt writer-error result remain intact in `standard.go`, whose bytes match the original writer stage. Fresh tests retain the initial conversion/interface, error traversal, nil/unreachable, format/JSON precedence, tuple/count and writer-return controls. All 39 previous fixture bodies remain an exact prefix of the 67-fixture file; old tracked tests remain unchanged.

## Issues

### Critical

None identified.

### Important

None identified. The six admitted dirty/clean alias pairs now retain exact callback paths or empty findings on both supported metadata source sets. The earlier make-origin and literal findings are closed by the permanent fixtures, rather than merely superseded by a different probe.

### Minor

None identified within this bounded source repair.

## Verification

Fresh `review-allocation-*` receipts record whole architecture 26 top-level tests, focused 16, actual root public boundaries 5, broader purity/edges/host-vet boundaries 3, and gomadtool consumer 30. All selected tests pass. Each architecture run executes 67 stock-host causal fixtures and 134 metadata observations. Dirty fixtures retain the specific callback-to-clock path; clean cases have empty findings; four unknown-provenance cases remain fail-closed. Errortype, make validate without regeneration, diff-check and gofmt pass.

Actual pinned unfiltered lint exits 1 with four byte-identical inherited diagnostics at initialization.go:123 and standard.go:222,304,390. Introduced 0, resolved 0. This remains a residual gate, with no suppression or sweep.

`allocation-review-audit.py` binds every new command, environment, cwd, timestamp, elapsed duration, timeout policy, child exit, raw log, tool/config identity, complete 15-source map and 1,042 protected inputs. It delegates immutable historical-view verification to `allocation-repair-audit.py`, preserving 234 historical artifacts, 80 earlier receipts, four scout receipts and 14 writer allocation receipts. The review report and command proof are frozen before the worker-audit gate runs.

Root separately staged the user-authorized nine-line AGENTS research-routing addition while HEAD and product source stayed frozen. Current/index AGENTS SHA-256 is 3ce85bf0fd1be8f06398b71eeae1959f0ee7081c885a431bb7764c79c39d602f. The new read-only worker-audit wrapper makes one exact-count in-memory substitution so the original admission's AGENTS assertion checks Git BASE bytes. All archived auditor bytes and other assertions remain unchanged. The new audit separately reconstructs the exact authorized document delta, binds working/index bytes and requires AGENTS.md as the sole staged path. Research routing does not change the reviewer selection. The document review belongs to root.

The corrected stock-valid RED shows 12 introduced alias failures. Git BASE passes those aliases and fails seven supplementary inherited controls. The origin-stage repair leaves two value-copy failures; the final candidate resolves them. The invalid unused-import fixture remains inconclusive, the two earlier zero-selection commands remain inconclusive, and the prior concurrency-audit failure plus stable retry remain disclosed. The writer's two draft-auditor failures are proof-construction failures, not product-gate failures. The initial test-stage metadata logging-only correction remains as previously documented.

## Recommendations

Root should retain the prior NEEDS_WORK reports unchanged and use this report/checks pair for the frozen allocation candidate. Keep inherited zero-variable, map-key formatting and general assignment/parameter-copy limitations explicit. This repair supplies no universal length/capacity, copy or formatting completeness claim and requires no broader source sweep.

## Assessment

Ready for a verified source-progress commit only. Original R8/R18/R19, task19/fn105D4, predecessors, task21, fixed first-baseline identities, full/completion/formal, affected-consumer and both patched-native qualifications remain open. Stock Go1.27.1 linux/arm64 execution is developmental; linux/amd64 and darwin/arm64 Load/vet evidence supplies no supported-native execution. Patched native Go is absent.

BASE and HEAD remain f931c9879e3017b346562667b9f6fbcc4db458ec on gomad, with the candidate uncommitted. Requested reviewer gpt-6.1-sol/high and writer share the configured Codex family; actual execution model metadata is unknown. Tier remains session (jev-unavailable(no_key)). This same independent reviewer context performed no product, Git/index, Flow, MILESTONES, pin, dependency, config or lifecycle mutation. Root owns lifecycle and commit.
