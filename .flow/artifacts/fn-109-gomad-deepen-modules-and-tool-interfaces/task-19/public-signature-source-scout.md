# Public-signature source findings

Read-only `/root/public_signature_source_scout` inspected the current public
host APIs for task 19. Requested Codex thinking-scout routing was
`gpt-6.1-sol/high`; the single judge returned
`Tier: session (jev-unavailable(no_key))`. Actual execution metadata was not
independently observable. No tests, builds, generation, package loading,
mutations or native qualification resulted from this investigation.

## Confirmed retained internal identities

Paths below are relative to `tools/gomad3`.

| Public exposure | Source | Internal identity retained |
| --- | --- | --- |
| `CampaignPlanInspection.Journal` | `runner/inspect.go:78` | `campaign.ExecutionJournalPlan` |
| `CampaignPlanInspection.ArtifactCapacity` | `runner/inspect.go:79` | `campaign.ArtifactCapacityPlan` |
| `CampaignInspection.ArtifactCapacity` | `runner/inspect.go:295` | pointer to `campaign.ArtifactCapacityPlan` |
| `ExecutionJournalInspection.Limits` | `runner/inspect.go:314` | `campaign.ExecutionJournalPlan` |
| `CompatibilityPackEvidence` underlying fields | `target/capability.go:115` | governance pointer, module-evidence slice, rule-evidence slice |
| `pinimpact.Spec.Packs` | `upgrade/pinimpact/pinimpact.go:77` | callback result slice of `compatibility.ValidatedPack` |

The Runner records come from `runner/internal/campaign`. The six journal
bounds and six artifact bounds use public `record.Uint64String`, but their
outcome fields retain internal `campaign.JournalCapacityOutcome`. Merely
defining a new public type from those internal structs keeps that nested
identity. Construction is localized to `projectCampaignPlan` and
`projectCampaign`; detached public reporting values must replace direct copies
there. Preserve field order, tags, Uint64String encoding and pointer presence.
Existing Runner inspection and CLI E2E consumers still need their reports.

Target's defined `CompatibilityPackEvidence` keeps internal fields from
`compatibility.PackEvidence`: `Governance *PackGovernance`,
`Activation []ModuleEvidence`, and `Rules []PackageRuleEvidence`.
ModuleEvidence retains `Adapter *PackAdapter`; rule evidence also retains
ModuleEvidence, PackSource, PackForeignSource and LinknameEvidence slices.
Governance and adapter leaves contain strings/string slices; rule evidence
contains the named child values, strings and capability string slices.
`qualification/analysis.Report.Packs`, qualification-set WorkloadReport and
qualification-comparison Input propagate this public graph.

The projection owner is `target/capability_evaluation.go:370–375`. Introduce
public nested evidence values and explicitly project this graph there, keeping
the public outer name. Record's compact CompatibilityPack and TargetAdapter
types omit governance, activation, source/rule and full adapter data and are
not substitutes. Preserve exact tags, field order, nil/empty slices, pointer
presence, ordering and every value, with complete JSON/canonical comparisons.

Pinimpact's callback is used in production, not just test injection:
`cmd/gomadtool/compatibility_pack_refresh.go:82–87` selects the refreshed
authoring root's packs directory; `packPinImpact` supplies the callback to
Evaluate at :317. Default pin-impact and upgrade-maintenance consumers use
LoadPacks. The selected correction is public directory intent,
`Spec.PacksDirectory string`, selecting LoadPackDirectory for a nonempty path
and LoadPacks otherwise at the current delayed load point. Preserve baseline
and candidate module validation before pack loading, explicit authoring-root
selection, default environment behavior, load count and error/report handling.
Remove the internal-typed callback without removing the production override.

## Visibility and positive controls

Check every internal segment and exact parent-directory boundaries. A
Runner-internal type is unavailable even to sibling Gomad packages. A
module-root internal type is legal for same-module consumers but unavailable
to an external module such as `example.com/gomad-runner-consumer`, including
with a local replace. Public APIs must remain constructible by that external
consumer; the same-module case is a separate legal-visibility control.

Defined-type RHS syntax alone is not a violation. `choice.DiagnosticRecord`
has only built-in numeric fields after underlying-type resolution. Target's
CompatibilityIdentity has only strings; CompatibilityDisposition and
CompatibilityRemediation are string-based. Accept these. A true alias keeps
the named internal identity and is different from a defined public type with
an entirely public underlying graph.

Ordinary private storage remains encapsulated: PreparedCapabilityReview's
private preparation Inspection, CampaignSpec's private guidance/resume state,
and methods on unreachable private executor receivers are not public exposure.
Walk public embeddings/promoted members and reachable private same-package
named types, such as Descriptor's private adapter type with exported string
fields. Do not classify those solely by an unexported type name.

Fixtures must cover direct/pointer fields, callback/slice results, aliases,
defined builtin-only versus retained internal nested graphs, embeddings and
generic arguments/constraints. The inspected host APIs had no exported aliases
or generics; those fixtures exercise future checker coverage. The existing
external Runner fixture constructs request seams, not inspection or pack
evidence; extend its compile-positive with construction of the repaired public
values and pinimpact directory intent.

## Corrective scope

Task 19 owns the checker and these confirmed signature repairs. Its corrective
Touches include Runner inspection, target capability reports, pinimpact and
the pack-refresh consumer; the intended Go changes must be inventoried before
implementation in go-interface-changes.md. Public projections do not move the
private campaign or compatibility policy owners. Establish old-source checker
RED and positive public construction, fixed-identity report/JSON and error
precedence preservation. None of this source investigation closes R8 or D4.

Writing-for-agents groups each defect with its correction and completion
evidence so the checker cannot gain a broad exception for the existing leak.
