# Task 32 corrective independent source review

Verdict remains NEEDS_WORK. The original slice finding is closed, but one analogous introduced map-alias defect remains Important. Root should retain this source batch uncommitted until that bounded alias repair passes review. This report is the authoritative corrective source verdict; the original independent-source-review.md remains immutable historical NEEDS_WORK evidence.

BASE and HEAD remain f931c9879e3017b346562667b9f6fbcc4db458ec on gomad. The frozen candidate is repair-final.json. Root retains sole Git/index/commit, Flow, parent, MILESTONES and admission ownership. Reviewer edits are confined to this new report/checks/audit and review-repair-* diagnostic scripts/receipts/logs. No product source, public surface, dependency, pin, runtime, suppression or config was edited.

The same reviewer context applied the original requesting-code-review template and verification-before-completion instructions against task32, original R8/R18/R19/task19, AGENTS Codex routing, README and MILESTONES. Requested reviewer remains gpt-6.1-sol/high; actual host model metadata remains unavailable. Reviewer and writer share the configured Codex family. The existing Tier: session (jev-unavailable(no_key)) was retained without another routing or bridge invocation.

## Strengths

The accepted slice repair at effects.go:784 initializes a shared element abstraction at fresh make slice allocation before the conversion copies the abstract header. The element keeps its declared type and known-zero state. The permanent dirty/clean converted-slice regression at error_provenance_test.go:146 passes fresh literal runtime counters of 1/0 and both metadata source sets now retain the exact Dirty -> time.Now path or clean empty effects. Original pointer-alias control still passes. The original Important slice finding is closed.

The four additional controls at error_provenance_test.go:148 prove concrete zero-value Leaf.String methods remain visible, clean concrete values are accepted, fresh nil interface elements are accepted and unknown interface elements remain unresolved. The zero-element known state is scoped to fresh make slices; the unknown-interface control and original unknown writer-return case stay fail-closed. The original 29 fixture file prefix remains exact, including existing recipes/counters/assertions. standard.go is byte-identical to the initial reviewed candidate, so the signature-aware Unwrap and fmt writer-error repairs are preserved.

## Issues

### Critical

None found.

### Important

1. Converted maps still detach the first inserted element at effects.go:750.

The conversion shallow-copies abstractValue and therefore copies its current elements pointer. Fresh make(map[int]func()) has nil elements. The slice-only initialization repair leaves that map state unchanged. Set through a converted named map installs elements only on the copy; Go's two map values share the same storage, while the analyzer's caller does not observe the new callback.

The exact dirty case is `v:=make(map[int]func());helper.Set(helper.Callbacks(v));v[0]()` with `type Callbacks map[int]func();func Set(v Callbacks){v[0]=Dirty}` and Dirty incrementing Calls before time.Now. The clean companion assigns `func Clean(){}`. Both stock fixtures compile and execute their asserted callback counters, dirty 1 and clean 0, with only record.Check as a pure production root and empty PackageEdges.

Git-admitted baseline production detects record.Check -> canonicaljson.Dirty -> time.Now for dirty and accepts clean effects=[] on both linux/amd64 and darwin/arm64 metadata source sets. The repaired candidate reports unresolved callback for both cases on both source sets. Thus it loses known concrete provenance and introduces a pure-program false positive. This is the same shallow-copy ownership defect as the accepted slice finding, now concretely established for map aliases within the original task32 payload/alias-preservation obligation.

Immutable review-repair-map-baseline.json/log exits 0 for both cases; review-repair-map-candidate.json/log exits 1 with exactly two named failures. review-repair-map-probe.py uses a temporary Go source overlay only. The identical supplemental fixture source and generated test hash appear in both raw logs. The candidate overlay replaces only the new test; baseline additionally substitutes the immutable baseline effects.go/standard.go already verified against Git BASE. Receipts bind exact argv/cwd/environment/tool/config hashes, probe/runner hashes, baseline production hashes and unchanged checkout source/protected maps. No diagnostic fixture was installed in product source.

Preserve first-insertion callback provenance through the named map conversion without changing the caller's concrete type or weakening unknown-state handling. Retain exact dirty/clean permanent regressions in the admitted new test file and the six repaired slice/zero/interface controls. Root should evaluate and route the bounded repair through the same source writer, then return it to this reviewer. Do not generalize a builtin/unknown waiver from these initialized storage cases.

### Minor

None found.

## Verification and evidence

Fresh whole architecture and focused provenance/context/initialization/range/mutation suites pass, including 35 causal fixtures and 70 explicit supported-source metadata observations. Every dirty fixture retains its exact concrete leaf and time.Now on one record.Check path; clean controls have empty effects and two unknown-interface/return controls are unresolved. Five required root public/consumer tests each actually ran once and passed. Broader root pure-effect/exact-edge/vet tests passed, with host-package vet inspecting darwin/arm64, linux/amd64 and actual linux/arm64 source sets. The whole gomadtool consumer, make validate, diff/gofmt and errortype pass. All reviewer commands and delegates are terminal.

Fresh unfiltered pinned lint exits 1 and remains byte-identical to original baseline, initial final and writer repair logs. Its four inherited diagnostics are initialization.go:123 errcheck; standard.go:222 and :304 QF1003; standard.go:390 errcheck. Introduced/resolved diagnostics and line changes remain zero. No suppression or inherited-finding sweep occurred.

repair-review-audit.py reads and runs the preserved repair-audit.py against the frozen current tree, then independently checks the new command/env/timing/tools/config/log/source/protected bindings, actual selected RUN/PASS inventory, original prefix preservation, exact slice-only production delta, dirty/clean map overlay proof and historical immutability. Historical audits run only through repair-audit's transparently saved-stage view; the original audit was never run directly against the repaired tree. All 17 original worker, 13 original reviewer and 14 writer repair receipts remain bound; all 89 historical files and 1042 protected inputs are unchanged. The two wrong-package writer selections retain their explicit inconclusive status; actual root-package receipts and fresh reviewer root-package runs supply coverage.

Initial new test-first/final files retain the single metadata t.Logf difference already disclosed by the first review. The repair appends six fixtures after the exact original candidate file prefix. Historical tests remain byte-identical.

One reviewer audit attempt, review-repair-worker-audit, exited 1 at the whole-artifact equality assertion because the reviewer wrote this report while that audit was running. Source/protected maps stayed unchanged. That failed receipt remains immutable and provides no successful audit claim. After the report write ended, review-repair-worker-audit-stable reran the same audit with stable artifact inputs and exited 0. The recorded change in inputs justifies that single retry.

## Recommendations

Root should retain the closed slice finding separately and route the one outstanding map finding for a causal repair within effects.go and the new regression file. Keep all prior receipts/logs/stages/reports immutable, use new names for the next frozen candidate and verification, and return the repaired source for independent review before staging.

## Assessment

Ready to merge? No. Source-progress commit? No, one Important map-alias defect remains. The slice correction closes the first finding and preserves concrete-zero/nil/unknown behavior, but current conversion still rejects a proven clean map callback and loses its dirty counterpart's causal path. The source gate is NEEDS_WORK.

Stock Go1.27.1 linux/arm64 execution and linux/amd64/darwin/arm64 Load/vet metadata are developmental. Patched Go is absent. Original R8/R18/R19/task19/fn105D4/predecessors/task21, first-baseline fixed identities, full/completion/formal and both supported patched-native/affected-consumer acceptance remain open. This review supplies no formal SHIP or acceptance waiver.
