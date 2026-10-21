# Gomad: explore site scope for choice exploration

## Goal & Context

Choice exploration forces alternatives at increasing prefixes of the decision trace in bounded breadth-first rounds. Every decision in the module is a branching point, and most of them sit in the Go runtime, grpc, net/http and other dependencies rather than in the code under test, so the budget is mostly spent on prefixes the operator does not care about.

This spec adds a site scope to choice exploration: a list of Go package patterns naming which decisions exploration may force. Decisions outside the scope still occur and are still recorded, driven by the seed and never forced. The scope is part of the exploration plan and the campaign record so the explored envelope states what was in scope. It depends on the attribution from the choice site symbolization spec.

## Architecture & Data Models

**Interleaved ordering (owner decision 2026-10-20):** implement on the Rust host after fn-232.8, without waiting for all of fn-232 or Go-host retirement. Go package locations and algorithms are semantic references only; no parallel Go-host feature is built. Preserve the existing R-IDs and same-lineage evidence/behavior contracts.

At campaign start the Runner runs the base execution, symbolizes every site in its trace, and resolves the patterns to a set of site fingerprints with a digest. Later rounds symbolize only sites not yet seen, since new prefixes reveal new sites. The filter applies where forced-alternative proposals are generated: a decision whose fingerprint is outside the set yields no proposals. Rounds, transactions and resume are unchanged.

For runnable-kind decisions the scope matches the creation site of the alternative goroutines. The runtime overlay already takes the creation PC when it assigns a goroutine identity; this spec adds the creation site offset to the goroutine-identity material so alternative identities in the trace map to creation sites. That is a versioned change to the choice wire schema with a corpus expectation diff, and it is the only contract change across the three site specs.

## API Contracts

`explore` and `plan` with `--strategy=choice-exploration` accept `--site-scope=<pattern>[,<pattern>...]`; a pattern is a Go package path or a prefix ending in `/...`. The plan payload and the campaign record carry:

```
"site_scope": {"patterns": ["go.temporal.io/server/service/history/..."], "resolved_sha256": "sha256:...", "resolved_count": 37}
```

The round summary and campaign record carry per round `{"forced_in_scope": N, "skipped_out_of_scope": M}`. A first round with no in-scope site stops with reason `site_scope_empty`.

## Edge Cases & Constraints

Record hash, failure signature and artifact format are unchanged; scope lives in the plan and campaign record only. Resume refuses a changed scope with the existing changed-bound error shape. Runnable decisions are out of scope until R11 and R12 land, and the summary says so.

## Acceptance Criteria

- **R1:** `explore` and `plan` with `--strategy=choice-exploration` accept `--site-scope` as a comma-separated list of package paths or `/...` prefixes. Errors: empty list, an empty element, a pattern with whitespace or an invalid import path character, or the flag under any other strategy → status 2 before the first execution.
- **R2:** With a scope, exploration forces alternatives only at decisions whose symbolized package matches a pattern and treats every other decision as seed-driven. Errors: a decision whose symbol is unavailable is out of scope; no other error surface.
- **R3:** Without a scope, exploration behaves exactly as today. No error surface.
- **R4:** With a scope and a target module that has no symbol information, `explore` fails with status 2 before the first execution naming the reason. No other error surface.
- **R5:** With a scope that matches no site in the base execution's trace, the first round reports an empty frontier with stop reason `site_scope_empty` and the campaign still publishes its base artifact. No other error surface.
- **R6:** The exploration plan and campaign record carry the scope patterns, the resolved site set digest and count. Errors: a plan whose scope block is malformed → refused on open with the existing plan validation error.
- **R7:** `resume` refuses a campaign whose scope patterns or resolved digest differ from the persisted ones, with the existing changed-bound error shape and status. No other error surface.
- **R8:** Each round records forced in-scope and skipped out-of-scope counts in the summary and campaign record, so exhausting the bounds reports the explored envelope under the scope. No error surface.
- **R9:** Record hash, failure signature and artifact format are unchanged. No error surface.
- **R10:** Select-kind decisions match the scope by the symbolized package of their site. No error surface beyond R2.
- **R11:** Runnable-kind decisions are in scope when at least one alternative goroutine was created in an in-scope package; otherwise out of scope. Errors: an alternative whose creation site is unavailable counts as out of scope.
- **R12:** The runtime overlay records each goroutine's creation site offset alongside its identity, versioned in the choice wire schema with regenerated codecs and a corpus expectation diff. Errors: a trace of the previous schema version replays and inspects as today and is treated as having no creation sites for scope purposes.

## Boundaries

- No exclude list; include patterns with `/...` cover the known uses.
- No scope for `--strategy=simulation-exploration` or seed campaigns.
- No change to records, failure signatures or artifact layout; the only wire change is R12.
- No symbolization beyond what the symbolization spec provides.

## Decision Context

Maintainability (plan review): duplication - none identified; structure - scope resolution, filtering and counting extend ChoiceState::commit, the largest function in campaign/exploration/choice.rs; keep future changes cohesive and review that owner before adding branches.

An include list keeps the envelope statement simple: the record says exactly which packages could be forced. Out-of-scope decisions stay on the tape so replay is unaffected and the filter is purely a search heuristic. Runnable decisions default to out of scope in the first delivery because the Runner cannot attribute them until the overlay records creation sites, and spending budget on unattributable decisions defeats the purpose. Rejected filtering by site fingerprint directly as unusable: fingerprints change with every module build.

## Quick commands

```bash
cargo test --manifest-path tools/gomad/rust/Cargo.toml -p gomad site_scope
# Run the Rust explore/plan/resume corpus lanes with and without site scope.
```
