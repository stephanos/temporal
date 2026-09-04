# Remove unused Umpire tooling after runtime and authoring cutovers

## Goal & Context

After fn-64 and fn-62 are complete, remove remaining Umpire tooling that has no retained
workflow or consumer. Maintainers should be able to identify the purpose of every remaining
package and command without navigating abandoned runtime generations or obsolete authoring glue.

Fn-64.8 owns the initial legacy execution cutover; fn-64.10 owns its documentation and regression
reconciliation. This follow-up covers residual unused tooling across the remaining surface,
including general artifact and older-generation support that those tasks deliberately preserve.
It is independent of fn-60 serialization cleanup and fn-65 authoring experimentation.

## Architecture & Data Models

Inventory the post-cutover package and command surface against the Case Runtime, Lean Producer,
ordinary model authoring, generation, regression, and retained downstream workflows. Record each
package and command as retained with a concrete owner/consumer or removed with evidence that its
workflow is obsolete. Inspect repository callers, build and generation references, scripts,
workflow commands, fixtures, documentation, and retained Flow specs; Go imports alone do not
establish command liveness.

Reuse fn-64.8 migration evidence and extend its accounting for newly removed tests and fixtures.
The inventory supports deletion decisions; it does not introduce a runtime registry or framework.

## API Contracts

Preserve retained Case preparation/execution, Host, verification, Producer, generation, and
promotion contracts. Obsolete commands and packages are removed with their direct references;
no compatibility wrapper or replacement command is required for a retired workflow. Retained
workflows keep their observable errors, canonical bytes, identities, and behavior.

## Edge Cases & Constraints

A package without internal callers can still serve a CLI or retained downstream consumer.
Ambiguous ownership blocks deletion of that item until resolved. A retained downstream need must
name the spec and concrete contract; hypothetical future reuse is insufficient justification.
Older Umpire generations and general artifact helpers are eligible only where their remaining
consumers are also obsolete; names or age alone are not deletion evidence.

Preserve existing comments on retained or moved code. Regenerate managed output through its owner
if a deletion changes generator inputs. Do not mask failures by dropping regression selectors or
rewriting fixtures. This is removal work: retained runtime complexity, allocation behavior,
concurrency, security boundaries, and behavior at ten times normal load remain unchanged; no new
I/O, recovery, or error-handling mechanism is introduced.

## Acceptance Criteria

- **R1:** Execution starts only after fn-64 and fn-62 complete. Every remaining Umpire tooling
  package and command has a recorded retained workflow/consumer or an evidence-backed removal
  decision. Errors: unclassified items or unresolved ownership prevent completion; no item is
  deleted solely because it lacks Go importers.
- **R2:** All items classified for removal, their exclusively owned helpers/tests/fixtures, and
  obsolete direct build, generation, script, workflow, and documentation references are removed.
  Errors: retained consumers, dangling references, unaccounted deleted Test/Fuzz cases, or
  unexplained fixture losses block completion.
- **R3:** Retained workflows and explicit downstream contracts remain buildable with unchanged
  behavior, diagnostics, canonical artifacts, identities, and comments. Errors: loss of fn-5
  generic promotion, current authoring or Case Runtime support, required generator inputs, or a
  concrete downstream contract blocks the affected deletion; no silent compatibility shim is added.
- **R4:** Focused tagged tests, the complete retained model/runtime regression gates, generation
  checks when applicable, model lint, and repository lint pass or report verified inherited
  failures. Errors: reduced test selection used to conceal a regression, stale active docs, or
  any unexplained new failure blocks completion.

## Boundaries

- No implementation before the two prerequisite specs complete.
- No repetition of fn-64.8/.10 cutover work or resurrection of superseded fn-61/fn-63.
- No new runtime, authoring language, broad refactor, test consolidation, dependency, or CI system.
- No deletion outside Umpire tooling except directly owned orphaned inputs/outputs and references
  established by the consumer inventory; shared consumers remain supported.
- No speculative cleanup of fn-65 work in progress and no dependency on fn-60.

## Decision Context

A separate follow-up allows both cutovers to establish the actual surviving surface before
choosing deletions. Consumer accounting is necessary because initial runtime retirement explicitly
preserves older-generation and general artifact functionality. Removing proven unused code reduces
maintenance cost without changing the execution architecture. A wholesale directory purge would
risk retained commands, generators, and downstream workflows.

## Quick commands

```bash
go test -count=1 -tags test_dep ./tools/umpire/...
make umpire-build-model
make umpire-check-regression
make lint-model
GOLANGCI_LINT_FIX=false make lint-code
```

Use the post-cutover commands and complete selectors delivered by fn-64.10; inspect their current
build definitions before execution. Include the integration tag only for integration tests.

## References

- fn-64.8 and fn-64.10: initial deletion ledger, Case Runtime cutover, and complete regression gates.
- fn-62: completed ordinary-authoring surface.
- Retained fn-22, fn-26, fn-29, fn-33 and other open workflow specs: explicit downstream consumers.
- Umpire 4 specification and Lean Authoring Guidelines: ownership, comments, and compatibility.
