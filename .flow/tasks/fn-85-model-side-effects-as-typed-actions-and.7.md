---
satisfies: [R7, R8]
---
# fn-85-model-side-effects-as-typed-actions-and.7 The set command, abstraction claims, derived Case identity and the list-driven generator

## Description
Add the `set` command with `purpose:`, `bind:`, `repeat:`, `queries:`, `cover:` and `budget:` (R7): a functional set compiles to one Case per Query with identity `temporal.case.<set>.<query>` and fixture `<set>-<query>-case.json`; `umpire-case --list` lists exactly the functional sets' Queries; every Case realizing a multi-value class records an abstraction claim row in provenance (R8). Fold in fn-83 .4's dropped concerns: the Go conformance generator reads `--list` instead of its functional table, and the artifact test is table-driven over the testdata directory.

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean` (`set`), `model/Umpire/Command/Authoring.lean` (set admission: bindings, purposes, examples for realized classes), `model/Umpire/Command/Records.lean` (`PartyBinding`, `SetPurpose`, `AbstractionClaim`), `model/Umpire/Case/Producer.lean` (claims into provenance; example selection for `driven` classes; `observed`-party action on a functional path rejects), `model/Umpire/Provenance.lean` (an `AbstractionClaimRow` in fn-87's structured provenance), `model/Temporal/Case/Registry.lean` (functional-set Cases register by derived identity; duplicate derived fixture rejects), `model/Temporal/Tool/Testpilot.lean` (`--list` over functional sets), `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go` (functional table and count guard removed; `--list` consumed), its `generate_test.go`, `tests/testcore/testpilot/artifact_test.go` (table-driven decode/prepare/identity over `testdata/*-case.json`), `tests/testcore/testpilot/README.md`
**Touches:** [model/Umpire/Command/**, model/Umpire/Case/Producer.lean, model/Umpire/Provenance.lean, model/Temporal/Case/Registry.lean, model/Temporal/Tool/Testpilot.lean, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tests/testcore/testpilot/**]

### Approach
- Set admission rejections (R7): an unbound non-`system` party; a bound `system`; a functional Query whose path needs an action of an `observed` party; a `verify` Query in a functional or canary set; a duplicate derived fixture; an exploratory set without a coverage goal. Canary white-box Known Gap and exploratory enumeration are task .12.
- Examples: a multi-value class realized on a functional path with no example rejects at production naming the class; single-value classes carry no claim.
- `--list` order is the generator's determinism assertion: sort by Case ID; `(set, query)` gives a total order.
- The Go generator keeps `syntheticEntry()` hard-coded (it is not a Model Case) and reads everything else from `--list`.

### Investigation targets
**Required:**
- `model/Temporal/Case/Registry.lean:20-81` — entries, `register_case`, `registeredCases%`
- `model/Temporal/Tool/Testpilot.lean:17-78` — the renderer and `--list`
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:87-120,309-355,405-441` — `functionalEntry`, functional generation, the `--list` consumer, `syntheticEntry`
- `tests/testcore/testpilot/artifact_test.go:30-300` and `fixture_table_test.go:16-29`
- `model/Umpire/Provenance.lean` (post-fn-87 rows)

**Optional:**
- `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.4.md` — the folded concerns

### Key context
- Case bytes do not depend on switch values; identity derives from set and Query names only.

## Acceptance
- [ ] `set` declarations elaborate; each R7 rejection this task owns has a `#guard_msgs` specimen; a functional set over the success Model produces a Case with ID `temporal.case.<set>.<query>` and fixture `<set>-<query>-case.json`
- [ ] `umpire-case --list` prints exactly the functional sets' Queries sorted by Case ID; the Go generator has no functional table and no fixed-count guard; the artifact test is table-driven
- [ ] a Case realizing a multi-value class carries the claim row with action, field, class and example; single-value classes carry none; a missing example rejects at production naming the class. **Amended by task .2:** a class is a *member* of the input domain, so every class has exactly one member and nothing in the Model counts the concrete values a class covers. The claim's trigger is therefore the presence of an `examples:` line for that class, not a member count: a class an author wrote an example for is one they claim several realized values behave alike in, and a class with no example carries no claim
- [ ] `make umpire-check-case-runtime-conformance` exit 0; `go test -tags test_dep ./tools/umpire/... ./tests/testcore/testpilot/...` green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
