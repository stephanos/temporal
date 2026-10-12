---
satisfies: [R4, R5]
---
# fn-165-gomad-architecture-checks-structural.4 Implement the purity rules with review marks and re-triage the effect fixtures

Touches: tools/gomad3/internal/puritycheck, tools/gomad3/testdata/architecture

## Description
Over typed call sites in pure files only: a standard-library call must be on the allowlist (`host-effect`); a module call must resolve to a function declared in a pure file (`host-effect`), including same-package impure files; `time.ParseInLocation` is admitted only when the location argument is the identifier `time.UTC`; every method declared in a pure file is checked whether or not it is called; a call through a function value, method value or interface method whose static type is declared outside the pure set is `dynamic-call` unless the same line or the line above carries `//gomad:dynamic reviewed <date> <reason>`; a call on an interface declared in the pure set whose module implementations are all pure is static; a mark on a line without a dynamic call, or without a date and reason, is `stale-review`. Log the review marks per package. Re-triage the 36 fixtures in `testdata/architecture/effects`: the eight host-effect fixtures and the pure fixtures keep their expectations; callback-resolution fixtures become `dynamic-call` fixtures with a marked and an unmarked variant. Port `PublicSignatures` from the old `program.go` onto the new loader.

## Acceptance
- Each host-effect fixture (indirect-clock, environment, filesystem, process, network, timer, entropy, goroutine) reports `host-effect`; each pure fixture reports nothing.
- `time.ParseInLocation(layout, value, loc)` with `loc` a variable reports `host-effect`; with `time.UTC` it does not.
- An unmarked dynamic call reports `dynamic-call`; the same call with a dated mark passes; a mark on a static call reports `stale-review`; a mark without a date reports `stale-review`.
- A `MarshalJSON` method in a pure file that calls `os.ReadFile` reports `host-effect` without anything calling the method.
- `PublicSignatures` fixtures in gomad3's `architecture_test.go` pass through the new entry point.
- `puritycheck` is at most 2,000 non-test lines in total.

## Done summary
Typed purity rules check stdlib allowlists, module calls, every pure-file method, dynamic calls with dated review marks and stale marks; review fixes make promoted and externally satisfiable interface calls dynamic, check all json/fmt callback hooks and require stdlib stream wrappers to be built over checked streams. Effect fixtures re-triaged.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c6afac5097, 46bf30c512, 88bff11fc5, c20d82bf05, b3eee0867d
- Tests: root architecture/purity selection with GOMAD3_ARCH_ALL_PLATFORMS=1 plus ./internal/archcheck ./internal/puritycheck (0), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: