---
satisfies: [R1, R4]
---
# fn-83-author-a-live-case-from-a-model-file.3 The case command, Case registry, and umpire-case --list/--render

## Description
Add the sixth command `case` to the success syntax module as an elaborator that resolves the Query, template, evidence mapping, and fault hooks, derives the Case identity from `fixture`, converts the Temporal bundle to the Producer input, calls `Umpire.Case.Producer`, and registers the result in a new environment extension (R1, R4). Make definition family and source per file. Replace the renderer's functional dispatch table with `--list` and `--render` over that registry, register the two typed Cases, and delete the success Producer file. This is the first environment extension in `model/`; there is no in-repo precedent.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Success/Syntax.lean` (the `case` elaborator; family and source derivation), `model/Temporal/Feature/Nexus/Success/Authoring.lean` (family and source become parameters), `model/Temporal/Feature/Nexus/Success/Model.lean` (ends in a `case ... fixture "async-nexus"` block), `model/Temporal/Feature/Nexus/Success/Producer.lean` (deleted), `model/Temporal/Case/Registry.lean` (new), `model/Temporal/Tool/Testpilot.lean` (functional dispatch replaced by the materializing elaborator), `model/Temporal/Feature/Nexus/Success/TypedUnary.lean` and `TypedNexus.lean` (one explicit register call each; Programs, Profiles, Contracts untouched), `model/Temporal/Feature/Nexus/Success/Tests.lean` (new `#guard_msgs` and the distinct-ID `#guard`), `model/lakefile.lean` if the exe needs new roots
**Touches:** [model/Temporal/Feature/Nexus/Success/**, model/Temporal/Case/**, model/Temporal/Tool/Testpilot.lean, model/lakefile.lean]

### Approach
- The `case` command must be `elab ... : command` (not `macro`) like the `model` command, because it resolves names and writes the extension. Follow the `model` command's located-error style: `throwErrorAt <syntax> <message>` with message helpers alongside the existing ones, and `resolveMember`-style listing of declared spellings.
- Identity: `fixture "<name>"` is required; derive Case ID `temporal.case.<name>`, Program ID `<caseId>.program`, Contract ID `<caseId>.contract`, scope literal `<name>`. The async-Nexus fixture therefore changes in exactly those fields plus Provenance; regenerate through the owning target and list the diff in the receipt. With those fields masked, Program and Contract must be byte-identical to .2's output; write that mask comparison as a Lean or Go test, not a manual check.
- Family and source: the five existing commands take the definition family from the enclosing namespace and the source from the elaborating file (`Lean.Elab.getFileName`/`getRef` position), replacing the module-level constants; the Nexus success family value must not change under that derivation. Pin one `#guard` that two Models in different test files produce distinct target IDs and sources.
- Resolve the Query by name and reject `verify` forms; resolve each `evidence` Action against the Scenario's `actions exactly` list, rejecting unmapped and unselected Actions; resolve event kinds through .2's resolver against the template's sources.
- Registry: a `SimplePersistentEnvExtension` whose entries are `(declaration Name, caseId, fixtureName)`; never a closure, because entries are written to the `.olean`. A term elaborator (for example `registeredCases%`) in the renderer's root module reads the imported environment, sorts by Case ID, and materializes a `List` of constant references; it throws the duplicate-fixture diagnostic. The renderer keeps `renderTestpilot`/`renderSynthetic`; the synthetic and conformance arms stay reachable by their existing arguments for the Go conformance builder, and only the functional arms are replaced by the registry.
- Pin with `#guard_msgs (error)`: `verify` Query, unmapped selected Action, evidence for an unselected Action, unknown event kind, duplicate fixture name (five pins; the two fault pins belong to .5).

### Investigation targets
**Required:**
- `model/Temporal/Feature/Nexus3/Syntax.lean:47-94, 114-195` — message helpers, `domainConstructors`, `resolveMember`, and the `model` elaborator to mirror
- `model/Temporal/Feature/Nexus3/Authoring.lean:15-40` — the module-level `family` and `source` constants to parameterize
- `model/Temporal/Tool/Testpilot.lean:8-38` — renderers and the dispatch table
- `model/Temporal/Feature/Nexus3/Tests.lean:572-758` — the `#guard_msgs` pinning convention
- `model/lakefile.lean:101` — the renderer exe declaration (`umpire-case` after fn-82)

**Optional:**
- Lean core `registerSimplePersistentEnvExtension` docs; `Lean.Elab.Term` elaborator examples for materializing environment data

### Key context
- fn-82 task .8 respells the five commands and renames the exe; build on its grammar. Each retired keyword keeps a located-error arm; the `case` command adds none.
- fn-82 task .9 adds a test that every backticked Lean name in `UMPIRE4_SPEC.md` resolves; the `case` concept entry (.8 of this spec) must satisfy it.
## Acceptance
- [ ] The success Model file ends in a `case ... fixture "async-nexus"` block and the Producer file is deleted
- [ ] The regenerated async-Nexus fixture differs only in Case ID, Program ID, Contract ID, scope literal, and Provenance; a masked comparison test proves Program and Contract equality, and the receipt lists the diff
- [ ] `umpire-case --list` prints the registered functional Cases sorted, including both typed Cases; `--render <id>` prints bytes equal to each checked-in fixture; unknown ID exits non-zero naming the known IDs
- [ ] Five `#guard_msgs` blocks pin the diagnostics listed in Approach; one `#guard` pins distinct IDs and sources across files
- [ ] `make umpire-check-case-runtime-conformance` and `make lint-model` pass
## Done summary
The sixth command lands. A Model file plus one `case` block is a Case.

- `model/Temporal/Feature/Nexus/Success/Syntax.lean`: the `case` elaborator (`fixture`, `realizes`,
  `as <template>`, `evidence <action> ← history <kind>`), the per-file origin derivation for all
  five existing commands, and `scenario`/`query` converted from macros to elaborators so they
  record their surface.
- `model/Temporal/Case/Registry.lean` (new): three environment extensions holding names and plain
  data only (a Scenario's selected Action order, a Query's form and Scenario, each registered
  Case), plus `register_case` for Cases that carry their identities in Lean, and the
  `registeredCases%` term elaborator that materializes the sorted list and rejects a duplicate
  fixture or Case ID.
- `model/Temporal/Feature/Nexus/Success/Authoring.lean`: `Origin` (family + source) threaded through
  `SuccessModel`; `producerInput`, `produce`, `produceCase` moved here from the deleted Producer.
- `model/Temporal/Feature/Nexus/Success/Model.lean` ends in `case asyncNexusSuccess fixture
  "async-nexus"`; `Producer.lean` is deleted.
- `model/Temporal/Tool/Testpilot.lean`: `--list` / `--render <case-id>` over the registry; the
  functional dispatch table is gone. Fixture-name arguments still resolve (through the registry),
  which is what keeps the Go generator green until .4 deletes its table.
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go`: the async-nexus manifest
  entry's expected Case ID follows the derivation.

Fixture diff, exactly two fields, both derived from `fixture`:
  caseId              temporal.case.async-nexus-success           -> temporal.case.async-nexus
  contract.contractId temporal.case.async-nexus-success.contract  -> temporal.case.async-nexus.contract
The Program ID was already the derived value, the run-scope literal is the fixture name, and the
Provenance is unchanged. A `#guard_msgs (info)` masked comparison on the canonical ProtoJSON pins
that the same Model under the previously stated identity is byte-identical once those two fields
are masked.

Identity derivation moved two test-file Models to their own families, which is the derivation
working: `RaceSyntaxTests` is now `temporal.nexus.success.raceSyntax.*` and `Tests`' probe Model is
`temporal.nexus.success.tests.*`. No fixture carries either.

Five `#guard_msgs (error)` pins: `verify` Query, unmapped selected Action, evidence for an
unselected Action, unknown history event kind (listing all 60 admitted kinds), duplicate fixture.
Plus `#guard`s that two Models in different files get distinct target IDs, families and sources.

Deviation: the duplicate-fixture diagnostic fires in the `case` command (a located error on the
`fixture` string) as well as in the materializing elaborator; the spec placed it only in the
latter. The command's is the one pinned, because it is the one an author sees.

Verified: `umpire-case --render <id>` is byte-equal to each of the five checked-in fixtures, an
unknown ID exits 1 naming the known IDs, and `make umpire-check-regression` is exit 0 end to end
(562 Lean jobs, all eight conformance checks, 6 passing live identities).

Review: SHIP, 2 non-blocking findings. The P2 (`packageRelativePath` splitting on the first
`/model/`) was valid and fixed by taking the last segment. The P3 (O(n^2) duplicate scan over five
registry entries) was not taken.
Pinned reviewer `claude:claude-fable-5-1:high` is account-limited for this session, so the review
ran on `claude:claude-sonnet-4-5:high` -- a same-family fallback, not an equivalent cross-family
review.

stage: impl-review - ran (model: claude-sonnet-4-5, high; fable pinned but account-limited)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 508cc86636
- Tests: make umpire-check-regression (exit 0), cd model && mise exec -- lake build, make umpire-check-case-runtime-conformance, make lint-model (0 findings outside generated Temporal/API/Proto.lean), umpire-case --render <id> byte-equal to all five checked-in fixtures
- PRs: