I’ll read the task contract first, then trace the changed quick-validation paths through their callers and shared cache behavior.
The quick command selects packages across both workspaces and routes TLC cache hits through the current result validators. I’m checking whether selection covers authored-source consumers and whether cache identity includes the inputs those validators rely on.
I found a selection gap: files under `temporal/feature` are mapped only to the fixture package, but the realization crate also imports files from that directory. Those edits can therefore skip realization tests. The narrow fixture filter already handles test-file edits correctly.
- **Severity**: P2
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/dev/xtask/src/quick/mod.rs:14`
- **R-IDs**: []
- **Problem**: The mapping selects only `umpire-temporal-tests` for every edit under `temporal/feature`, overlooking other consumers. For example, `activity/probe_gaps.rs` is imported by `umpire-temporal`, whose `tests/probes.rs` asserts that `unannotated()` returns no entries. Adding an entry selects the fixture package but skips that assertion, allowing quick validation to pass. Reverse-dependency expansion cannot recover the missing consumer because it expands outward from the fixture package.
- **Suggestion**: Map shared authored sources to their actual consuming packages before expanding reverse dependencies. Preserve fixture-only selection where appropriate for Model edits.

Focused selection tests could not start because Cargo requires writes denied by the read-only sandbox.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R2 | met | Activity fixtures execute native/interpreter agreement; cache identity covers emitted inputs; hits re-enter validators; full gates force bypass. Recorded warm runs took 20.265 s and 33.084 s. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge; no blocking introduced findings.