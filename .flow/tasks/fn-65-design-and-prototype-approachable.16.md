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
Added narrow owner-local checked constructors for stable family identities, value-aware Property patterns and predicates, exact Behavior sequences, and explicitly bounded Queries. Nexus2 fixtures admit and execute all three baseline operations plus guarded race cases through the existing checkers, preserve fixed identities/fingerprints, expose alternatives/providers/forms/bounds/policies, and retain typed negative diagnostics and missing-replacement analysis evidence.

The actual finite-table transition edit readmitted successfully and changed the Target fingerprint. Single-run direct elaboration measured 2.04s for admission/equivalence and 0.53s for the edit probe; repeated-run variance, scaling, editor behavior, product-owner readability, and human usability remain unmeasured. Exported admission dependencies are limited to propext, Classical.choice, and Quot.sound; no native_decide, checkedTarget native default, sorry/admit, or custom trust was added.

No commits, push, worktree, reset, revert, or cache deletion were performed; HEAD remains 7774fdc7ac751ac959816c9829516ce54af57194 and the user retains commit ownership. Go lint remains the inherited 1,316-diagnostic baseline with exact normalized SHA256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077 and zero diff from task 15.

stage: impl-review - ran [2026-09-05T23:40:00Z..2026-09-05T23:44:49Z] NEEDS_WORK..SHIP (model: codex:gpt-5.6-sol:medium; session: 01a073e2-1420-7e23-8021-5c2dd13dfd07)
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: tracker-sync - skipped(config: bridge inactive)
## Evidence
- Commits:
- Tests: baseline: (cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests Temporal.Feature.Nexus2.Tests) (pass; 68 jobs), (cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests Temporal.Feature.Nexus2.Tests Temporal.Feature.Nexus2.AuthoringTests Temporal.Feature.Nexus2.AuthoringEditProbe) (pass; 75 jobs), make lint-model (pass), make lint-code GOLANGCI_LINT_FIX=false (inherited exit 2; 1316 diagnostics; normalized SHA256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077; zero diff vs task 15), direct lake env lean Temporal.Feature.Nexus2.AuthoringTests elaboration/admission: 2.04s single local run, direct lake env lean Temporal.Feature.Nexus2.AuthoringEditProbe actual finite-transition edit: 0.53s single local run, #print axioms constructor admissions: propext, Classical.choice, Quot.sound only; QuerySpec fixture has none, impl-review codex:gpt-5.6-sol:medium: SHIP (session 01a073e2-1420-7e23-8021-5c2dd13dfd07)
- PRs: