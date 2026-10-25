I’ll read the task specification first, then review the diff against its contracts, tests, and documentation.
The native build/check paths and bootstrap are in place. I’m checking whether prerequisite validation and the replacement tests preserve the promised failure behavior, and whether the joined proof matches the reviewed source.
The committed receipts show matching guest builds from two checkout paths and freshness checks against the published asset. I couldn’t rerun the focused Cargo test because the read-only sandbox blocks Cargo’s build lock. I’m finishing with static checks; the full semantic gate remains assigned to task .3 by the specification.
I found a portability defect in the replacement tests: they check `DYLD_FALLBACK_LIBRARY_PATH` through `/bin/bash` fixtures. macOS strips that variable when launching protected system binaries, so the success-path tests fail there before they can verify the native launcher.
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/dev/xtask/tests/guest_pipeline.rs:112`
- **R-IDs**: [R2, R3]
- **Problem**: The replacement loader control runs `#!/bin/bash` fixtures while requiring `DYLD_FALLBACK_LIBRARY_PATH=loader-marker`. On macOS with SIP enabled, protected system binaries purge dyld environment variables. The fake mise, previous wrapper, and compiler probe therefore discard the marker, causing guest success-path tests to fail regardless of whether the native launcher preserves it. [Apple’s runtime protection documentation](https://developer.apple.com/library/archive/documentation/Security/Conceptual/System_Integrity_Protection_Guide/RuntimeProtections/RuntimeProtections.html) confirms this behavior.
- **Suggestion**: Inject the marker immediately before launching the native wrapper and observe it through native compiler and previous-wrapper fixtures. Keep shell orchestration checks separate from the loader inheritance assertion.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Process ownership belongs to other tasks. |
| R2 | partial | Native orchestration, admission, artifact validation and joined reproduction are implemented; the macOS control above needs correction. |
| R3 | partial | Python guest controls have native replacements, but the new loader fixture breaks macOS success paths. |
| R4 | deferred | Guest contracts are documented; final gate documentation belongs to .3. |
| R5 | deferred | Final validation pipeline remains assigned to .3. |
| R6 | deferred | Process syscall migration is outside task .5. |

Unaddressed R-IDs: []

Shell syntax checks passed. The focused Cargo test could not run because the sandbox blocks Cargo’s build-lock write. The committed guest hash matches the publication receipt.

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>