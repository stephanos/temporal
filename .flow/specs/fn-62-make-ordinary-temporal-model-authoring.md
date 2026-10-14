# Complete ordinary Temporal model authoring after the Nexus2 prototype

> HTML render lens: open local `.flow/artifacts/fn-62-make-ordinary-temporal-model-authoring/spec.html` — regenerable; markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context

Complete only the ordinary-authoring requirements left uncovered by fn-65. The intended author is a Temporal engineer with basic Lean knowledge. The resulting established Nexus journey must expose semantic choices while hiding repeated structural assembly and representation transport. Operators and end users receive no intended runtime or configuration change.

Fn-65 completed all 19 tasks and passed whole-spec completion review. Its evidence comparison establishes R3 as covered, R1/R2/R4/R5/R8/R9 as partial, and R6/R7 as uncovered. This residual plan replaces the seven never-executed task descriptions; task `.2` is repurposed for established Lifecycle migration, and `.8` separates Known Gap propagation from checked authoring. Implementation requires a fresh plan review. Completed fn-58 and fn-65 remain provenance dependencies.

## Architecture & Data Models

Deepen existing public owners and consume the constructor interfaces delivered by fn-65. Ordinary production finite Targets retain author-supplied domains, encoders, enumerators, closure evidence, and Action-executability evidence through `FiniteMachine`. The separate Nexus2 finite-table experiment retains its recorded prototype boundary.

Property, Behavior, Query, and Observation remain separate languages with their existing raw checkers and explicit proof-taking checked constructors. Temporal owns its family-root specialization; generic Umpire code contains no Temporal names. Existing checked planner-kernel derivation is reused unchanged.

Move the existing checked Known Gap vocabulary below Query without changing its data or validation contract. Queries carry an explicit default-empty authored set. Checked composition combines authored and phase-owned sets before artifact publication and makes the exact rows available to Case compilation. Gaps remain non-behavioral data.

## API Contracts

- `FiniteMachine.targetDefinition(machine, id, source, definitions, requiredCapabilities)` returns the existing TargetDefinition, deriving only its setup/kernel fields from the machine. `FiniteMachine.authoredTarget(machine, id, source, definitions, requiredCapabilities, composition, occurrences)` returns AuthoredTarget by adding the existing authoredPlanning capability. The existing FiniteMachine record remains the sole author-evidence input, including all five domains/encoders, both enumerators and all eight closure/executability proofs. The constructors remove hand-built TargetDefinition and dependent planning transport; they do not derive author evidence. Final `checkTarget` remains separate.
- A Temporal-owned family helper fixes the `temporal` root and explicit kind while authors provide stable family/suffix components. It returns existing identity contracts; raw IDs retain syntax-only validation until the owning language checks references.
- Reuse `DefinitionFamily`, `QueryLimitSpec`, `PropertySpec`, `ExactSequenceSpec`, `QuerySpec`, and transition-result constructors. Existing `.checked` seams retain explicit checker-success evidence. No wrapper inserts native proof evaluation or infers outcomes, providers, references, or Limits.
- Observation constructors produce existing inert profiles, rules, mappings, dispositions, ordering, closures, and Evidence bounds from explicit typed fields. `checkObservation` remains authoritative, and checked extraction retains its proof argument.
- `KnownGap` and `KnownGapSet` keep their current row schema and error precedence, with compatible public re-exports. The authored Query set defaults to checked empty and is excluded from behavior fingerprints. An author may declare no gaps; truthful omission is not inferable by a checker.
- Authored/phase composition preserves every row field and canonical order, includes exact overlaps once, and reports conflicting details through an explicit `KnownGapError` result before publication. It never substitutes empty data, drops a row, or silently omits an artifact. Default-empty authoring preserves existing bytes.
- `composePlanningKnownGaps(query) : Except KnownGapError KnownGapSet` unions phase and authored gaps. `plan(query, kernel) : Except KnownGapError PlannerRun` performs that composition before traversal, even when search would select no artifact. Private finishing consumes the already checked union and stays total; `artifactOfSelection(query, trace, reason, explored) : Except KnownGapError ExperimentSpec` uses the same composition contract for direct callers. Query evaluation failures remain `PlanningOutcome.invalid QueryError` inside a successful PlannerRun, distinct from the outer metadata error.
- `PlanningRequestError` has exactly `knownGap KnownGapError` and `artifactIntent ArtifactIntentError` cases. `planWithArtifactIntent` returns `Except PlanningRequestError PlannerRun`, preserving intent-first precedence. Space and Promotion retain their existing outer result types and add a `knownGapCheckFailed` error kind carrying the complete underlying gap error. Finite kernel derivation, bounded candidate traversal and case analysis signatures remain unchanged.
- Every Query-preserving derivation copies `authoredKnownGaps`, including constructor declaration lowering, Space lowering and Promotion rechecking. Record updates preserve it automatically, with regression coverage; a default field value is not a substitute for copying attachments from a base Query.
- Expose an exact checked conversion to the existing Case compiler input. Runtime Run/Verdict propagation is already owned by the Case Runtime and is not redesigned here.

## Edge Cases & Constraints

Preserve established Nexus public imports, explicit providers, IDs, Behavior Fingerprints, metadata, deterministic traces/plans, artifact bytes, warning/trust inventory, and diagnostic precedence. Avoid source relocation during migrations. Any intentional source correction or newly authored gap must name its exact provenance/gap-bearing byte and checksum delta; neither changes Behavior Fingerprints or modeled outcomes.

Missing finite proofs fail elaboration before checking. Invalid raw declarations continue to fail at their existing typed boundary, retaining offending and related IDs and relevant source coordinates. Unsatisfiable Behavior and Limit Reached retain their responsible status and cannot establish success. Missing, ambiguous, conflicting, unsupported, or causally unrelated Evidence remains fail-closed.

Helpers remain pure deterministic Lean construction, with no registry, callback, runtime I/O, new dependency, global instance selection, or stronger asymptotic traversal at ten times declaration volume. Preserve existing comments. No `sorry`, `admit`, custom axiom, or additional compiler-trust dependency may enter load-bearing declarations; audit transitive trust against the explicitly approved boundary, not merely existing syntax. Preserve explicit author proof arguments even where kernel synthesis is difficult.

Complexity evidence is a structural audit of added work, not elapsed-time assertions: finite Target and identity/Observation/Query constructors add only record assembly or one pass over their explicit input, with no nested rescan or repeated normalization/checking. Known Gap extraction retains the existing set algorithms; composition calls existing checked union once per planning request (or direct artifact-construction request), without rechecking it per row or search candidate. Tasks `.1`, `.3`, `.5`, `.6`, `.8` record called functions, input sizes and traversal counts; `.7` checks those inventories together. For 10× independent declarations, wrapper-only work must scale by at most the same linear factor, while any unchanged baseline checker/set complexity is disclosed separately. No cached or unequal-work timing substitutes for this audit.

## Approach

1. Deepen the proof-carrying finite assembly interface and establish failed-construction evidence.
2. Specialize Temporal identities and migrate the established Lifecycle using explicit author evidence.
3. Migrate all three established Nexus operations through existing constructors and planner derivation.
4. Add typed Observation composition and migrate the established Observation declaration.
5. Separate Known Gap vocabulary ownership and checked Query attachment from checked downstream composition.
6. Publish the compiled established Nexus reader path and verify exact compatibility, public imports, trust, and final gates.

## Acceptance Criteria

- **R1:** A compiled public-facade walkthrough completes the established Nexus Target, Property, Behavior, Query, plan, and Observation journey, including authored Known Gaps, with explicit semantic choices and no internal, Experimental, runtime, or verification imports. Errors: representative malformed ID/reference, missing finite proof, invalid raw Target/transition, and invalid Observation specimens execute at their established elaboration or typed-checker boundary.
- **R2:** The ordinary proof-carrying finite interface removes repeated structural assembly while retaining author-supplied ordered domains, encoders, enumerators, closure/executability proofs, explicit providers/connectors, metadata, and `checkTarget`; established Lifecycle uses it. Errors: actual missing-proof constructions fail elaboration; colliding encodings, invalid raw incomplete kernels, missing capabilities, and unresolved competing providers retain typed diagnostics.
- **R4:** Established declarations use Temporal-rooted explicit-kind identities, stable author suffixes and source locations, and existing named per-stage Query Limits. Errors: helper callers cannot substitute a foreign root; malformed resulting/raw IDs, duplicate/crossed references, and invalid/zero/wrong-unit Limits retain owning-checker errors. Source order and instance search choose no identity or behavior.
- **R5:** The three established Nexus operations use existing Property/Behavior/Query and transition-result constructors with materially less repeated assembly, unchanged checker authority, explicit success evidence, and Target-owned outcomes. Errors: invalid clauses, missing capabilities, unsatisfiable Behavior, Target mismatch, invalid Limits, and omitted proof arguments remain visible; no helper introduces hidden native evaluation or a second production language.
- **R6:** Typed readable Observation profile/rule/mapping construction and the migrated Nexus Observation retain explicit field identities, dispositions, ordering, closures, provider reconciliation, sources, and Evidence bounds. Errors: missing/unknown or wrongly typed fields, duplicate mappings/dispositions, absent dispositions, provider conflicts, invalid ordering/closure, over-limit Evidence, and missing/ambiguous/conflicting Evidence retain fail-closed results and diagnostic precedence.
- **R7:** Optional checked model-owned Known Gaps compose deterministically with phase gaps through checked Queries, planning, and artifacts and are available as exact downstream Case rows without affecting behavior. Empty authored sets preserve existing bytes; exact overlap appears once. Errors: malformed code/subject, duplicates, conflicting detail, and noncanonical external order retain `KnownGapError`; unknown wire categories reject; cross-set conflict is visible before publication. Gaps cannot establish success, silently disappear, or imply that omitted limitations were detected.
- **R8:** Migrated established Lifecycle, operations, and Observation preserve public imports, provider selection, IDs, fingerprints, metadata, selected traces/plans, artifacts, and failure precedence. Only specifically recorded source-provenance or authored-gap deltas are allowed. Errors: unexplained identity, byte, ordering, outcome, trust, warning, or diagnostic drift blocks completion.
- **R9:** Public documentation and a concise compiled quickstart cover ordinary versus expert finite authoring, raw/check/checked values, stable IDs, explicit composition, Target-owned outcomes, typed Limits, Observation, Known Gaps, and established reader order. Nexus2 remains clearly experimental. Focused and aggregate builds, regression, import checks, axiom audit, and model lint pass. Errors: stale/unchecked examples, facade leaks, new unapproved trust, placeholders, new warnings, or new lint diagnostics block completion; only verified inherited output may use the repository's existing baseline policy.

## Quick commands

```bash
cd model && mise exec -- lake build Umpire.TargetTests Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests Umpire.Observation.Tests Umpire.Planning.Tests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.ObservationTests
```

The final task runs aggregate model roots, `make umpire-build-model`, `make umpire-check-regression`, `make lint-model`, and `make lint-code GOLANGCI_LINT_FIX=false` serially. Go tests use `-tags test_dep`; full regression uses a physical canonical temporary directory. Focused task gates precede the one final broad gate sequence.

## Boundaries

No new authoring DSL, generic proof synthesizer, production adoption of prototype syntax or derived author evidence, redesign of the expert TransitionKernel path, Experimental authoring redesign, System integration, Evidence collection, runtime/Go/CLI behavior, credentials, configuration, deployment, or claim-assessment changes. No new global registry, required-gap inference, or extra Known Gap vocabulary. Broad generated API drift verification and new CI workflows remain declined; existing focused regeneration and compatibility checks remain applicable.

## Decision Context

The user explicitly deferred fn-62, prototyped fn-65, and requested retaining only uncovered requirements. R3 is therefore removed from this residual acceptance inventory without renumbering later IDs: fn-65 provides `IncrementalPlannerKernel.ofCheckedQuery` and executable identity/completeness failures. Migration consumes that capability rather than rebuilding it.

R2/R4/R5 retain only the author-evidence, Temporal-family, and established-migration contracts not proved by the separate prototype. Fn-65's successful frontend `Except` values do not constitute automatically kernel-checked constants. Its narrow AUT-07/AUT-08 exceptions remain bounded to the experiment; this plan uses existing proof-taking constructors and ordinary production rules rather than broadening those exceptions.

Known Gap extraction is behavior-neutral ownership work; authored attachment and conflict-reporting composition are explicit new semantics. Keeping those contracts separate avoids hiding validation changes in a move. Preserve source-shaped rows, default-empty bytes, complete source-linked diagnostics, and exact migration artifacts as established by project memory. Do not use unequal-work timing or cached 10x fixtures as scaling evidence.

## Early proof point

Task `.1` demonstrates smaller finite assembly while preserving explicit proof failure and checkTarget admission. Task `.2` then proves exact established Lifecycle migration. If either fails to reduce ceremony without hiding author inputs or changing meaning, revisit that constructor seam before operation migration.

## Requirement coverage

| Req | Residual contract | Tasks |
| --- | --- | --- |
| R1 | Compiled established journey through Observation and gaps | `.2`, `.4`, `.5`, `.7` |
| R2 | Explicit finite author evidence and Lifecycle | `.1`, `.2` |
| R4 | Temporal identities, sources, named Limits | `.3`, `.4` |
| R5 | Existing constructor operation migration | `.4` |
| R6 | Observation helpers and migration | `.5` |
| R7 | Checked authored gaps and deterministic propagation | `.6`, `.8` |
| R8 | Exact established compatibility | `.2`, `.4`, `.5`, `.8`, `.7` |
| R9 | Public guide, trust, imports, final gates | `.7` |

## References

Completed fn-58 and fn-65; fn-65's Nexus2 evidence inventory and completion review; Umpire AUT-01–AUT-08, SEM-04–SEM-09, MOD-06–MOD-08, PLN-01, ART-01/ART-02/ART-04/ART-09–ART-12, EVD-04/EVD-05; Lean Authoring Guidelines; project memory on behavior-neutral refactors, source-shaped schemas, exact artifacts, nested diagnostics, and fair authoring comparisons.
