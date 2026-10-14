I’ll read the task specification first, then review the diff and its integration with the existing CI and checker commands.
The documentation matches the current backend selection, exact-state deduplication, and replay gate. The only executable change adds descriptor generation to both CI jobs; I’m checking its Make dependencies and the focused workflow test. The full cold-build timing remains explicitly pending.
No blocking findings introduced by this changeset. The CI step invokes the existing descriptor generator before either job builds Lean, and the workflow test pins both placements. The documentation agrees with the existing backend implementation.

Verification was limited by the read-only sandbox: focused Go tests and the plan-index check could not create temporary build directories; the inventory check consequently did not run. The descriptor-generation dry run passed.

## Requirements coverage

Task-scoped requirements, including the carried CI obligation:

| R-ID | Status | Evidence |
|---|---|---|
| R13 | deferred | Defer-only branch does not apply: R22 selected adoption. |
| R20 | partial | Both jobs now generate the missing descriptor; complete cold-build timing remains pending in `UMPIRE4_ORDER.md`. |
| R21 | partial | `UMPIRE4_ORDER.md` records the successful 23-file rollback drill, but the task evidence lacks its detailed diff stat. This paperwork gap is nonblocking under the supplied rubric. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>