# Task 32 final corrective source review

Verdict remains NEEDS_WORK. The make-allocated slice and map reproductions are closed, but the accepted map-alias finding persists for an empty map literal. One Important conversion-provenance defect remains. Root should retain the source batch uncommitted until that constructor case is repaired and reviewed.

This is the authoritative current source review of map-repair-final.json. Earlier reports remain immutable historical evidence. BASE and HEAD are f931c9879e3017b346562667b9f6fbcc4db458ec on gomad; source changes are uncommitted. Root alone owns product source, Git/index/commits, Flow, admission, parent requirements and MILESTONES. Reviewer writes are confined to new final review artifacts and review-final-* scripts/receipts/logs.

The same reviewer context applies the requesting-code-review template and verification-before-completion skill against task32, original R8/R18/R19/task19, AGENTS Codex routing, README and MILESTONES. Requested reviewer remains gpt-6.1-sol/high, actual execution metadata unavailable, same configured Codex family as writer. Tier: session (jev-unavailable(no_key)) remains the sole routing result; no judge, bridge, download, reroute or formal review was invoked.

## Strengths

The three-line make-map correction at effects.go:788 establishes a shared declared-element abstraction before conversion. Permanent map dirty/clean tests now retain the exact record.Check -> canonicaljson.Dirty -> time.Now path or empty findings, with literal runtime callback counts 1/0 on stock Go and both supported metadata source sets. Fresh nil-interface map contents are accepted and unknown map contents remain unresolved. The prior make-slice finding and its concrete zero-method, nil-interface and unknown-interface controls remain green. Both accepted findings' make-origin reproductions are closed.

The original 35-fixture test file is an exact prefix of the current 39-fixture file. standard.go remains byte-identical to the initial candidate, preserving signature-aware Unwrap and concrete fmt Writer.Write error results. Existing argument/capture, recursion, initialization, JSON/format precedence, mutation/range, source identity and public/consumer tests retain their assertions and pass fresh checks.

## Issues

### Critical

None found.

### Important

1. Preserve first-insertion provenance for empty map literals at effects.go:750.

The conversion shallow-copies the abstractValue header and its current elements pointer. The repair initializes elements only in the builtin make branch. An empty map literal at effects.go:609 still allocates an abstract value with nil elements. Set through a converted named map writes the first element into the copied header, and the caller's map misses the callback even though the real Go maps share storage. This is the remaining constructor variant of the accepted map-alias finding.

The exact supplemental fixture changes only the allocation in the prior reproduction. It uses `v:=map[int]func(){};helper.Set(helper.Callbacks(v));v[0]()` with `type Callbacks map[int]func();func Set(v Callbacks){v[0]=Dirty}` and Dirty incrementing Calls before time.Now. The clean companion assigns `func Clean(){}` through the same alias. Both real stock modules compile and pass literal counters, dirty 1 and clean 0. Only record.Check is a pure production root and PackageEdges is empty.

Git-admitted baseline production detects record.Check -> canonicaljson.Dirty -> time.Now for dirty and accepts clean effects=[] on both linux/amd64 and darwin/arm64 metadata source sets. The current candidate reports only unresolved callback for both cases on both source sets. This loses known concrete provenance and falsely rejects a pure callback.

Immutable review-final-literal-baseline.json/log exits 0 for two cases; review-final-literal-candidate.json/log exits 1 with both named failures. review-final-map-literal-probe.py retains exact supplemental source and its hash in raw logs and uses a temporary Go source overlay. Candidate replaces only the new regression file; baseline additionally selects the original immutable effects.go/standard.go verified against Git BASE. Receipts bind exact argv/cwd/environment/timing/tools/config/log/source/protected maps, probe/runner hashes and baseline production hashes. The new audit validates exact overlay keys/substitutions and identical supplemental fixture source. No product file was changed by diagnosis.

Establish shared writable element provenance at this actual map-literal allocation before conversion, preserving the destination method set without retagging the caller or waiving unknown receivers. Add dirty/clean empty-map-literal controls in the admitted new test file and retain all existing make-origin controls. Root should route this bounded constructor repair to the same task32 writer and return it for independent review before staging.

### Minor

None found.

## Verification and evidence

Fresh whole architecture (24 top-level tests) and focused provenance/context/initialization/mutation/range (14 top-level tests) gates pass, with 39 causal fixtures and 78 supported-source metadata observations. Five required root public/consumer boundaries each actually ran once and passed. Broader root pure-effect/exact-edge/vet gates passed; host-package vet inspected darwin/arm64, linux/amd64 and actual linux/arm64 metadata. Whole gomadtool consumer (30 top-level tests), make validate, diff/gofmt and errortype passed. All reviewer commands/delegates are terminal.

Actual unfiltered pinned lint remains exit 1 with exactly four inherited diagnostics, byte-identical to baseline and writer final. They are initialization.go:123 errcheck; standard.go:222 and :304 QF1003; standard.go:390 errcheck. Introduced/resolved findings and line changes remain zero. No config, pin, suppression, dependency, runtime, public signature or classification was changed.

final-review-audit.py runs the preserved map-repair-audit only after all live artifact writes end, then independently verifies all new exact argv/env/cwd/timing/tool/config/log/source/protected bindings, actual RUN/PASS counts, exact supplemental baseline Git bytes and overlay substitutions. The writer auditor binds 167 unchanged historical files, 57 prior receipts, 11 map-repair receipts, three bounded snapshots and 1042 protected inputs. Earlier historical audits are replayed solely through their saved-source views. Prior wrong-package selections remain inconclusive; the prior disclosed artifact-concurrency audit failure and stable retry remain immutable. No earlier audit/report/evidence is overwritten.

Original test-first/final files retain the single metadata t.Logf difference disclosed in the first review. Each repair preserves the prior final regression file as an exact prefix and appends controls. Historical tests remain byte-identical.

## Recommendations

Keep the closed make-origin reproductions separate from the remaining map-literal case. Repair and verify that actual allocation origin within the admitted effects.go/new-test scope, retaining current receipts and reports under immutable names. Root should return the next frozen candidate to this reviewer before committing source progress.

## Assessment

Ready to merge? No. Source-progress commit? No, one Important map-literal alias defect remains. The make-map repair closes its exact reproduction and retains nil/unknown behavior, but the same conversion still rejects a proven clean map-literal callback and loses the dirty counterpart's concrete path. The source verdict is NEEDS_WORK.

Stock Go1.27.1 linux/arm64 and supported-platform Load/vet metadata remain developmental. Patched Go is absent. Original R8/R18/R19/task19/fn105D4/predecessors/task21, fixed first-baseline identities, full/completion/formal and both native/affected-consumer acceptance stay open. This source review supplies no formal SHIP or acceptance waiver.
