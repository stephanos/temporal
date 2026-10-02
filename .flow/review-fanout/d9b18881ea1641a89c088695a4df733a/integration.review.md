I’ll read the task spec first, then trace the changed code through its callers and consumers, focusing on integration issues introduced by this diff.
The new key clears `explored` and recomputes both nested Plan checksums. I’m tracing how Case production, promotion, and replay recovery use that key, and checking whether CI covers the new goldens.
Both existing bridge test executables passed, including the new fallback checks. The Go consumers treat candidate digests as opaque identifiers, and Case production reads the checked witness rather than the Plan checksum; I haven’t found a broken consumer assumption. The golden check is still running, with sandbox-related cache errors so far.
No blocking integration findings.

Both prebuilt bridge test executables passed, including fallback checks. The pinned Plan and receipt agree on counts and witness steps. Fresh golden verification was blocked by the read-only sandbox.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R11 | partial | Plan/receipt pins and CI check added; repeated local and CI execution evidence unverified. |
| R18 | partial | Existing golden files remain unchanged; task’s flip-list evidence is absent. |
| R19 | met | Shared normalized key reaches ledger, replay, and promotion; fallback checks pass. |

The outstanding execution records and flip-list documentation are nonblocking process observations.

Unaddressed R-IDs: []
Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>