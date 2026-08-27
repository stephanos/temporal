---
satisfies: [R1, R4]
---
# fn-30-release-evidence-graph-and-manual.1 Define reusable release semantics and the fixed Temporal policy

## Description
### Umpire4 reconciliation (normative)

Release evidence policy, retention/signing, human roles, revocation, workflows, and authorization are owned by a named downstream release-policy component under the standalone canary/release boundary or an existing external release platform—not by `tools/umpire`. Umpire receipts are immutable generic inputs only. The release owner consumes retained standalone-canary evidence plus external build/deployment attestations, preserves each trust class, and acquires no semantic reinterpretation or deployment authority. Replace legacy `tools/umpire/release` paths and reusable Umpire release-policy types accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Define the pure release vocabulary, fixed Temporal policy, and the executable policy protocol for R1 and R4. Keep reusable candidate, evidence-slot, graph, decision, trust, expiry, and authorization contracts under Umpire while the exact four-profile policy remains Temporal-owned.

**Size:** M
**Files:** focused modules/tests under `model/Umpire/Evaluation/Release/`, `model/Temporal/System/Evaluation/Release.lean` and tests, fixed policy executable entry point/manifest, umbrella/lake declarations
**Touches:** [model/Umpire/Evaluation/Release.lean, model/Umpire/Evaluation/Release/**, model/Temporal/System/Evaluation/Release.lean, model/Temporal/System/Evaluation/ReleaseTests.lean, model/Temporal/System/Evaluation/ReleasePolicyMain.lean, model/Umpire.lean, model/Temporal/System.lean, model/lakefile.toml]

### Approach
- Follow the authored-versus-checked and canonical-identity seams described in `model/Umpire/ARCHITECTURE.md:31-44` and the portable-artifact boundary at `model/Umpire/ARCHITECTURE.md:210-235`.
- Put only domain-neutral closed types, checked present/gap slots, validation, canonical projections, graph limits, terminal ordering, expiry, and append-only authorization semantics in Umpire.
- Compile the exact candidate schema, four evidence profiles/versions, freshness windows, Known Gap policy, trust roles, and decision rules in the Temporal system module.
- Define canonical `ReleasePolicyInput/v2` and `ReleasePolicyOutput/v2` plus one stdin/stdout Lean executable and generated manifest binding its SHA-256 to `ReleasePolicy/v2`; it reads no files, environment, network, clock, or secrets.
- Add focused Lean tests for canonicalization, fixed-slot boundaries, rejected-over-held ordering, expiry, complete sorted reasons, protocol round trips, and import-direction purity.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ARCHITECTURE.md:31-44` — checked-value lifecycle and identity boundary
- `model/Umpire/ARCHITECTURE.md:210-260` — portable artifact contract and dependency invariant
- `model/Umpire/Artifact.lean` — existing portable artifact vocabulary
- `model/Temporal/System.lean` — Temporal system umbrella convention

**Optional** (reference as needed):
- `model/Umpire/CoreTests/Canonicalization.lean` — canonical identity test pattern

### Acceptance
- [ ] Reusable release modules build without importing Temporal-specific modules or naming Temporal, provider, workflow, environment, checker, or deployment concepts.
- [ ] The Temporal policy fixes all candidate, slot, profile, version, age, limit, Known Gap, trust-role, and terminal-decision constants from the parent spec.
- [ ] The fixed executable accepts/emits exactly one bounded canonical protocol value, binds policy/input identities, and has no ambient I/O capability.
- [ ] Lean tests cover equality/order independence plus every invalid, gap, stale, revocation, limit+1, expiry, and rejected-versus-held boundary in R1/R4.
- [ ] `cd model && mise exec -- lake build Umpire.Evaluation.Release.Tests Temporal.System.Evaluation.ReleaseTests` passes.
## Acceptance
- [ ] Reusable and Temporal-owned modules have the dependency direction required by R1.
- [ ] The first policy and its executable protocol are closed, versioned, canonical, and fully boundary-tested.
- [ ] Focused Lean builds, protocol fixtures, and purity scans pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
