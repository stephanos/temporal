---
satisfies: [R3, R7]
---
# fn-255-thorough-framework-testing-generated.4 cargo-mutants per framework crate; triage the first run

Touches: [model/umpire/.cargo/mutants.toml, model/umpire/mise.toml, model/umpire/**/tests/**, model/umpire/**/src/**]

## Description
Deliver per-crate mutation commands, a bounded smoke run and scheduled triage reports (spec R3), within R7 budgets.

**Size:** M
**Files:** cargo-mutants config (location per fn-254.9, default `model/umpire/.cargo/mutants.toml`), `model/umpire/mise.toml` pin, new tests for missed mutants, dead-code removals.
**Touches:** [model/umpire/.cargo/mutants.toml, model/umpire/mise.toml, model/umpire/**/tests/**, model/umpire/**/src/**]

### Approach
- Serialize changes to shared source, tests, framework pins/configuration and the budget table with .2/.3/.5/.6/.7; inspection may run in parallel, but each mutation run owns its scratch source and any real fixes wait for exclusive ownership.
- Pin cargo-mutants via mise. Exclusions: `Debug`/`Display` impls, generated code, test support crates, and Temporal crates (out of scope).
- Provide scheduled runs per crate, starting with the core: `umpire-front`, `umpire-ir`, `umpire-checker`, `umpire-tla`, `umpire-tlc`, `umpire-dsl`. Use nextest if fn-254.3 has landed. Set timeouts from baseline test times.
- At closure, run a fixed smoke manifest with hard mutant-count and wall-clock caps from the committed budget table. Full per-crate campaigns remain scheduled-only, with per-mutant timeouts and a total run cap; report unfinished work and resume inputs when capped.
- Triage observed missed mutants: add a test (the default), record it as equivalent with a reason, or remove dead code. Timeouts and unviable mutants are reported separately and never counted as caught.
- Record smoke and scheduled results separately, with per-crate caught/missed/timeout/unviable counts, wall time, exact manifest and completed/remaining work. Report generation belongs to the framework command; evidence may be linked from `model/design/gate-speed.md`. Full campaigns and exhaustive triage are not spec-close prerequisites.

### Investigation targets
**Required:**
- `model/design/gate-speed.md` (fn-254.1) — test times for timeouts
- `model/umpire/front/src/`, `model/umpire/backends/native/src/` — highest-value crates

## Acceptance
- [ ] Committed config and commands support each framework crate; the bounded smoke manifest runs within its work/time caps with counts recorded.
- [ ] Smoke findings are triaged; scheduled reports retain observed missed mutants and completed/remaining work without requiring full-campaign completion at closure.
- [ ] Report controls separate caught, missed, timed-out and unviable mutants, and demonstrate capped partial results; timeouts never count as caught.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
