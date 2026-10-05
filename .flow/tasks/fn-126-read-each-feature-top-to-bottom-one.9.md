---
satisfies: [R21]
---
# fn-126-read-each-feature-top-to-bottom-one.9 Lint a reachable state that is no end and enables nothing (stuck-state)

## Description
Implements R21: the `stuck-state` lint kind (owner request, 2026-10-05).

A reachable state of a machine or composition that is not an `end` state and in which no action class has an enabled row is a finding: machine, state, and a shortest path to it as the witness, at the machine's position. It is on by default, like `silent-rejection` and `never-enabled`. Reachability and stuck states are already computed by the reader (`tools/umpire/model/machine.go:158`); this adds the lint pass, the kind, its message and its fixtures.

Runs in parallel with fn-126.5: it touches `tools/umpire/lint/**`, its testdata, `tools/umpire/lint/testdata/coverage.golden` if affected, the README's lint table and, if today's Models have stuck states, their acceptances in `model/ir/*.lint.json` with a reason each (or a reported Model bug; do not change a Model here). Whichever of this and fn-126.5 lands second rebases the acceptance files.

## Acceptance
- [ ] `stuck-state` is a default lint kind; it reports every reachable non-end state with no enabled row, with a shortest path, and nothing else.
- [ ] One fixture produces exactly the expected finding; one passing fixture produces none; the README lists the kind.
- [ ] Every finding on today's Models is either accepted in its `*.lint.json` with a reason, or reported to the host as a Model bug (no Model edited here).
- [ ] `./tools/umpire/lint`, `./tools/umpire/model`, the model gate, `make umpire-check-cases` and lint-code-fast pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
