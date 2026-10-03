## Review: fn-109 task 12 — separate detached `Artifact` from owned `Opened` handle

I could not execute the Quick commands in this session (no shell available), so test results are taken from the diff's evidence record, not from a run I observed. Everything below is from reading the diff and the current repository state.

**Summary of what the change does.** `artifact.Artifact` loses its `root *os.Root` and becomes a pure detached reference. A new `*artifact.Opened` owns the pinned root with a private manifest. `OpenPayload`, `ReadPayload`, `CopyPayload` and `TargetSharing` become methods on `*Opened`, with their bodies and error texts unchanged. `Manifest()` and `Snapshot()` return deep copies via a reflective `cloneManifest`. All 26 production and test consumers are migrated. I confirmed via grep that no `.Detached()`, free-function payload accessors, or field-style `opened.Manifest.X` usages remain anywhere under `tools/`, and that `record.ExecutionRecord` contains only exported fields and no interface, func, or chan kinds, so the reflective copy cannot hit its panic branch or an unexported-field `Set` panic. The `TestCloneManifestSharesNoMemory` test populates every field by type walk and would catch either regression if the record type grows one.

**Correctness checks that pass:**
- Validation is untouched: `validateDirectory`, `openValidatedFile`, `readValidatedFile`, `hashValidatedFile` are unchanged and still run inside the methods. Nothing moved to callers.
- Store's validated-reuse path now uses `existing.Snapshot()` plus `sharingOf`; `sharingOf` only reads `Path` and `Manifest`, so behavior is identical to the prior field-by-field copy.
- `deepCopy` preserves nil-vs-empty for slices and maps, which matters for canonical JSON (`build_tags: []` vs `null`).
- `preflight` previously leaked the root on validation failure because the deferred close saw the zeroed named result. Now the handle is a local and the close error joins the validation error. This is a real fix, correctly documented in `go-interface-changes.md`.
- `acceptedInput` now passes a copy into publication rather than the live manifest. Same bytes, no aliasing.
- Test coverage matches every clause of R13 and the task's acceptance list: detached reference has no unexported fields and no `Close`, published vs opened equality, open/double-close/nil-close, use after close for all four payload methods, snapshot and `Manifest()` mutation isolation, pinned-directory read after real-directory replacement, and the full rejection matrix (unlisted, over bound, mode, size, hash, symlink, escaping directory).

### Findings

**P3** · Confidence 100 · introduced · `tools/gomad3/artifact/open.go:292` · R-IDs: [R13]
**Problem:** `ReadPayload` on a nil `*Opened` reports `artifact payload "x" is not listed`, while `OpenPayload`, `CopyPayload` and `TargetSharing` on nil report `artifact is not open`. The test at `opened_test.go:114` carries a comment explaining the asymmetry. The author preserved this intentionally to match the old zero-`Artifact` behavior, so it is a judgment call, not a bug, but the four methods on the same handle now disagree on what a nil receiver means.
**Suggestion:** Either add an `opened == nil` guard at the top of `ReadPayload` returning `artifact is not open` (closed-but-non-nil handles keep "not listed" first, so documented error precedence for real handles is unchanged), or leave as is. Not blocking.

**P3** · Confidence 100 · introduced · `tools/gomad3/artifact/publication_test.go:43`, `tools/gomad3/runner/watchdog_replay_test.go:149`, and similar test sites · R-IDs: []
**Problem:** Each `opened.Manifest()` call performs a full reflective deep copy. Several migrated test conditions call it four or five times in one expression. Production call sites each bind it to a local once, so this is test-only noise, but it sets a pattern that would be costly if copied into a hot loop.
**Suggestion:** Bind `manifest := opened.Manifest()` once per test block, as the production migrations already do.

**FYI (not findings):**
- `TestOpenedArtifactReadsPinnedDirectoryAfterReplacement` overlaps with the existing `TestOpenedArtifactRemainsPinnedAcrossPathReplacement` in `store_test.go`. The new one adds manifest-identity assertions and a real replacement artifact, so the overlap is defensible.
- The byte-identical publication claim in `go-interface-changes.md` rests on an uncommitted scratch test. The publication code path is unchanged apart from the reuse-path snapshot, and existing store tests cover identities, so this is a process note only.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R13 | met | Distinct `Artifact` (exported fields only, no `Close`) and `*Opened` (private manifest, owns root). `TestArtifactReferenceHoldsNoOpenResource`, `TestPublishedReferenceMatchesOpenedHandle`, `TestClosedArtifactRejectsPayloadAccess`, `TestManifestCopiesCannotChangeOpenedHandle`, `TestCloneManifestSharesNoMemory`, `TestOpenedArtifactReadsPinnedDirectoryAfterReplacement`, `TestOpenedArtifactRejectsChangedPayloads` cover every clause. All consumers migrated; `go-interface-changes.md` records declarations, consumers and migrations. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>

VERDICT=SHIP

## Disposition (implementer)

- P3 nil `ReadPayload`: fixed in the follow-up commit; every payload method on a nil handle now fails with `artifact is not open`. A closed non-nil handle keeps the original check order.
- P3 repeated `Manifest()` in tests: not changed. The migrated test expressions keep their original shape to keep the diff mechanical; the copies are of KB-sized manifests in tests only, and production sites bind one copy.
