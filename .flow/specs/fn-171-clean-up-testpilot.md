# Clean up Testpilot

## Conversation Evidence

> user (2026-10-12): "analyse the package structure of common/testing/testpilot with a research agent - does it make sense? is it well organized? are internal concerns hidden? is it simple to understand? is it clean and tidy? is it using deep modules?" Also: "include tests/testcore/testpilot".
> user (2026-10-12): "testpilot is mostly temporal-specific; I wouldn't call it a 'framework' but it can benefit from separating the reusable/generic bits, too". Conductor recommended keeping the location (`common/testing/testpilot`) and separating inside it.

## Goal & Context

The research (2026-10-12) found package jobs clear, internals well hidden and the middle deep; no relayout is warranted [paraphrase]. Genuinely generic, with no Temporal vocabulary or API import: `internal/ir`, `internal/verification`, `cel`, `predicate`, `duration`, `casefile`, `recordedrun`, `evaluation`, `publish`, `campaign`, `replay`. Not generic: the Case instruction set (`instruction.proto` imports Temporal API messages), `contract`'s opcodes and entrypoint kinds, `ProfileSpec.CommandTypes`, and about 1,500 lines of `internal/execution` that realise Temporal semantics. The core/adapter split is real at the import level, not at the vocabulary level [inferred].

## Acceptance Criteria

- **R1:** The generic set imports nothing from the Temporal adapter, `go.temporal.io/api` or `go.temporal.io/sdk`, enforced by an absolute import rule with no named exceptions. Offline readers (`recordedrun`, `evaluation`) no longer link the runtime: the umpire-assess and canary binaries do not need the runtime for assessment. `internal/execution` names its Temporal files visibly, and the README states that the instruction set is Temporal's.
- **R2:** Each Driver type has one spelling: the adapter imports `contract` directly, and the root keeps only the aliases callers need.
- **R3:** Docs and placement are tidy: Case-version claims are current, the CLI section moves to `tools/umpire`, every package has a doc, the module map matches `casefile`, test-only exports in `tests/testcore/testpilot` move to `_test.go`, generated-case loaders move to `tools/umpire/lower`, and the canary preflight test stops importing `tests/testcore/testpilot`.

## Boundaries

Held for the Go propagation phase. No relayout and no move out of `common/testing/testpilot` (owner, 2026-10-12).

## Decision Context

- 2026-10-12, owner: Testpilot is not a framework; separate its reusable parts in place.

## Requirement coverage

| Req | Task |
| --- | --- |
| R1 | .1 |
| R2 | .2 |
| R3 | .3 |
