---
satisfies: [R2, R3, R5, R7]
---
# fn-65-design-and-prototype-approachable.16 Expose checked constructor authoring across Target, Property, Behavior and Query

## Description
Implements R2, R3, R5, R7; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Umpire/Target/**`, `model/Umpire/Property/**`, `model/Umpire/Behavior/**`, `model/Umpire/Query/**`, `model/Temporal/Feature/Nexus2/**`
**Touches:** [model/Umpire/Target.lean, model/Umpire/Target/**, model/Umpire/Property.lean, model/Umpire/Property/**, model/Umpire/Behavior.lean, model/Umpire/Behavior/**, model/Umpire/Query.lean, model/Umpire/Query/**, model/Temporal/Feature/Nexus2/**]

### Approach
Build narrow constructors in the existing language owners over the admitted finite and guarded semantics. Reuse typed error results and capture types at Target/Language.lean:123; no catch-all new authoring framework. This is the constructor comparison specimen and successful-branch admission baseline; the next task measures source-aware compile-time frontends.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property.lean` — frozen public facade
- `model/Umpire/Property/Check.lean` — checked extraction boundary
- `model/Umpire/Behavior/Language.lean` — authored/checked Behavior
- `model/Umpire/Query/Language.lean` — Query forms/limits
- `model/Umpire/Target/Language.lean` — source capture and native default hazard
- `model/Temporal/Feature/Nexus2/DESIGN.md` — authored interface comparison

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests Temporal.Feature.Nexus2.Tests Temporal.Feature.Nexus2.AuthoringTests Temporal.Feature.Nexus2.AuthoringEditProbe)
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

### Measured task evidence
Direct `lake env lean` elaboration of the executable constructor admission/equivalence fixture took 2.04s in one local run. After adding one typed finite-table transition and readmitting it through `checkModelTarget`, direct elaboration of the isolated edit probe took 0.53s in one local run. These are single-run local observations without cache deletion; repeated-run variance, 10x scaling, editor completion/hover/navigation/recovery, product-owner readability, and human usability remain unmeasured.

Baseline only existing roots before creation; after implementation include the new roots named below. Run focused commands during iteration, and the parent final gates at prototype completion. Use the Makefile LEAN_LAKE platform wrapper if direct Lake invocation cannot find the macOS SDK. Preserve comments and existing unrelated changes. No commits unless the user requests them.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.
## Acceptance
- [ ] Constructor specimens express all three baseline operations, race and guarded cases without encoded field/reference/payload pairing, repeated raw/check/extraction proofs or feature planner transport. Explicit roots/kinds/keys, alternatives, providers, Query forms, named stage/unit bounds and policies remain inspectable.
- [ ] Public checked declarations are available only from successful existing checker results. Demonstrate actual admission in an executable build/test gate; raw Lean record elaboration does not count. Invalid references, duplicate/malformed IDs, missing capabilities, Target mismatch, contradictory constraints and invalid/missing units retain language-owned typed error/related-ID information.
- [ ] Preserve parent/case/exception/clause identities and explicit local Action occurrence keys. Rename/move Lean declarations or docs without semantic identity drift; change behavior to change its fingerprint. No implicit instance/source-order choice of provider, outcome or bounds.
- [ ] Compile constructor comparison fixtures and assert checked semantic equivalence to prior Nexus2 declarations with fixed IDs/inputs; include unsupported guard operators and missing replacement semantics. Keep one default example definition and separate alternative specimens.
- [ ] Audit exported checked values and their proof dependencies, measure admission/elaboration and one ordinary state/transition edit. No hidden native_decide fallback; report unsuccessful or unmeasured ergonomics honestly.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: