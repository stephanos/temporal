# fn-122-capabilities-and-their-laws.8 Move the capability vocabulary to temporal/capabilities and give each Model a Capabilities.scala

## Description
Owner question (2026-10-04): should capabilities be defined apart from the Models, in a `temporal/*` package? Today the split is half done. The Temporal laws and catalog already live in `model/temporal/laws`, but the framework still names Temporal's capability vocabulary: `enum Capability { Closable, Terminable, Pausable, Cancelable, Pollable, Describable }` in `model/umpire/laws/Catalog.scala`, the six binding case classes in `model/umpire/Capabilities.scala`, and a hard-coded set of those six names in `model/lifter/Capabilities.scala:125,369`. Each Model's adoption (`capabilities(m, limits)(Closable(...), ...)`) sits inside its `Properties.scala` next to feature-specific claims.

Decision taken: the capability vocabulary belongs to Temporal, and the framework keeps only the mechanism. Each entity's adoption stays with its Model, because it binds that Model's fields, but it gets a file of its own.

**Entry gate:** after fn-122.5 (law lint kinds), before fn-122.6 (documentation and close).

**Size:** M
**Files:** `model/umpire/{Capabilities.scala,laws/**}`, `model/temporal/laws/**` → `model/temporal/capabilities/**`, `model/lifter/Capabilities.scala`, each Model folder's new `Capabilities.scala`, capability lifter fixtures and refusals, `tools/umpire/internal/golden/{config.json,original.json}`, `model/README.md`, `.plans/SEMANTIC_PROTOCOLS.md`.
**Touches:** [model/umpire/**, model/temporal/**, model/lifter/**, tools/umpire/internal/golden/**, tools/umpire/model/**, model/README.md, .plans/SEMANTIC_PROTOCOLS.md]

### Approach
- Framework (`model/umpire`): a generic capability abstraction (a trait or marker that a capability kind extends, with its bound fields), `Law`, `Catalog`, `Brought`, `capabilities(...)`, `except`/`overriding`. It names no Temporal capability.
- Temporal (`model/temporal/capabilities`, renamed from `model/temporal/laws`): the six capability kinds and their bindings, the Temporal laws and the one `given Catalog`.
- Lifter: recognize capability kinds by the framework abstraction instead of the hard-coded six names; refuse a binding that is not a declared capability kind, with a fixture.
- Models: move each `capabilities(...)` declaration from `Properties.scala` into `<Model folder>/Capabilities.scala` (standaloneactivity, its `admission/` and `compositions/` subpackages, nexusoperation), keeping Definition IDs (DefinitionScope pins) and generated claim names `<machine>.<law>`.
- Behavior frozen: `model/ir/**` (including `*.laws.json`), `model/cases/**` and `lifts/expected/**` byte-identical apart from source positions and moved function symbols recorded in the golden config.

## Acceptance
- [ ] `model/umpire/**` and `model/lifter/**` name none of Closable, Terminable, Pausable, Cancelable, Pollable or Describable; a test fails if they reappear there.
- [ ] The capability kinds, their bindings, the Temporal laws and the catalog live in `model/temporal/capabilities/`; each Model folder that adopts capabilities declares them in its own `Capabilities.scala`.
- [ ] Generated claims, sidecars, IR and Cases are unchanged apart from positions and recorded symbol moves; the catalog test, lifter refusal fixtures, original-baseline and migration goldens, model gate, `lint-model`, Go tooling suite and `lint-code-fast` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
