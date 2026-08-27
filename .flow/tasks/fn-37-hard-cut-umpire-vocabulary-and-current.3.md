---
satisfies: [R2, R3]
---
# fn-37-hard-cut-umpire-vocabulary-and-current.3 Rewire model languages to fingerprints, Limits, and Known Gaps

## Description
Complete R2 and R3 across checked authoring and Planning. Replace arbitrary digest strings with fingerprints derived from checked behavior, make executable Target behavior fingerprintable, and replace bounds/omissions with exact Limit and Known Gap types. Task `.5` owns the v2 Artifact envelope; this task regenerates the canonical Target/Query goldens whose behavior references change.

**Size:** L
**Files:** `model/Umpire/{Target,Property,Behavior,Query,Planning}/**`, `model/Umpire/Core.lean`, current Artifact adapters, Temporal model/config consumers, Target compatibility fixtures, Switch/Nexus query fixtures
**Touches:** [model/Umpire/Core.lean, model/Umpire/Target/**, model/Umpire/Property/**, model/Umpire/Behavior/**, model/Umpire/Query/**, model/Umpire/Planning/**, model/Umpire/Artifact.lean, model/Umpire/Examples/Fixtures/SwitchExactActionQuery.json, model/Temporal/**/*.lean, model/Temporal/Feature/Nexus/Fixtures/OperationsAsyncStartQuery.json, model/Temporal/Feature/Nexus/Fixtures/OperationsCancellationQuery.json, model/Temporal/Feature/Nexus/Fixtures/OperationsSuccessfulCompletionQuery.json]

### Approach
- Replace every behavior-bearing `semanticDigest`, `contractDigest`, and matching helper with typed `behaviorFingerprint` derivation in the owning checked language.
- Add a required finite `TargetBehaviorDomain`, separate from `FinitePlanningAvailability`, with complete Setup, State, Action, Outcome, and Observation domains plus canonical encoders, domain-coverage proofs, and transition-closure proofs.
- Mechanically construct the canonical `TargetBehaviorDescription` by evaluating the checked kernel's `initialStates` and `steps` over the complete domains. Include its sorted initial-state and transition rows in the Target fingerprint. Existing soundness/completeness proofs establish agreement with `authoritativeInitial` and `authoritativeStep`.
- Keep fingerprint coverage when `FinitePlanningAvailability` is `.unavailable`: unavailable means “cannot Plan,” not “hash metadata.” Reject Target authoring with a typed diagnostic when behavior-domain coverage is absent or incomplete.
- Replace `LawRequirement.semanticDigest` and the free `DefinitionId -> Prop` pairing with a typed `LawDefinition` whose canonical inert body is interpreted into the proposition proved by `LawWitness`. Include canonical law bodies in Capability Contract and Target fingerprints.
- Define each other owner module's smallest canonical behavior view; exclude source locations and documentation, and remove arbitrary author-supplied fingerprint labels.
- Rename `BoundUnit` and `TypedBound` to `LimitUnit` and `Limit`; rename bound exhaustion/status/diagnostic fields to Limit Reached vocabulary without changing completeness claims.
- Introduce exactly `KnownGap { kind, code, subject, detail }`. `kind` is the closed enum `capability-contract | input | interpretation | claim`; `code` is a namespaced Definition ID; `subject` is an optional Definition ID; `detail` is optional text. Encode fields in that order, sort rows by `(kind, code, subject-or-empty, detail-or-empty)`, reject exact duplicates, and reject conflicting detail for the same `(kind, code, subject)`.
- Replace context-specific `semanticIdentity` uses: Definition ID ordering becomes `definitionId`; Artifact identity remains for Task `.5`.
- Regenerate Target compatibility fixtures plus `SwitchExactActionQuery.json` and all three `Operations*Query.json` files from their existing authoritative Lean tests/producers.
- Add mutation tests proving transition results and interpreted law bodies change Target fingerprints while source/documentation and declaration order do not.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:105-165` — executable kernel, law requirements, and witnesses.
- `model/Umpire/Target/Language.lean:1-65,180-320,541-610` — Target availability, current metadata-only canonical view, and composition.
- `model/Umpire/Target/Tests/Canonicalization.lean` — current fingerprint mutations to strengthen with executable transition/law-body mutations.
- `model/Umpire/Property/Language.lean` — Property canonical behavior view.
- `model/Umpire/Behavior/Language.lean` — Behavior canonical view and occurrences.
- `model/Umpire/Query/Language.lean` — Query fingerprint, tie-break, and Limits.
- `model/Umpire/Planning/Types.lean` — planner statuses, counts, current bounds, and omission strings.
- `model/Umpire/Artifact.lean:30-110` — temporary source adapter that must keep compiling before v2.
- `model/Umpire/Examples/SwitchTests.lean` and `model/Temporal/Feature/Nexus/OperationsTests.lean` — authoritative query golden producers.

### Key context
A Behavior Fingerprint is generated from behavior, not an author-facing version label. The prototype deliberately requires finite canonical Target behavior coverage. A Target that cannot prove that coverage is rejected; it is never assigned a metadata-only fingerprint. Limit Reached remains distinct from Exhaustive Search.
## Acceptance
- [ ] Checked Target, Property, Behavior, and Query values expose typed Behavior Fingerprints generated by their owner modules; Planning propagates those typed values.
- [ ] Target fingerprints include mechanically enumerated initial-state/transition rows and interpreted law bodies, with proofs covering the complete finite Target Behavior Domain and authoritative kernel relations.
- [ ] A transition-result mutation or law-body mutation changes the Target fingerprint without metadata edits; source path, documentation, and declaration-order mutations do not.
- [ ] `FinitePlanningAvailability.unavailable` Targets retain the same fingerprint coverage, while missing/incomplete behavior-domain coverage fails Target checking with a typed diagnostic.
- [ ] Limits replace bounds; Limit Reached and Exhaustive Search remain independently tested outcomes.
- [ ] Known Gap is exactly the closed `{kind, code, subject, detail}` record; invalid kinds, malformed IDs, duplicate rows, conflicting descriptions, and noncanonical order reject.
- [ ] Target compatibility fixtures, `SwitchExactActionQuery.json`, and all three `Operations*Query.json` fixtures are regenerated from authoritative Lean producers.
- [ ] No public `semanticDigest`, `semanticIdentity` tie-break, `TypedBound`, `BoundUnit`, omission string, or arbitrary fingerprint label remains in the migrated scope.
## Done summary
Rewired checked Target, Property, Behavior, Query, and Planning around owner-generated typed Behavior Fingerprints, executable Target behavior domains, Limits, and exact closed Known Gaps. Target encoders and finite Planning domains now fail closed on collisions or duplicates; Query-only candidate-evaluation limits cannot enter Property, structural ordering is delimiter-safe, and all authoritative Target/Query/artifact/regression fixtures were regenerated from their producers.

Baseline: green via existing gate receipts at 667e66c4. Verification: combined Lean Quick green (131 jobs), pinned Go green, and current regression target green (137 jobs). Artifact-v1 semantic identity and omission wire adapters remain intentionally owned by task .5. Review memory capture was skipped because repository memory is configured but not initialized.

stage: impl-review - ran [2026-08-27T16:21:53Z..2026-08-27T16:43:42Z] (NEEDS_WORK → SHIP)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 95d3a5f8db84820ca6e416ed66d696d2d0d231df, d4dc228e1dd3245d0078d2221f1b75416c68c29c
- Tests: cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect, mise exec -- go test ./tools/umpire/..., mise exec -- make umpire-check-regression
- PRs:
