I’ll read the task specification first, then review the diff for agreement between the code, tests, and documented contracts.
The reviewed range also includes the earlier core API cleanup and correlated-state fix. The removals match the plan so far. I’m tracing the consolidated Nexus header path and replacement tests for any behavior or coverage changes.
The Nexus merge still clones the cached route header before adding caller headers, preserves collision checks and byte limits, and the new empty-reply test pins the SDK’s null payload encoding. The race test could not start because the read-only sandbox blocks Go’s temporary build directory. I’m finishing static checks and checking the remaining validation commands.
No surviving findings. The removals, replacement tests, and documentation agree with the task’s contract.

`git diff --check` passed. Verification limits:

- Race tests and vet could not start: sandbox denied temporary build directories.
- Live-test gate failed during Lean build: sandbox denied artifact writes.
- Lint stopped because another golangci-lint process was running.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | State matching includes ordered fields; focused rejection and work-accounting tests added. |
| R2 | partial | Listed declarations removed and callers updated; final deadcode gate belongs to fn-94.17. |
| R3 | deferred | fn-94.7–8. |
| R4 | deferred | fn-94.6, .9–10. |
| R5 | deferred | fn-94.11–12. |
| R6 | deferred | fn-94.13. |
| R7 | deferred | fn-94.14–15. |
| R8 | partial | Focused commands attempted; environment prevented verification. Final gates remain fn-94.17. |
| R9 | deferred | fn-94.16. |
| R10 | deferred | Final measurement belongs to fn-94.17. |
| R11 | met | Existing identity goldens and fixtures unchanged by this range. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>