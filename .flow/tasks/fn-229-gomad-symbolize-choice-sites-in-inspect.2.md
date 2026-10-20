---
satisfies: [R1, R2, R5, R6, R9]
---
# fn-229-gomad-symbolize-choice-sites-in-inspect.2 Inspect output: attach symbols to sites, decisions and divergences

**Touches:** tools/gomad/rust/gomad/src/cli/inspect.rs, tools/gomad/testdata/parity/**, tools/gomad/CLI.md
**Semantic references only:** tools/gomad/internal/runner/inspect/inspect.go, inspect testdata, tools/gomad/CLI.md

## Description
Implement once on the Rust host after fn-232.5 and the existing site-spec predecessors; the Go locations below document existing semantics only. Port the behavior into the Rust ownership surface above and extend the shared golden/behavioral corpus. Do not create or extend a Go-host implementation.
When the record has a choice profile, open the retained module through the artifact handle, build the symbolizer once, and populate `ChoiceSite.Symbol`, `ChoiceDecision.Symbol` and divergence expected/observed decision symbols with the exact JSON shape in the spec's API Contracts (field order fixed, `inlined_from` always present, file/line omitted when unknown). Site-missing records get `unavailable: site_missing`. Add a golden `--json` test on an existing retained artifact. Document the field in CLI.md.

## Acceptance
Golden inspect JSON for a retained artifact includes symbols on every site and decision and is byte-stable across runs; an artifact with a stripped module still inspects with `unavailable: no_symbols`; record hash and failure signature of the artifact are unchanged (asserted in the test).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
