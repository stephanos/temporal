I’ll read the task spec first, then review the diff against its acceptance criteria and the existing validation entrypoints.
The change records a design decision without moving runtime code. I’m checking the inventory against the current Makefile and tooling, with particular attention to preserving validation coverage during the workspace split and keeping Temporal guest builds tied to their caller’s inputs.
No surviving findings. The design covers the existing gate stages and tooling, preserves intermediate coverage across both workspaces, and assigns guest identity and migration responsibilities explicitly.

The committed verifier passed: 19 stages, 13 script-directory files, nine xtask modules and 45 root joins. `git diff --check` also passed.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R7 | deferred | Task .9’s design acceptance is satisfied by `model/design/framework-gate.md` and the recorded Decision Context. Runtime implementation and standalone proof remain assigned to .13, .10 and .11. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>