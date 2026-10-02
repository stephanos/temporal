I’ll review the exact task-8 patch, trace the final tests, and verify the retained evidence bindings without modifying files or rebuilding the toolchain.

The final tests assert CLI exit statuses, stop the identified coordinator, verify the journal count is exactly 2 or 0, and wait for owned descendants to exit before resuming. I’m checking the comparison exclusions against the record types and independently hashing the evidence and frozen sources.

All 111 artifact hashes, three frozen sources, and 60 protected inputs match. The final log contains 20 complete passes, 40 exact-boundary kills, and 120 record comparisons; the host gate passed with the explicit stock compiler. I found a comparison gap: clearing the entire journal also removes its non-wall capacity limits, which the task contract requires to match. I’m checking whether another assertion covers those limits.

- **Severity:** P2
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `tools/gomad3/cmd/gomad/e2e_test.go:357`
- **R-IDs:** [R9]
- **Problem:** `left.Journal, right.Journal = nil, nil` excludes the journal’s non-wall capacity limits along with its permitted storage differences. These limits are not compared elsewhere. Changing only `Journal.Limits.MaximumBytes` leaves the comparison equal; independent `inspect` validation checks each store’s consistency, not equality between campaigns. This misses the explicit requirement to preserve non-wall resource bounds.
- **Suggestion:** Assert both journals exist and compare their `Limits` before excluding journal storage metadata.

Independently verified the exact patch reconstruction, 111 artifact bindings, three frozen sources, 60 protected inputs, 12 retained command statuses, and 21 payload hashes. Retained final logs show 20 complete passes, 40 exact-boundary kills, 120 record comparisons, and a passing host gate with the explicit stock compiler. No files were modified or toolchains rebuilt.

The documented linux verification gap, root-lint limitations, and separately owned fn-112.11/.12 issues do not affect this verdict.

## Requirements coverage

Coverage is limited to task 8’s assigned requirement.

| R-ID | Status | Evidence |
|---|---|---|
| R9 | partial | Built CLI explore/replay, exact-count coordinator kills, cleanup, resume, manifest comparisons, and storage validation are covered. Journal capacity-limit equality remains unchecked. |

Unaddressed R-IDs: [R9]

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":["R9"]}
```

<verdict>NEEDS_WORK</verdict>