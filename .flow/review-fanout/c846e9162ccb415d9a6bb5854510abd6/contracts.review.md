I’ll read the task specification first, then review the diff for consistency between the implementation, tests, comments, and promised behavior.
The new tests cover restriction, result ordering, duplicate and disabled-source errors, and refinement metadata removal. I’m now checking inherited setup and evidence data, along with the pins that protect Control’s existing behavior.
I haven’t found a contract mismatch so far. The Go tests cannot create their temporary build directory in this read-only sandbox. I’m checking whether the existing Lean artifacts allow the focused tests to run without writing files.
The derived-machine tests and Control tests pass when run directly with Lean. The Temporal differential is still running. I’m also comparing freshly rendered Control and Pair Cases with their committed fixtures, without writing files.
The code review is complete with no blocking findings. The remaining checks are still running. Lake’s build failed on sandbox write restrictions; the focused Lean tests that completed passed using existing dependencies.
The freshly rendered Control and Pair Cases match their committed fixtures byte for byte. The focused derived-machine and Control tests also pass. The Temporal differential has not completed yet; Go reader verification remains blocked by the sandbox.
No surviving findings.

Direct Lean runs passed for `Umpire.Command.Tests.Derived` and `Temporal.Feature.Nexus.Control.Tests`. Freshly rendered Control and Pair Cases matched their committed fixtures byte for byte.

Validation limits: Lake and Go checks encountered sandbox write restrictions. The Temporal differential was stopped before completion.

## Requirements coverage

Coverage reflects task `.4`; sibling-task requirements are deferred from this review.

| R-ID | Status | Evidence |
|---|---|---|
| R1–R5 | deferred | Assigned to sibling tasks. |
| R6 | met | Derivation rules, diagnostics, catalog ownership, and refinement removal covered by passing tests; Control contains one extension function. Caller integration belongs to `.5`. |
| R7–R8 | deferred | Composition work in sibling tasks. |
| R9 | partial | Control and Pair fixture bytes match; committed artifacts unchanged. Go/live reader verification remains incomplete. |
| R10 | met | Control Case and witness preserved; other Query implementations unchanged. |
| R11–R13 | deferred | Assigned to sibling tasks. |
| R14 | met | Control’s derived-machine Query retains its existing differential pin; no new Queries introduced. |
| R15 | deferred | Assigned to `.7`. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>