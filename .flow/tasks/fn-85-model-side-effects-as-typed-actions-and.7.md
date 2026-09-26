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
- [x] `set` declarations elaborate; each R7 rejection this task owns has a `#guard_msgs` specimen; a functional set over the success Model produces a Case with ID `temporal.case.<set>.<query>` and fixture `<set>-<query>-case.json`
- [x] `umpire-case --list` prints exactly the functional sets' Queries sorted by Case ID; the Go generator has no functional table and no fixed-count guard; the artifact test is table-driven
- [x] a Case realizing a multi-value class carries the claim row with action, field, class and example; single-value classes carry none; a missing example rejects at production naming the class. **Amended by task .2:** a class is a *member* of the input domain, so every class has exactly one member and nothing in the Model counts the concrete values a class covers. The claim's trigger is therefore the presence of an `examples:` line for that class, not a member count: a class an author wrote an example for is one they claim several realized values behave alike in, and a class with no example carries no claim
- [x] `make umpire-check-case-runtime-conformance` exit 0; `go test -tags test_dep ./tools/umpire/... ./tests/testcore/testpilot/...` green


## Done summary

### The `set` command

`set <name>` takes `purpose:` (functional, canary or exploratory), `bind:` (`<party>: driven |
observed`), `repeat:`, `queries:`, `cover:` (`rows | results | classMembers`) and `budget:`, and
elaborates into `Umpire.Command.SetDeclaration` (`PartyBinding`, `SetPurpose`, `CoverageGoal` in
`Records`) with a `Registry.SetEntry` beside it. It checks what the Model alone decides, each at the
line that made it: every party the set's machines' actions name is bound and `system` is not, a
party no action names rejects, a binding is `driven` or `observed`, a functional or canary set lists
`find` Queries and an exploratory set a goal and a budget (the other keys reject on the wrong
purpose), a functional path performs no action of an `observed` party (the Case would have to
perform it), and `repeat:` belongs to a functional set and names a switch a realization registered.
The parties are scoped to the actions the set's Queries' machines step on -- an exploratory set,
which lists no Query, to the actions declared beside it -- because a Model file imports other
Models' actions. Ten `set` rejections are pinned in the success tests by `#guard_msgs` (an unbound
party, a bound `system`, a party no action names, an `observed` party's action on a functional path,
a `verify` Query, `repeat:` on a canary set, an unregistered switch, an exploratory set without
`cover:`, `queries:` on an exploratory set, an unknown purpose), and two more on the `case` block over
a set (a duplicate derived fixture, a set that is not functional). An unknown binding word, a
duplicate binding, a missing `budget:` and `cover:`/`budget:` on a non-exploratory set reject with
their own messages but have no pin.

A switch is the platform's, so `register_switch <SwitchBinding>` (Umpire) records its name and
values, and `Temporal.Case.Syntax` registers the Nexus realization's `implementation` switch once;
an unknown name rejects at `repeat:` naming the declared switches.

### Derived Cases

`case <name> realizes <set> as <template> evidence <lines>` realizes every Query of a functional set:
one realization for all, and per Query the identity `temporal.case.<set>.<query>` with fixture
`<set>-<query>`, the evidence lines for the Actions that Query's own path selects (a line for an
Action a path never selects is a production rejection, so the map is filtered per Case), and a
registry entry like any Case, so a duplicate derived fixture rejects with the existing message and
`umpire-case --list` and the list-driven generator carry it. A set that is not functional rejects.
The success slice declares `nexusSuccessTests` over `completion` and its Case
`temporal.case.nexusSuccessTests.completion` is generated as `nexusSuccessTests-completion-case.json`,
which the table-driven artifact test decodes, prepares and pins. The `fixture`-named `case` block
stays until `.11` removes it, and the four Cases that register their values explicitly stay until
fn-86 R3, so `--list` is the registry rather than the functional sets alone.

### Abstraction claims

`CaseProvenance` gains its ninth row kind, `AbstractionClaim` (action, field, class_name, example),
additive on fn-87's shapes: `make protoc` regenerated `api/testpilot/v1` (case.pb.go and its
helpers only), `Testpilot.Protocol` re-elaborated, `Testpilot.Authoring.abstractionClaim` and a
`provenance` argument carry it with guards, `Umpire.Provenance.AbstractionClaimRow` lowers it, and
`protocol_test.go`'s field-name pin names it. On the Model side a machine's declared Model carries
each action member's class as the Model spells it (`DeclaredNames.actionClasses`, the
`ClassValue.render` spelling an `examples:` line uses), `Umpire.Command.classClaims` pairs each
example with the member that realizes its class (`Producer.ClassClaim`), and the Producer records
the claims of the classes the Program performs. Per the `.2` amendment a class with no example
carries no claim and rejects nothing. Pinned on a `probe` action with a `slow → Sluggish` example:
the slow path's Case carries the row, the quick path's carries none.

### Folded from fn-83 `.4`

Already in the tree from its Go half (dda17feda8): the generator reads `--list` with no functional
table and no fixed count (the six-class guard is the conformance manifest's), and the artifact test
is table-driven over `testdata/*-case.json`. This task adds the README paragraph on set-derived
fixtures.

### Gates

`lake build` green (628 jobs); `make umpire-gen-case-runtime-conformance` produced the new fixture
and left every other unchanged; `umpire-check-case-runtime-conformance`,
`umpire-check-testpilot-protocol`, `umpire-check-testpilot-authoring`, `umpire-check-goldens`,
`umpire-check-inventory` and `umpire-check-retired-vocabulary` exit 0; `go test -tags test_dep
./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...` green;
`GOLANGCI_LINT_BASE_REV=43cb178 make lint-code-fast` 0 issues; `LEAN_NUM_THREADS=1 make
lint-model` at the baseline.

Self-review: no second backend is installed in this cloud session, so this owes a cross-model
re-review before the completion review, as the tasks before it do.

### Re-review fix, 2026-09-19

The cross-model re-review found the `set` command's party scan and the `case` block's claim list
resolving declarations by bare name over the whole registry, so a same-named Query or action in an
imported Model could stand in for the one the set lists. A machine now records the `action`
constants it steps on (`MachineEntry.actionDecls`), the set resolves each listed Query to its
constant once and reads the parties off those declarations, and the claim list is built from them.
`Umpire.Command.AbstractionClaim` and `AbstractionClaim.ofRow` were a second spelling of the
provenance row nothing used; both are deleted.

## Evidence
- Commits: ace06c5
- Tests: cd model && lake build; make umpire-gen-case-runtime-conformance; make umpire-check-case-runtime-conformance; make umpire-check-testpilot-protocol; make umpire-check-testpilot-authoring; make umpire-check-goldens; make umpire-check-inventory; make umpire-check-retired-vocabulary; go test -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...; GOLANGCI_LINT_BASE_REV=43cb178 make lint-code-fast; LEAN_NUM_THREADS=1 make lint-model
- PRs:
