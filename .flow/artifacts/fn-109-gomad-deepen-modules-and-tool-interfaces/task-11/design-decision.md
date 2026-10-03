# Task 11 (R17/S5): capability collection, pure evaluation, linked projection

Base revision: `5d093b214d` (branch `gomad-fn109`).

## Decision

Public types stay in `target`, because `TestPublicPackagesDoNotExportTypeAliases` forbids
moving them behind aliases. The three concerns have separate private owners:

- **Collection** (`target/capability_collection.go`) owns every host effect of a review. It
  runs the bounded `go list`, reads the build overlay and the package sources, resolves and
  digests adapter replacements, runs the `go list std` check for exec provenance, and loads
  the compatibility packs (`loadCompatibilityPolicy`: `compatibility.LoadPacks`, which reads
  `GOMAD3_COMPATIBILITY_PACKS`, plus the host platform). Before this change pack loading
  happened inside `compatibility.Select`, in the middle of evaluation.
- **Evaluation** is pure. `target/internal/capabilitypolicy` owns the policy decisions: pack
  selection through `compatibility.SelectPacksForPlatform`, forbidden imports, foreign
  sources (headers exempt), linkname facts, missing Go source, and the exact first-party
  simulation bridge pins, which moved byte-for-byte. `target/capability_evaluation.go`
  validates the evidence shape, converts packages for the policy, and projects its findings
  into the public review. It reuses the existing compatibility policy and adds no evaluator
  or registry.
- **Linked projection** (`target/capability_linked.go`) reads the embedded record
  (`readLinkedCapabilityRecord`, used by `ReadCapabilityManifest`, exec preparation and
  `finishGoTarget`) and narrows a closure review to live, guarded and eliminated findings
  through `livecap.ProjectFindings`.

Callers no longer sequence the steps themselves. `prepareExec`, `validateProvenance` and the
tests call `reviewRecordedClosure`. That function checks the closure identity, then loads the
policy, then evaluates. This is the old order: schema check, `Select` (which loaded the
packs), structure checks, findings. Fresh reviews go through `projectCapabilityReview`
(collect packages, load policy, evaluate), which keeps the old order of source errors before
pack errors. Both pack loading and selection failures keep the
`select target compatibility packs:` prefix.

Inventory: `internal/sourceinventory` (architectural owner `sourceinventory`, may import only
`hostfs`) owns `Digest`. `target` and `deterministicio` import it and map its
`CapacityError` to their own `AdapterCapacityError`. The bounded regular-file reader that
`target` and the inventory both need moved from `target.readBoundedRegularFile` to
`hostfs.ReadBounded`, with the same error text.

A fresh review evaluates its collected packages once and records the selected packs. A
recorded closure is evaluated once against the packs it names.

## Enforcement

- `TestCapabilityEvaluationHasNoHostEffect`: `capabilitypolicy` imports only `slices`,
  `sort`, `strings` and `internal/compatibilitypack`, and calls only `SelectPacksForPlatform`
  from it. `capability_evaluation.go` may import only an allowlist (`fmt` only for `Errorf`,
  `filepath` only for `Base`, `record` only for `ParseSHA256`, and only the listed
  compatibility data types plus `DigestSources`). It references no package-level `target`
  function declared in another file. The check covers calls, conversions and function values.
  A mutation check confirmed that it rejects `loadCompatibilityPolicy` or `filepath.Join`
  taken as values in the evaluator, and `compatibility.LoadPacks` in `capabilitypolicy`.
- `TestExactModuleEdges`: `target` and `deterministicio` import `internal/sourceinventory`,
  that package imports only `internal/hostfs` from the module, and `target` no longer exports
  `DigestAdapterSourceInventory`.
- `TestEvaluateUsesOnlyTheSuppliedPolicy`: evaluation ignores a broken
  `GOMAD3_COMPATIBILITY_PACKS`.

## Equivalence

- `TestCapabilityReviewGoldenCanonicalBytes` checks six goldens in
  `target/testdata/capability-review/`, captured from the base implementation before any
  edit. They cover ordered closure findings, sources (overlay, header, assembly, linkname,
  test variant, generated test main, module and local replacements), a nested adapter
  replacement with its inventories, linked and guarded projections with live, guarded and
  eliminated blockers and denied boundaries, and a recorded closure. They are byte-identical
  after the change.
- The inventory encoding is pinned by `TestDigestPinsInventoryEncoding`. Its digest was
  computed with the base `target.DigestAdapterSourceInventory`.
- Locally only (scratch, not committed): canonical `ReviewCapabilities` output for real
  targets is byte-identical before and after on the same toolchain. The targets are the
  simulation fixture, the `gomad3sim` test variant, and linked and guarded reviews of
  dead/live `os/exec` and `os.Readlink`.

## Pre-existing, left unchanged

`TestBuiltInSimulationLinknamesPinCurrentFirstPartySources` and
`TestClosureReviewSupportsSimulationFixtureAndRefusesHarnessTests` fail on the base revision.
`tools/gomad3sim/runtime_time_toolchain.go` gained a `gomadSimulationTimeCurrent` directive
in commit `ad90b462e0` ("wip") without a pin update. The acceptance criteria require the pins
to stay unchanged, so this task does not repin. The owner of that runtime change must update
the pin.
