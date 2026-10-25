---
satisfies: [R3]
---
# fn-255-thorough-framework-testing-generated.4 cargo-mutants per framework crate; triage the first run

Touches: [model/.cargo/mutants.toml, mise.toml, model/umpire/**/tests/**, model/umpire/**/src/**]

## Description
Measure how well the framework's tests catch code changes, and close the gaps found (spec R3).

**Size:** M
**Files:** cargo-mutants config (location per fn-254.9, default `model/.cargo/mutants.toml`), `mise.toml` pin, new tests for missed mutants, dead-code removals.
**Touches:** [model/.cargo/mutants.toml, mise.toml, model/umpire/**/tests/**, model/umpire/**/src/**]

### Approach
- Pin cargo-mutants via mise. Exclusions: `Debug`/`Display` impls, generated code, test support crates, and Temporal crates (out of scope).
- Run per crate, starting with the core: `umpire-front`, `umpire-ir`, `umpire-checker`, `umpire-tla`, `umpire-tlc`, `umpire-dsl`. Use nextest if fn-254.3 has landed. Set timeouts from baseline test times.
- Triage every missed mutant: add a test (the default), record it as equivalent with a reason, or remove dead code. Timeouts and unviable mutants are reported separately and never counted as caught.
- Record mutants per crate, caught/missed/timeout/unviable counts and wall time in `model/design/gate-speed.md` or a sibling note.

### Investigation targets
**Required:**
- `model/design/gate-speed.md` (fn-254.1) — test times for timeouts
- `model/umpire/front/src/`, `model/umpire/backends/native/src/` — highest-value crates

### Acceptance
- [ ] Committed config; a run per core crate with counts and wall time recorded.
- [ ] Every missed mutant from the first run is triaged, with tests added or reasons recorded.
- [ ] The report separates missed, timed-out and unviable mutants.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
