I’ll read the task spec first, then trace the diff and relevant callers for correctness issues within the requested scope.
The new key clears `explored` counts and recomputes both checksums. Exploration history, ledger credit, Replay digests, and promotion names use that key. I’m checking the remaining consumers and golden checks for cutover regressions.
I haven’t found a correctness defect in the key changes. The read-only sandbox is blocking the normal golden check’s writes; the existing bridge test binaries are running, and I’ll also try comparing the rendered goldens in memory.
Both existing bridge test binaries passed, including the fallback checks, and all 14 rendered goldens matched the committed files in memory. The normal Make target failed on sandbox write permissions. I’ll repeat the in-memory comparison to check local determinism.
No surviving correctness findings.

Both existing bridge test binaries passed, including fallback checks. All 14 goldens matched committed bytes in two separate in-memory runs. Fresh rebuilding was blocked by read-only sandbox permissions; CI execution was not verified.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R11 | partial | Two local comparisons passed; CI golden check added, execution unverified. |
| R18 | met | Existing golden bytes unchanged; cutover commit documents an empty flip list. |
| R19 | met | Normalized witness keys cover ledger credit and Replay; fallback tests passed. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>