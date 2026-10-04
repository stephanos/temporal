# fn-122-capabilities-and-their-laws.8 Move the capability vocabulary to temporal/capabilities and give each Model a Capabilities.scala

## Description
Owner question (2026-10-04): should capabilities be defined apart from the Models, in a `temporal/*` package? Today the split is half done. The Temporal laws and catalog already live in `model/temporal/laws`, but the framework still names Temporal's capability vocabulary: `enum Capability { Closable, Terminable, Pausable, Cancelable, Pollable, Describable }` in `model/umpire/laws/Catalog.scala`, the six binding case classes in `model/umpire/Capabilities.scala`, and a hard-coded set of those six names in `model/lifter/Capabilities.scala:125,369`. Each Model's adoption (`capabilities(m, limits)(Closable(...), ...)`) sits inside its `Properties.scala` next to feature-specific claims.

Decision taken (owner, 2026-10-04: `model/umpire` stays Temporal-agnostic as far as realistic): the capability vocabulary and every law belong to Temporal, and the framework keeps only the mechanism. That includes the "entity-neutral" laws in `model/umpire/laws/Laws.scala` (`terminalStatesAreFinal`, `closedIsRejectedUniformly`): their bodies read Temporal capability kinds and their citations are Temporal server files, so they move too and the framework holds no law at all. Each entity's adoption stays with its Model, because it binds that Model's fields, but it gets a file of its own.

**Entry gate:** after fn-122.5 (law lint kinds), before fn-122.6 (documentation and close).

**Size:** M
**Files:** `model/umpire/{Capabilities.scala,laws/**}`, `model/temporal/laws/**` → `model/temporal/capabilities/**`, `model/lifter/Capabilities.scala`, each Model folder's new `Capabilities.scala`, capability lifter fixtures and refusals, `tools/umpire/internal/golden/{config.json,original.json}`, `model/README.md`, `.plans/SEMANTIC_PROTOCOLS.md`.
**Touches:** [model/umpire/**, model/temporal/**, model/lifter/**, tools/umpire/internal/golden/**, tools/umpire/model/**, model/README.md, .plans/SEMANTIC_PROTOCOLS.md]

### Approach
- Framework (`model/umpire`): a generic capability abstraction (a trait or marker that a capability kind extends, with its bound fields), `Law`, `Catalog`, `Brought`, `capabilities(...)`, `except`/`overriding`. It names no Temporal capability.
- Temporal (`model/temporal/capabilities`, renamed from `model/temporal/laws`): the six capability kinds and their bindings, every law with its server citations (including the two formerly entity-neutral ones) and the one `given Catalog`. `model/umpire/laws` keeps only the generic `Law`/`Catalog`/`Brought` types, or merges into `model/umpire`.
- Lifter: recognize capability kinds by the framework abstraction instead of the hard-coded six names; refuse a binding that is not a declared capability kind, with a fixture.
- Models: move each `capabilities(...)` declaration from `Properties.scala` into `<Model folder>/Capabilities.scala` (standaloneactivity, its `admission/` and `compositions/` subpackages, nexusoperation), keeping Definition IDs (DefinitionScope pins) and generated claim names `<machine>.<law>`.
- Behavior frozen: `model/ir/**` (including `*.laws.json`), `model/cases/**` and `lifts/expected/**` byte-identical apart from source positions and moved function symbols recorded in the golden config.
## Acceptance
- [ ] `model/umpire/**` holds no law and no server citation, and `model/umpire/**` and `model/lifter/**` name none of Closable, Terminable, Pausable, Cancelable, Pollable or Describable; a test fails if they reappear there (remove the capability allowlist entries fn-114.12's agnosticism guard carries, if it landed first).
- [ ] The capability kinds, their bindings, the Temporal laws and the catalog live in `model/temporal/capabilities/`; each Model folder that adopts capabilities declares them in its own `Capabilities.scala`.
- [ ] Generated claims, sidecars, IR and Cases are unchanged apart from positions and recorded symbol moves; the catalog test, lifter refusal fixtures, original-baseline and migration goldens, model gate, `lint-model`, Go tooling suite and `lint-code-fast` pass.
## Done summary
Moved the capability vocabulary and every law out of the framework. `model/umpire` now holds only the mechanism, and each adopting Model folder declares its capabilities in its own `Capabilities.scala`.

**What changed**
- **Framework (`model/umpire`).** It names no capability and no law.
  - `Capabilities.scala` defines `CapabilityOf` (no longer sealed, and with no `kind` member: the companion is the kind), `Waiver`, `Capabilities` and `capabilities`.
  - `Catalog.scala` (moved from `umpire/laws`) defines `CapabilityKind`, `Law`, `Brought` and `Catalog`. `Catalog.single`/`pair` are keyed by kinds.
  - `umpire/laws` is gone.
- **Temporal (`model/temporal/capabilities`, renamed from `model/temporal/laws`).**
  - `Capabilities.scala` holds the six bindings. Each companion extends `CapabilityKind`.
  - `Close.scala` holds `terminalStatesAreFinal` and `closedIsRejectedUniformly` with their server citations, moved from `umpire/laws/Laws.scala`. Their bodies stay the core form, so the IR is unchanged.
  - `Terminate`, `Cancel` and `Pause` are unchanged.
  - `Catalog.scala`: the one `given catalog` now lists every law, with no framework entries.
  - `Catalog.test.scala` moved with the folder.
- **Models.** Declarations moved from `Properties.scala` into `Capabilities.scala` in standaloneactivity, `admission/`, `compositions/` and nexusoperation:
  - `productCapabilities`, `protocolCapabilities`;
  - `admissionCapabilities` with `deliveryAfterClose`;
  - `overQueueCapabilities`, `overMatchingCapabilities` with `closedAnswer` and `queueStepsOn`;
  - `operationCapabilities` with `repeatedRequestsAnswer`.

  `closedRejectsOrRepeats` stays in nexusoperation's `Properties.scala`, as the Nexus operation's own claim, so the sidecar's `overriddenBy` symbol is unchanged.
- **Lifter.**
  - It recognizes a capability by the framework abstraction: a case class extending `CapabilityOf` whose companion extends `CapabilityKind`, which is also the kind the catalog keys by.
  - It finds a capability's action fields, path to a live state and Run expectation by their field types, not by the names terminate/pause/…/reach/expect.
  - It matches a declaration to the catalog by the kind companion's full name, as the runtime `Catalog` compares kind objects. Messages and the sidecar keep the simple name. The fixture `sameName` declares another kit's `Closable` and is brought none of Temporal's laws.
  - It refuses a binding whose companion is no kind (the new fixture `unkinded`, `rejects.txt`).
  - It reads no framework TASTy again: `Lift.scala`'s `umpire/laws` exception is reverted.
- **Guard.** `model/gate/test/CapabilityVocabulary.test.scala` fails if `model/umpire/**` or the lifter's sources (`model/lifter/*.scala`, `model/lifter/test/**`) name Closable, Terminable, Pausable, Cancelable, Pollable or Describable. The lifter's fixtures under `testdata`, which lift Temporal's laws, are not its sources.
- **Goldens.**
  - `config.json`: one source-path merge for `model/temporal/capabilities/` replaces the two for the old law paths.
  - `config.json`: `source_root_additions` names `Capabilities$package$.productCapabilities` and `protocolCapabilities`.
  - The parity test reads `Capabilities.scala` and the law files at their new paths.
- **Docs:** `model/README.md` and `.plans/SEMANTIC_PROTOCOLS.md`.

**Frozen behavior**
- `model/ir/**` changed only in positions and in the `source` labels' moved root names; the `*.laws.json` sidecars changed only in positions.
- `model/cases/**`: only the four generated Cases changed, in source paths and lines.
- `lifts/expected/**`: positions, plus the refusal lines for `unkinded` and `sameName`.

**Review:** round 1 gave SHIP with one P2 and two P3s. The P2 was simple-name kind matching; the P3s were the unread `CapabilityOf.kind` and Temporal field names in two lifter messages. All three are fixed in 70d9777324. Round 2: see the receipt.

**Decisions**
- **Started with `--force` ahead of fn-122.5.** Task 5's cross-spec gate, fn-120.3 (model lint and the accepted-findings file), is still todo. Task 8 reads nothing task 5 builds and keeps the sidecar format task 5 will read.
- **Kinds are the bindings' companions.** A separate enum would name Temporal's capabilities in the framework.
- **Guard scope:** the lifter's sources, not its testdata fixtures.
- **Coordination with fn-114.12:** the conductor's merge reconciles the allowlist entries fn-114.12's agnosticism guard carries for the capability files. This branch never touched `model/umpire/realize/**` or `model/temporal/realize/**`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3ffccca013, 70d9777324
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (idempotent; IR/sidecar/Case diffs positions and source labels only), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, incl. CapabilityVocabulary guard), go test -json -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 5853 pass), make lint-model (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), go test -tags test_dep ./tests/testcore/testpilot/ (exit 0), flowctl claude impl-review --spec claude:claude-opus-5-5:high (round 1 SHIP with P2+2xP3 fixed; round 2 SHIP)
- PRs: