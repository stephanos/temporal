Prior finding: fixed

Journal normalization now preserves schema, record counts, and all capacity limits. Both journals are required, and copying prevents baseline mutation. The negative control fails under the prior normalization and passes with the fix.

Verified both patch reconstructions and evidence hashes. Fresh retained logs confirm 20 complete CLI passes, 20 capacity controls, 40 exact-boundary kills, 120 record comparisons, and passing host/vet/format gates. No surviving findings.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R9 | met | Built CLI explore/replay and kill/resume comparisons now enforce journal capacity equality; bound final gates pass. |

Unaddressed R-IDs: []

Native Linux remains unverified. Previously recorded lint limitations and fn-112.11/.12 remain outside this repair.

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>
