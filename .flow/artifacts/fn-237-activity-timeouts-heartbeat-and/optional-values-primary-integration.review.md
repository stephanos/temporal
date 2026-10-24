I’ll read the task specification first, then trace the changed code’s interactions with callers and backends.

The spec defines three distinct boundaries: native factories validate configuration, source binding validates selected values and starts, and typed decoding rejects invalid backend codes. I’m checking that the new nominal types preserve those boundaries through binding and replay.

The diff includes controls for malformed codes, correlated starts, eager fallbacks, selected World swaps, and native/TLC replay. Cargo cannot rerun them here because its build lock is on a read-only filesystem. I’m continuing the code review and checking for usable cached test binaries.

The cached focused binaries passed 31 tests covering source admission, binding, decoding, native accessors, eager evaluation, and priming. I haven’t found a blocking integration issue so far. I’m finishing the preserved-artifact check and requirements coverage assessment.

No surviving findings in the reviewed range.

Verification: 31 cached focused tests passed; all 827 preserved artifacts matched their manifest. Fresh Cargo execution was blocked by the read-only build-lock filesystem.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Nominal binding, exact decoding, eager fallback, countdown agreement and replay controls; existing facts preserved. |
| R2 | deferred | Optional-value prerequisite delivered here; activity behavior belongs to task .3. |
| R3 | deferred | Compatibility refinement belongs to task .3. |
| R4 | deferred | Live admission and cancellation evidence belong to task .4. |
| R5 | deferred | DESIGN.md additions belong to task .5. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>