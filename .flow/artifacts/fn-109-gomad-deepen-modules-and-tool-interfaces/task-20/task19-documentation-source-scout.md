# Task 19 documentation reconciliation scout

Task 20 still owes concrete Go caller migration and architecture-check ownership
guidance. The current World boundary paragraphs already describe the corrective
design. This is bounded preparation against moving task-19 source, with no
implementation admission, formal review verdict, test result or native claim.

Read task 20 through `flowctl cat` after `flowctl usage`, the whole ARCHITECTURE,
SPEC, README, CLI and TUTORIAL, the interface inventory, World decision, and
the three prior task-20 scouts. Requested Codex thinking scout is gpt-6.1-sol
at high. Tier remains session (jev-unavailable(no_key)), using the existing
one-time judgment. Actual model and effort metadata are unobservable.

## Remaining guidance

- Public reports. Add the actual replacement names to the intentional Go
  migrations and ARCHITECTURE's inspection/capability ownership discussion.
  `runner/inspect_capacity.go:9,20` defines
  `ExecutionJournalLimitsInspection` and `ArtifactCapacityInspection`.
  `CampaignPlanInspection.Journal` and `.ArtifactCapacity`,
  `ExecutionJournalInspection.Limits`, and `CampaignInspection.ArtifactCapacity`
  now use those public values (`runner/inspect.go:78,295,314`). Callers that
  construct these fields use the public types and string outcome fields.
  The private campaign owner retains the operational plans and explicit
  projections (`inspect_capacity.go:31,35`); inspection does not own campaign
  transitions. `target/capability.go:117` retains `CompatibilityPackEvidence`
  and exposes its complete nested graph through `CompatibilityPackGovernance`,
  `CompatibilityModuleEvidence`, `CompatibilityPackAdapter`,
  `CompatibilityPackageRuleEvidence`, `CompatibilityPackSource`,
  `CompatibilityPackForeignSource`, and `CompatibilityLinknameEvidence`
  (`:126,135,143,154,164,169,175`). Construct those target-owned nested values
  instead of internal policy evidence. `capability_evaluation.go:370,414`
  explicitly projects and copies nested storage, preserving nil/empty and
  pointer distinctions in source. Final preservation evidence belongs to
  task 19. Relevant existing IDs are EVIDENCE.INSPECTION and TARGET.CAPABILITY.

- Pack-directory intent. Document `pinimpact.Spec.Packs` replacement as
  `PacksDirectory string` (`upgrade/pinimpact/pinimpact.go:68,75,165`). An empty
  field retains normal embedded/environment selection; an explicit field
  supplies the directory containing pack files. Refresh passes
  `filepath.Join(compatibilityRoot, "packs")` (`compatibility_pack_refresh.go:83,313`).
  Evaluation loads packs after baseline/candidate module validation and keeps
  load failures in pin evaluation (`pinimpact.go:153,165,170`). The source
  comment at `:75` calls this an authoring root, but
  `internal/compatibilitypack/v2_selection.go:173` and the actual refresh caller
  identify its `packs/` child. Avoid carrying that terminology conflict into
  caller guidance. Existing CLI grammar needs no new flag. Use
  MAINTENANCE.DEPENDENCY and MAINTENANCE.COMPATIBILITY.

- World migration detail. Keep the current boundary paragraphs and add a small
  caller example or precise contract pointer. External wrappers, joins and
  custom callback errors passed directly to `Recorder.FinishError` must be
  normalized outside World, then supplied as
  `world.Terminal{Kind: world.TerminalCapacity, Detail: detail}` or the existing
  replay-divergence/invalid-input kind to `FinishTerminal`. Detail must be
  nonempty; empty, inferred and quiescence kinds are rejected. `Finish()`
  remains the inference operation (`world/recording.go:74,78,91,144`). Original
  sentinels, nonnil `*CapacityError`/`*ReplayDivergenceError`, and private
  model-generated errors retain the closed convenience path; typed-nil inputs
  reject without callbacks (`world/errors.go:54`). Direct rejection leaves
  recording active and returns the fixed unsupported message, without the old
  callback-derived detail/wrapped identity. General reporting remains
  `world/process.Session.FinishError`, whose validated path captures detail
  before capacity, replay, then invalid-input classification and descriptor
  cleanup (`session.go:106,114,122,137`). Public sentinel rebinding cannot
  redefine the model's immutable identities. Use WORLD.MODEL,
  WORLD.LIFECYCLE and WORLD.REPLAY without a new requirement or terminal format.

- Architecture gates. ARCHITECTURE's Maintenance gates (`:747`) omits the new
  checker owner, inventory coverage and failure interpretation. Cite
  `internal/gomadtool/architecture`, which inventories module source/nested
  modules, reports owner/module edges and stale exclusions, and parses/types
  the selected platform source (`architecture.go:47`, `program.go:34`,
  `edges.go:9,47,76`). Public-signature checking traverses nested public type
  graphs, generic arguments/constraints and promoted methods against the
  external consumer's Go internal-import access (`program.go:138,253`). The
  existing test name `TestPublicPackagesDoNotExportTypeAliases` does not mean
  every alias or builtin-only defined type is forbidden (`architecture_test.go:208`).
  Purity roots are explicit (`effects.go:300`); reachable callbacks and host
  effects belong outside those roots, and unresolved bindings/changed pinned
  standard-library summaries become findings (`standard.go:60,331,396`).
  Explain that both-platform source analysis and host vet protect structure;
  runtime/platform qualification remains a separate gate
  (`architecture_test.go:224,228,244`). Recheck final checker names, root list
  and summary maintenance against the task-19 writer's finished source.
  Relevant IDs are MAINTENANCE.GOVERNANCE and VERIFICATION.TRACEABILITY.

## Already correct and existing discrepancies

ARCHITECTURE:532-541 and README:1237-1245 already distinguish closed model
completion from effectful process reporting, reject direct custom/external
wrappers, and explain detached terminals and immutable sentinel behavior.
SPEC:304-324 stays at product-model ownership; it need not acquire Go type names.
CLI:578-580 and README:1137-1139 already state refresh's authoring-root override;
TUTORIAL's World/backend discussion remains compatible. None promises arbitrary
direct recorder callback admission.

The migration inventory's task-19 section (`go-interface-changes.md:63-114`)
still says planned and requests final names. Reconcile it after source freezes.
Its current preservation obligations are requirements, not observed passing
evidence. Retain the earlier source and final-owner scouts for tasks 2-18.

The earlier forward-clock discrepancy also appears in CLI:147 and
TUTORIAL:259-266, which still describe a separate offset, versus README:790-809
describing the shared clock. The previous source scout already assigns its
reconciliation to D26's current-source/evidence owner. This scout adds no scope
or acceptance claim for that candidate.

Keep capability support (SPEC:202-208), same-seed repeatability
(README:277-280), verified exact replay (README:285-294), and expectation
matching (README:312-342) distinct. Preserve D12, absent exact-replay capacity
and host-clock escapes as open dispositions. Preserve D14's recorded Darwin
correction and historical source-bound evidence, using
`disposition-reconciliation.md`; that evidence does not qualify this integrated
candidate. No new performance or platform-support claim follows from a
structural checker or these source reads.

## Hash record and limits

SHA-256 snapshots were taken before detailed source inspection and after the
reads. Those two samples agreed. A final post-write check then detected active
writer drift in `effects.go`, from
`61042a6274d4490e526595fb3c0345d1f026d9a762695f7ab3a9b7c8ae791010`
through `fb422e09875065f89e87615eaf27b8a7648c18ec49b47e0e763437a7b8e243cc`,
then to `a2f1e685a7956b2eec2f6d30abe49e1b00288fc232a6cfc59fd43e4b53bb2cac`
at 2026-10-04T05:35:28Z.
The effects citations describe the read snapshot and need recapture at task
20's final freeze. Every other table hash is the same at its before/after
observations. The effects hashes above record that file's observed drift.
Additional loader and prior-scout cross-checks were separately double-hashed.
No sample freezes task 19 or establishes absence of intervening edits. Paths are
relative to `tools/gomad3/` unless prefixed `artifact/`; that prefix denotes
`.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.

```text
before = after SHA-256                                             path
190b53122346600db1f3e21d94f1011cde962d2e3d3322a00599c18a751db9f5 ARCHITECTURE.md
2176df8403a3a1dbf156b9da8fdedab979b50971691c20a6dcea6024dfe97719 SPEC.md
ffe5d52fd9ded62cf1d2a4c0755b9d166b369b377481cd8d2d65185dd7f74887 README.md
e65d94ff604c6a40a5ef660bcae00a13543a3a358387264a2a4009bee91e3965 CLI.md
99c2b8fbbc3402dffb8c1cf8442794a43ae92bed1cfc93fb996300d858de5e53 TUTORIAL.md
731851e8e3fda15a39734df3b0470aa3200f565d4743287b2098093ebfc34822 runner/inspect.go
8b3b142aee8b998576890c455a9a21ebb2b5a2043eed2c44fdd1934f38a534a6 runner/inspect_capacity.go
383dab9197c5b7fa635ea91e6a3cb5865a0a0d5fb922faf203c6296f369616ef target/capability.go
59210779488349d34c9c9be16568afc59f780a22076a06e49a68709c841f33a5 target/capability_evaluation.go
5cc677ae91963c82939841124ada35231af975b53ec0ce2ab3aa8241e967293f upgrade/pinimpact/pinimpact.go
cb09108ac1b0617ecf2f2251b825d8854cebf269edcae160371fef1a436aef3c cmd/gomadtool/compatibility_pack_refresh.go
7c80bfa7cce53d14967b8aff784461e855e70c066af32607fb6ca03a3da65f59 internal/compatibilitypack/v2_selection.go
fc00c9af42ebb2cb74fa593b9c026205a2df19bf4634e99a532f697c7fb4fc27 world/recording.go
12b8cd52d3cff41d2ccb17e3a2793ec0c74d5c6facb83b7ee407d473a6cb876e world/errors.go
c472d6d7a5640136daa6382905f72022018654fe588d7bcbea513025ff10601c world/process/session.go
cbed01e8d9a38953cbb7841bc32d2a9d5f815c83fabdac496a1afeef36ab0c2d architecture_test.go
68734d4d0967e8bf3fe33fab2c9256b678d0f431d42c6de14012526bf278a929 internal/gomadtool/architecture/architecture.go
a3f3dcd71de31609621095e45f485638b40a5225146940ee9b4fca5773748fc7 internal/gomadtool/architecture/program.go
e5a3446578b356423ae36d9f24209854058930d2e7bef3df1ac3a8135e839845 internal/gomadtool/architecture/edges.go
03bd277d232c397c1d618b2e38ed2c55606c5b00de8eea167657f9e5c802a02b internal/gomadtool/architecture/standard.go
79906c5ded6369b8b5cde4b15ecfbb5a1b6af86c613ea45a99431b45a220fd05 internal/gomadtool/architecture/memory_sources.go
1d3ec221a0d65346442cdfd25254acd9a376c27945e7bab28fea24e6dba17eb1 artifact/go-interface-changes.md
1f29877e4c45004f71aa15d651c1172872d1a6506183e24f9d5375bef24e00e2 artifact/task-19/world-terminal-design-decision.md
4f639255143c7def2dba7bf3d6a6de30a33d2dbda2256b59c369772f03506920 artifact/task-20/source-scout.md
0798083477608f385758ddf828320f65dcfa293ecae441652bc36542d311ee0a artifact/task-20/final-owner-source-scout.md
e1e0475343e5f646ebbead58a6d0a963c445798ce5ffc9bb5cac1d5fe918ab28 artifact/task-20/disposition-reconciliation.md
```

Only this uniquely owned artifact was written. No shared source/docs edits,
tests, builds, package loading, generation, Flow lifecycle changes or Git
mutations were performed.
