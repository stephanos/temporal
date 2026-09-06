# Remove unused Umpire tooling after runtime and authoring cutovers

> HTML render lens: open local `.flow/artifacts/fn-66-remove-unused-umpire-tooling-after/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context

Fn-64 and fn-62 are complete, and fn-65's separate authoring prototype is also complete. Remove
remaining Umpire tooling that has no retained workflow or consumer. Maintainers should be able to
identify the purpose of every surviving package and command without navigating obsolete artifact
execution support. Current model authors, generator users, and Case Runtime users retain their
observable contracts; operators receive no configuration or execution change.

Fn-64's initial cutover deliberately preserved general artifact support. This follow-up determines
which parts are now unused. It does not repeat the initial deletion or depend on fn-60.

## Architecture & Data Models

Use one static ownership and deletion ledger. Freeze the post-cutover source tree and enumerate
every Umpire tooling package and command, including build, generation, regression, script,
documentation, fixture, and retained-spec consumers. A retained row names a concrete consumer;
a removal row records the consumer searches and obsolete workflow. Go import absence alone is
insufficient for command liveness. Zero unclassified or ambiguous rows is the deletion gate.

The initial removal candidates are the public artifact validation package and its CLI, followed
by the internal runtime-configuration, evidence, result, and clone support that becomes orphaned.
The internal Experiment reader remains because regression-view generation and checking consume
it. Preserve its complete symbol closure, including sealing and checksum helpers, rather than
assuming a retained filename proves compatibility. Recompute this closure after CLI retirement.

The ledger keeps the frozen baseline separate from realized deletion dispositions. Enumerate each
removed Test/Fuzz name and fixture path, its owner, generated or handwritten origin, and whether
it is replaced or intentionally retired. Aggregate counts alone do not establish coverage. Keep
fn-64's historical ledger immutable and extend its accounting in the new ledger. Additional unused
items discovered by the inventory require a concrete reviewed task adjustment before deletion;
they must not be disguised as retained without a consumer.

## API Contracts

Preserve `PrepareCase` and `PreparedCase.Run`, Case/Profile/Host/Run/Verdict values, server and worker
authority, verification, Lean Producer and authoring, API and configuration generation, regression
views, Case conformance generation, vocabulary checking, and generic checked promotion.

Retire obsolete commands without a replacement CLI or compatibility wrapper. Retained commands
keep their process exit status, stdout/stderr diagnostics, canonical bytes, identities, and
behavior. Future Case-native replay, qualification, exploration, and canary work constrains the
shared contracts it actually consumes; proposed commands are not evidence that an old adapter is
still live. Fn-5's generic promotion primitives remain, without restoring its retired
caller-closure-specific command.

## Edge Cases & Constraints

An unresolved consumer, fixture origin, or ownership row prevents deletion and inventory completion.
Every fixture must be free of surviving test, documentation, manifest, and generator references
before removal. Change an owned generator/input only if the inventory establishes that its managed
output must change; otherwise existing generation checks must prove byte-identical outputs.

Retained code and comments remain unchanged except directly necessary orphan cleanup. Do not add
validation, hardening, a registry, a scanner, runtime behavior, or a new dependency. Retained
allocation, concurrency, crash handling, security, and 10x-load behavior remain unchanged because
the change removes unreachable workflows without adding execution work.

Keep the full live-test selector and exact inherited failure-identity set. Repository lint compares
against the frozen baseline minus only diagnostic headers belonging to ledger-approved deleted
files. Record the expected subtraction and resulting digest before checking; any added or
unexplained changed/missing diagnostic fails verification. A smaller diagnostic count is not
sufficient evidence. Capture actual terminal exit codes.

## Acceptance Criteria

- **R1:** Execution starts only after fn-64 and fn-62 complete. Every remaining Umpire tooling
  package and command has a recorded retained workflow/consumer or evidence-backed removal
  decision, with a frozen baseline and zero unclassified ownership before deletion. Errors:
  unresolved consumers, fixture origins, or ambiguous ownership prevent inventory completion;
  absent Go importers alone never justify deletion.
- **R2:** Remove all items classified for removal, their exclusively owned helpers/tests/fixtures,
  and obsolete direct build, generation, script, workflow, and documentation references. Account
  individually for removed Test/Fuzz names and fixture paths against the frozen baseline. Errors:
  retained consumers, dangling active references, unaccounted tests, or unexplained fixture loss
  prevent completion; preserve historical decision artifacts.
- **R3:** Retained workflows and explicit downstream contracts remain buildable with unchanged
  behavior, diagnostics, canonical artifacts, identities, and comments. Errors: loss of generic
  promotion, Case Runtime, authoring, a required generator input/output, the complete Experiment
  reader symbol closure, or a concrete downstream contract blocks the affected deletion; do not
  add a compatibility shim or replace a real command check with a mock.
- **R4:** Focused tagged tests and complete retained model/runtime, applicable generation, model
  lint, and repository lint gates pass or report verified inherited failures. Errors: reduced
  selectors, a changed live-failure identity set, changed generated bytes without an owned-input
  reason, stale active docs, or any new/unexplained lint diagnostic prevents completion. Lint
  subtractions must match individually approved deleted-file headers exactly.

## Boundaries

- No implementation before the completed prerequisites and fresh plan review are verified.
- No repetition of fn-64 cleanup or resurrection of historical fn-14/fn-61/fn-63 workflows.
- No new runtime, authoring language, broad refactor, test consolidation, dependency, or CI system.
- No deletion outside Umpire tooling except directly owned orphaned inputs/outputs and references
  proven by the inventory. The adjacent plan-index tool and its Make target remain out of scope.
- Preserve the completed Nexus2 prototype and existing generators; no dependency on fn-60.
- Broad generated Lean API drift verification and new CI coverage remain declined.

## Decision Context

Two completed cutovers now expose the real consumer graph. A static inventory followed by an
atomic artifact/CLI retirement and a post-retirement internal trim avoids treating age or naming
as deletion evidence. Inventory analysis precedes mechanical deletion, keeping the three tasks
cohesive and reviewable. A directory-wide purge would remove the retained Experiment reader.

Keep the sole metadata dependency on completed fn-62; completed fn-64 is foundation. No retained
downstream spec depends on cleanup output, so no reverse dependency is added. The declined
[generated API drift policy](../memory/declined/generated-api-drift-verification.md) still permits
existing generator checks and removal of obsolete references, without new drift or CI machinery.

## Quick commands

Use task-specific focused commands during inventory and deletion. The retained focused baseline is:

```bash
go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/cmd/umpire-gen-regression-views ./tools/umpire/regression
```

The final task runs the complete retained gates serially:

```bash
go test -count=1 -tags test_dep ./tools/umpire/...
make umpire-build-model
make umpire-check-regression
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

Use a physical canonical TMPDIR for full regression. Its existing generated-view and conformance
checks satisfy the corresponding generation gates when they run; do not repeat them solely for
bookkeeping. Include the integration tag only for integration tests.

## Early proof point

Task fn-66-remove-unused-umpire-tooling-after.1 proves complete consumer and deletion accounting,
including the retained Experiment reader closure. If it fails, resolve ownership and revise the
affected removal tasks before deleting anything.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Complete post-cutover ownership inventory | .1, .3 | — |
| R2 | Accounted removal and reference cleanup | .2, .3 | — |
| R3 | Exact retained consumer contracts | .1, .2, .3 | — |
| R4 | Focused and complete retained gates | .2, .3 | — |

## Key files

- `tools/umpire/artifact/artifact.go` and `tools/umpire/cmd/umpire-artifact/main.go`: public validation candidates.
- `tools/umpire/internal/artifactv2/artifact.go` and `tools/umpire/internal/artifactv2/natural.go`: retained Experiment reader.
- `tools/umpire/internal/artifactv2/runtime.go`, `tools/umpire/internal/artifactv2/evidence.go`, `tools/umpire/internal/artifactv2/result.go`, and `tools/umpire/internal/artifactv2/clone.go`: internal removal candidates.
- `tools/umpire/cmd/umpire-gen-regression-views/generated_view.go` and `tools/umpire/regression/generated_view.go`: retained reader consumers.
- `model/Umpire/Property/COMPATIBILITY.md`, `.plans/UMPIRE4_SPEC_COMPS.md`, and `.plans/UMPIRE4_COMPONENTS.md`: active ownership documentation.
- `Makefile`: command wrappers and retained aggregate gates.

## References

- Fn-64's migration ledger and final regression reconciliation.
- Fn-62's completed established authoring evidence and fn-65's completed prototype boundaries.
- Retained fn-22/fn-26/fn-29/fn-33 contracts and generic fn-5 promotion.
- Umpire 4 specification and Lean Authoring Guidelines.
