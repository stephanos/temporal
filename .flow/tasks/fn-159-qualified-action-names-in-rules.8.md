---
satisfies: [R15]
---
# fn-159-qualified-action-names-in-rules.8 Rename capability kinds Retries → Retryable, Deadline → Expirable

## Description
Rename the two capability kinds whose names break the adjective convention: `Retries` becomes `Retryable`, and `Deadline` becomes `Expirable`. The new names are a proposal; confirm them with the owner before regenerating. Every other capability names what an entity *can be* (`Closable`, `Pausable`, `Pollable`, `Terminable`, `Describable`), and feature-local facets will use nouns for what happens (`Pausing`, `Cancellation`). With the renames, every capability reads as an adjective and can't be confused with a facet.

**Size:** S–M
**Files:** `model/temporal/capabilities/{Retries,Deadline}.scala` and their tests (renamed to match), every model's `capabilities` section and capability overrides that name them, `model/README.md`, `model/SEMANTICS.md`, regenerated `model/ir/*` and `model/cases/*`, and the Go files that spell the qualified names (12 under `tools/umpire` mention `Retries` or `Deadline`; check each)
**Touches:** [model/temporal/capabilities/Retries.scala, model/temporal/capabilities/Retries.test.scala, model/temporal/capabilities/Deadline.scala, model/temporal/capabilities/Deadline.test.scala, model/temporal/capabilities/Properties.test.scala, model/temporal/features/**/*.scala, model/README.md, model/SEMANTICS.md, model/ir/*, model/cases/*, tools/umpire/**]

### Approach
- **Rename only the capability kinds:** the `final case class` and its `CapabilityKind` companion, their files, and their uses in `capabilities` sections and `overriding(...)` / `bound(...)` statements (e.g. `Retries.…`, `Deadline.deadlineTimesOut`). Do **not** rename the other things called "deadline": the activity's `deadline` timer object (`deadline.heartbeat`), timer and input names, `Timeout`/`TimeoutType`, the nexus close-policy designs `*WithDeadline`, or per-machine binding names like `startToCloseDeadline`. Decide with the owner whether those binding names should follow; leave them by default.
- **Comments and docs:** update the README and SEMANTICS text that names the kinds, and the comments that call their Properties "Retries' cancellation law", "Deadline's companion laws" and so on.
- **IR:** the generated capability Property ids change: `temporal.capabilities.Deadline.firesInWindow`, `….deadlineTimesOut` and the `Retries` equivalents. Regenerate and check that the diff contains only these qualified names (plus source paths, since the files are renamed). Rows, outcomes and Query answers must not change.
- **Go:** grep `tools/umpire` (and `common/testing/testpilot`, `tests/`) for `capabilities.Retries` / `capabilities.Deadline` and update any test or fixture that spells them.

### Prerequisite
- A working tree that builds the lifter and passes the Cases check (the in-progress IR `Duration` migration, plus `make protoc`).

### Quick commands
```sh
make umpire-gen-model
make umpire-check-model
make umpire-check-lint
go test ./tools/umpire/...
```

## Acceptance
- [ ] The capability kinds are named `Retryable` and `Expirable` (or the names the owner confirmed), in files named after them; no reference to the old kind names remains in models, docs, comments or Go.
- [ ] Other "deadline" names (timer objects, inputs, `*WithDeadline` designs, binding names unless the owner decided otherwise) are unchanged.
- [ ] Regenerated IR and Cases differ only in the capability Property qualified names and source paths; `make umpire-check-model`, `make umpire-check-lint` and `go test ./tools/umpire/...` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
