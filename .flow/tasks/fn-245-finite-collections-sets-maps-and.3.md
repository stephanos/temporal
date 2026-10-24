---
satisfies: [R4]
---
# fn-245-finite-collections-sets-maps-and.3 TLA+ emission and typed TLC decoding for sets, partial maps with the Absent sentinel, quantifiers and list Head/Tail

## Description
After .1 (parallel with .2 where files do not overlap; serial where `umpire-tla` tests consume front-end output). In `model/umpire/backends/tla` and `backends/tlc`: emit a set field as a subset of its finite element set with a type invariant and cfg bound; a map as a total function from the key domain to the value domain plus one `Absent` sentinel, so `contains_key` is a sentinel test and `MapGetOr` a guarded lookup; `Forall`/`Exists` as `\A`/`\E` over the element set; `Front`/`PopFront` as `Head`/`Tail` under the existing sequence emission; `Length` under fn-242's rule. Extend typed decoding (`backends/tlc/src/decode.rs`) to read sets, functions and sequences back into canonical `Value`s, refusing duplicates, the sentinel in a value position, and crossed types; replay through `checked_step` as today. Every Model without collections keeps the same TLC results; its modules and cfgs may be re-accepted after review. Controls: emission unit tests per node; a decoded-trace round trip per collection kind; a forged duplicate refusal.

Touches: model/umpire/backends/tla/**, model/umpire/backends/tlc/**.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
