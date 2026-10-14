---
satisfies: [R1, R2, R3, R4, R7]
---
# fn-167-gaps-declared-in-the-model-gap-replaces.2 Resolve gap subjects, owners, waivers and expiry references in Scala

## Description
Resolve annotations into internal lifter metadata (groundwork for R1–R4 and R7). No new protobuf fields or sidecar output are introduced here.

**Size:** M
**Files:** gap collection, lifter context/integration, capabilities integration, gap fixtures and docs.
**Touches:** [model/irgen/Gaps.scala, model/irgen/Context.scala, model/irgen/Lifting.scala, model/irgen/Lift.scala, model/irgen/Capabilities.scala, model/irgen/test/Gaps.test.scala, model/irgen/testdata/gaps/**, model/README.md, model/SEMANTICS.md]

### Approach
- Reuse existing symbol identities, positions, facet consumer ownership and capability waiver resolution. Keep annotation placement separate from matched finding owner; normalize each constructor to the current kind's exact subject representation.
- Resolve object/val annotations once and preserve annotation/subject positions. Detect duplicate full finding keys after action-class expansion and machine attribution, including across realizations that share a machine owner.
- Collect `except`/`overriding` metadata alongside gaps, preserving existing waiver emission and the gate copy path until task 5.
- Resolve expiry existence via an injected read-only spec registry: exact canonical id, then unique short id. Cover missing, ambiguous and malformed references. Do not introduce requirement-level completion semantics or a public Flow configuration surface.
- Integrate collection/validation into the normal Scala lift. Existing ProtoJSON remains the transport; the collected value is private implementation data for task 5, not a parallel IR.

### Investigation targets
Required:
- `model/irgen/Context.scala:297` — positions and owner helpers.
- `model/irgen/Capabilities.scala:377` — resolved waiver identities.
- `model/irgen/Lift.scala:129` — current Model assembly and waiver output.
- `tools/umpire/lint/accept.go:88` — exact acceptance key.
- `tools/umpire/lint/holes.go:711` — concrete hole subjects.
- `tools/umpire/lint/kinds.go:149` — evidence aggregation.

### Verification
Extend GapsSuite with each supported placement, shared facets, realization/machine owner differences, Property versus Query subjects, defaults, duplicate positions and invalid typed references. Pin unchanged semantic protobuf output modulo existing position normalization and existing waiver output on no-gap and gap fixtures; record full wire/inertness proof as still deferred. Run the complete lifter suite at this task boundary with frozen sources.

## Acceptance
- [ ] Canonical owner/subject resolution and duplicate-position controls pass for all supported placements.
- [ ] Expiry existence/refusal fixtures and waiver collection pass without changing current wire/copy behavior.
- [ ] Complete lifter suite passes; full transport-dependent R2–R4/R7 proof remains explicitly open.

## Done summary
Resolved typed `@gap` annotations into internal lifter metadata (`model/irgen/Gaps.scala`, `ResolvedGap`), integrated into every normal Scala lift after the Model is assembled, so nothing it reads changes the written IR. Existing ProtoJSON and `<file>.waivers.json` remain the only outputs; the collection is private data for task 5 (`Lifter.gaps`), not a parallel IR.

baseline: green — `umpire.irgen.GapsSuite` 2/2 before edits (baseline.log); selected mini compile/lift at base (before/lift, before-lift.log).
stage: impl-review - skipped(config: REVIEW_MODE=none; owner gates independent review once at Scala batch end)

Requirement evidence (groundwork only):
- R1/R2 owner/subject: GapResolutionSuite pins the exact (kind, owner, subject, exact) set for every supported placement: machine table kinds; two facets attributed to the one consuming machine; facet Property witnessOnly/unasked; verify Query unfired (Property's machine) vs find Query unrealized (Scenario's machine); refinement unread (`plain refines worker`); composition stuck; realization uncovered/unreachableBinding owned by the realization vs unperformed/unevidenced/unreadObservation owned by its machine. Whole-action shorthand expands to concrete class keys (Go `interp.Classes` spelling), phases canonicalised to catalog order (`move-fast in busy, done`), `until` default empty, reasons/positions/subject positions retained.
- R2 duplicates: duplicate full key refused naming both positions, within one machine (whole action + one of its classes) and across two realizations sharing a machine owner.
- R1 typed references resolving to nothing: unbound action, phase of another type, missing phases on a phased machine, kind not owned by the placement (uncovered on a machine, witnessOnly on a Query, unrealized on a verify Query), outcome/fact type mismatch, value held by no state field, realized kind with no lifted realization; plus IR-file run refusal of a gap on a declaration no IR file lifts.
- R7 expiry: injected read-only `SpecRegistry` (default reads `.flow/specs` ids); exact canonical id, else unique short id; missing, ambiguous and malformed refusals pinned. Requirement suffix is a reference only. No completion semantics or public config.
- R4 waivers: `except`/`overriding` collected beside gaps as synthesized `capability-waiver` entries with reason and position; existing waivers JSON/copy path unchanged (pinned).
- R3 (partial): fixture IR and waivers byte-identical to the same fixture with every `@gap` blanked (lines kept). Found and fixed a real inertness defect: TASTy spans of annotated definitions start at the annotation, so a gap moved its declaration's IR position; `Index.startPastGaps` now passes over leading `@gap`s only (other annotations unchanged).

Verification:
- `umpire.irgen.GapResolutionSuite` 4/4 and `umpire.irgen.GapsSuite` 2/2 (gaps-final.log, exit 0).
- Selected mini compile/lift (selected.sh): `activity-mini.json` and `activity-mini.waivers.json` byte-identical before/after (mini-cmp.log).
- Complete lifter suite with frozen sources (irgen-full.log, exit 1) vs the same suite at the base commit in a clean worktree (irgen-full-base.log): 115 vs 112 passing. Every remaining failure is inherited/environmental: real-feature lints (Dispatching.scala:99), stale model-scala.jar mini TASTy vs current grouping lint (Product.scala:40), ConventionsFixtures/real standalone. The only new names were (a) suites that hit initializationError in the worktree (no .flow/tmp artifacts) and fail per-test here on the same Product.scala:40 lint, (b) four ConventionsFixtures tests failing on a stale ignored scala-cli cache in testdata/conventions (re-run with the cache moved aside reproduces the base set exactly, conventions-fresh-cache.log; cache restored), (c) one GapResolutionSuite assertion broken by scalafmt reflowing the fixture, fixed and re-run green.
- scalafmt --check on changed files; git diff --check.

Failures retained: resolution-1..11.log (fixture setup: nested root names, Fact import shadowing, layout lint, Closed phase, unnamed Limits/Scenario), resolution-6 shows the inertness defect before the position fix (line 71 vs 86).

Open obligations (held, task 5/6): serialize gaps/waiver metadata through the shared IR and exclude from fingerprint/equivalence/table pins (full R3 wire proof); `unmapped` and `explicitWait` carry the typed reference identity (`exact = false`) and must be bound to the lint's `<field> <VALUE>` / `<script>/<command>` spelling with reader knowledge; Go matching must compare phase lists in catalog order (or as sets) since Go lists them in cell order; Flow status adapter for done-spec expiry (R7 lint half); whole-action expansion may name classes with no finding (stale under exact matching) — consumer policy for task 6.

Deferred gates: Go/proto/schema, regeneration, lint acceptance, equivalence over real features, broad Scala validation and independent review (batch end).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7c2250291bd356867f4a22087b40a8331e01fae4
- Tests: timeout 1800 mise exec -- scala-cli test model/irgen --server=false --test-only 'umpire.irgen.Gap*' (6/6; gaps-final.log), /bin/bash .flow/tmp/fn167/task2/selected.sh before|after; cmp activity-mini.json and activity-mini.waivers.json (identical; mini-cmp.log), timeout 5400 mise exec -- scala-cli test model/irgen --server=false (exit 1, inherited; irgen-full.log vs base worktree irgen-full-base.log), timeout 1800 mise exec -- scala-cli test model/irgen --server=false --test-only 'umpire.irgen.ConventionsFixtures' with stale testdata cache moved aside (base failure set; conventions-fresh-cache.log), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check <changed files>, git diff --check
- PRs: