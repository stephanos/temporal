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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
