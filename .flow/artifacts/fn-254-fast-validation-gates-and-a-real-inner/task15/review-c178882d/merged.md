Coordinator format repair and duplicate collapse only. Correctness and contracts each returned raw SHIP. Integration was interrupted by the600-second outer command limit and has no verdict; none is invented. The supported finalizer recovers the two atomically published draw results. These reports remain authoritative. Task closure still requires fixes of demonstrated regressions and a fresh integration-focused review.

## Issue 1
- **Severity**: P2
- **Title**: Interrupted inner-loop timing can leave source content modified
- **Confidence**: 75
- **Classification**: introduced
- **Problem**: The native timer appends a comment to tracked source and restores bytes through normal return or Drop. A supervisor signal can terminate the owned child before Drop, leaving the checkout changed; the deleted shell only touched the mtime. A // comment also assumes Rust syntax for the documented arbitrary selected source.
- **Suggestion**: Restore the original mtime-only edit using File::set_modified, or make content restoration survive termination if real content mutation is required.
- **File:Line**: model/dev/xtask/src/gate/timing.rs:115
- **R-IDs**: []

## Issue 2
- **Severity**: P2
- **Title**: Empty framework target directory writes guest outputs into source
- **Confidence**: 75
- **Classification**: introduced
- **Problem**: CARGO_TARGET_DIR set to the empty string becomes the caller directory in the framework build-guests leaf. The replaced shell treated empty as unset; the Temporal port already restores that behavior. Framework guest and native report outputs can therefore enter the source tree.
- **Suggestion**: Filter empty target values before applying the default target, and cover the empty-value vector.
- **File:Line**: model/umpire/dev/xtask/src/build_guests.rs:18
- **R-IDs**: []

## Issue 3
- **Severity**: P3
- **Title**: Shared protobuf error prescribes the wrong provisioning command for a consumer
- **Confidence**: 100
- **Classification**: introduced
- **Problem**: Missing generators always suggest cargo xtask proto-tools, which provisions the framework plugin root. The system selector uses the repository plugin root, provisioned by its own Make prerequisites, so the suggested command cannot fix that refusal.
- **Suggestion**: Use a plugin-root-specific generic hint or let the consumer provide the correct remedy without adding system names to framework code.
- **File:Line**: model/umpire/dev/xtask/src/proto.rs:85
- **R-IDs**: []

## Issue 4
- **Severity**: P3
- **Title**: Native control identities execute both in control stages and workspace tests
- **Confidence**: 75
- **Classification**: introduced
- **Problem**: The non-ignored Rust controls are selected by named control stages and again by the workspace tests stage. This repeats compiler/Make fixture work and may affect the epic's gate-time goal.
- **Suggestion**: Keep the current complete inventory and assess measured cost in task8 before changing selection; any later change must retain every maintained control and honest counts.
- **File:Line**: model/umpire/dev/xtask/src/gate.rs:159
- **R-IDs**: []

```json
{"classification_counts":{"introduced":4,"pre_existing":0},"unaddressed":[],"suppressed_count":{"50":1,"25":3}}
```

<verdict>SHIP</verdict>
