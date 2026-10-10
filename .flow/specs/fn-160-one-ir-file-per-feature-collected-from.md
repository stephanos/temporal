# One IR file per feature, collected from the feature's package

## Conversation Evidence

> user (turn 15): "explain the idea behind irFile - wht are there more than one per feature?"
> user (turn 16): "it seems overly complicated; why not a single file per feature? where you only specify the root of the graph and everything else is collected along with it?"
> user (turn 17): "yes, make separate spec"
> user (turn 17): "should we require a file to at most contain only one Machine? that would force a separate file for each and help scan them quickly as they are all visible from the file tree"
> user (turn 18): "yes add to fn-160."
> user (turn 33): "yes do A"
> user (turn 33): "but let's not have nested packages like pausing/ and instead a flat list of files grouped by their prefix"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 50% [user], 50% [paraphrase] -->

Every feature currently declares its checked-in IR files by hand. An `exports` object lists one `irFile(name)(roots*)` per file, and each root (machine, composition, `queries` section, `capabilities` section, realization) is named explicitly. The standalone activity splits into three files with about fifty roots between them, and nexus workflow into three more. The user finds this overly complicated. The lists are an inventory that has to be kept in step with the code, and a `queries` section nobody lists silently drops out of the IR.

The goal: one IR file per feature, collected automatically, with nothing to list.

## Architecture & Data Models
<!-- scope: technical -->

References in the model graph point downward: a Query to its machine, a realization to its machine, a machine to the machine it refines. No single object reaches everything a feature declares, so a single root object would miss the Queries, capability claims and realizations. [paraphrase]

The feature's **package** is the root instead. The IR generator, which already reads every Model's compiled TASTy in one run, collects every machine object, composition, `queries` section, `capabilities` section, progress claim and realization declared in the feature's package and its sub-packages (`product/`, `system/`). It then adds everything those reach, as today, e.g. a shared foundation's machines. One feature yields one IR file. [paraphrase]

What a feature folder is follows the existing layout rules: a folder with a feature file, such as the standalone activity, nexus standalone or nexus workflow. [inferred]

## Edge Cases & Constraints
<!-- scope: technical -->

- A declaration reached from two features is lifted into each feature's file, as a root of two files is today. [inferred]
- Today's split separates design alternatives and adversarial models from the main model: the history-record and admission designs, the held race, the forged control, and the close and reset designs. Merged in, they stay distinguishable by their machines and by the expectations their Queries already declare. [paraphrase]
- Generated Case file names start with their IR file's name, and two other trees pin Cases byte for byte (the functional tests' generated fixtures and the canary's case bindings). Merging files renames Cases in all three. [paraphrase]
- Go tests and the canary preflight harness name today's files (`activity-standalone-race`, `activity-standalone-record`, …) directly. [paraphrase]
- Each IR file has its own accepted-lint-findings file; merged files need merged acceptances. [paraphrase]

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Before any migration, every consumer of the IR's file boundaries is identified: Go tools, the gate, Quint export slices, Case naming and fixture trees, the canary, and any checker that is given one whole IR file and relies on it holding only some Queries. Each gets a stated migration or a stated reason to keep a split. [inferred] Errors: if a consumer needs a split that a single file can't serve, the spec records it and the user decides before migration starts.
- **R2:** Each feature folder produces exactly one IR file, holding every machine, composition, Query section, capability section, progress claim and realization declared in its package and sub-packages, plus what they reach. [paraphrase] Errors: a feature package that declares nothing liftable produces no file and no error; a declaration the generator cannot lift is refused as today, naming it.
- **R3:** No Model lists IR roots: `irFile` and the `exports` objects are removed, and the gate's construct-every-root test constructs what each feature package holds. [paraphrase] Errors: no error surface beyond R2.
- **R4:** A Query section, capability section or realization can no longer be left out of the IR by omission. [paraphrase] Errors: if a declaration must stay out of `model/ir` on purpose, an explicit opt-out marks it, and the generator reports each opted-out declaration.
- **R5:** The merged IR keeps every machine, Query, Case and claim the split files had, with the same behaviour. The only changes are file boundaries, file names and the names derived from them (Case files, lint acceptances, export slices). [inferred] Errors: any other difference in the regenerated IR or Cases is a defect to explain before it is accepted.
- **R6:** Every consumer from R1 is migrated (Go tests, canary preflight, fixture trees, export README tables, README), and the full gate passes. [inferred] Errors: no error surface beyond R5.
- **R7:** Each Model file declares at most one machine, composition or derivation, in a file named after it, in flat level folders where a family's files share a name prefix (no nested packages), so the sorted file list shows every machine and its group. [user] ("let's not have nested packages like pausing/ and instead a flat list of files grouped by their prefix"; "should we require a file to at most contain only one Machine? that would force a separate file for each and help scan them quickly as they are all visible from the file tree"; "yes add to fn-160.") Errors: a file with a second machine is refused by the lifter, naming the file and the machine; the regenerated IR differs only in source paths and positions.
- **R8:** Realizations form a layer of their own: a flat `realization/` folder beside `product/` and `system/`, each realization in a file named after its subject plus `Run`, with shared helpers in one kit file. [user] ("yes do A") Errors: a realization not named after the subject it realizes is refused by the lifter, naming both.
- **R9:** Layer dependencies point one way, realization → system → product: product imports neither system nor realization, system doesn't import realization, and only the realization folder may use the Temporal API. [paraphrase] Errors: an import against the direction, or an API use outside `realization/`, fails the lint at its line.

## Boundaries
<!-- scope: business -->

- Changing what the models say (machines, Queries, claims, realizations) is out of scope; only how IR files are assembled and named, and how Model files are laid out, changes. [inferred]
- Changing the IR schema is out of scope. [inferred]

## Decision Context
<!-- scope: both — conditionally substructured -->

The user asked for a single file per feature where only the root of the graph is specified. [user] A single root object was rejected because model references point downward and no object reaches a feature's Queries, claims and realizations. The feature's package serves as that root instead. [paraphrase] Keeping explicit `irFile` lists but defaulting the main file to its folder's sections was considered as a middle ground. It was not chosen: it keeps the multi-file split and part of the inventory. [paraphrase]

## Parked unknowns

- Whether any consumer relies on today's file boundaries in a way one file per feature can't serve. The exploration and race checkers are the suspects, from the race file's comment "so the history record's Queries are the ones its checkers were given". R1's investigation resolves this. [inferred]
