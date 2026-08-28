---
satisfies: [R1, R6, R7]
---
# fn-39-make-the-temporal-nexus-feature-model.5 Add the Nexus reading facade and align documentation

## Description
Establish the ordinary Nexus newcomer entry point, enforce its non-Experimental boundary, and align all maintained learning/architecture documentation with the completed internal decomposition (R1, R6, R7). Finish with the full compatibility and lint gates.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus.lean`, `model/Temporal/Feature/NexusTests.lean`, `model/Temporal/Feature.lean`, `model/TemporalModelTests.lean`, `model/ModelLint/ImportGraph.lean`, `model/ModelLint/ImportGraphTests.lean`, `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean`, `model/README.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_SPEC_COMPS.md`
**Touches:** [model/Temporal/Feature/Nexus.lean, model/Temporal/Feature/NexusTests.lean, model/Temporal/Feature.lean, model/TemporalModelTests.lean, model/ModelLint/ImportGraph.lean, model/ModelLint/ImportGraphTests.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean, model/README.md, model/ARCHITECTURE.md, .plans/UMPIRE4_SPEC_COMPS.md]

### Approach
- Add `Temporal.Feature.Nexus` as the documented ordinary aggregate and have `Temporal.Feature` consume it without exposing Experimental modules.
- Add a facade-only smoke test whose sole import is `Temporal.Feature.Nexus` and which exercises representative public Lifecycle, Operations, and Observation declarations; include it in `TemporalModelTests`.
- Extend the pure model import-graph policy with an exact ordinary-Nexus-facade rule that rejects direct or transitive reachability to `Temporal.Feature.Nexus.Experimental`, plus deterministic direct/transitive policy tests.
- Add root-facade navigation that links to the module/read-next guidance owned by Tasks 1 and 2, preserving all pre-existing comments.
- Add a module-level overview and section map to CallerClosure without moving declarations or changing its imports, namespace, source, semantics, or artifact.
- Update the model README learning path, architecture dependency/semantic-interface maps, and Umpire component decomposition to distinguish stable facades from internal physical files.
- Keep Observation's current description/path, leave the dated cleanup design as historical, and do not edit generated regression views.
- Run the focused build, complete regression gate, and model lint from the final tree; audit `git diff` against the declared scope and pre-existing dirty files.

### Investigation targets
**Required** (read before coding):
- `model/README.md:68-123` — current Switch-to-Nexus learning sequence and facade description.
- `model/ARCHITECTURE.md:37-119` — current import/dependency and semantic-interface guidance.
- `.plans/UMPIRE4_SPEC_COMPS.md:382-416` — logical family template and Nexus decomposition guidance.
- `model/Temporal/Feature.lean:1-3` — current ordinary Feature aggregate.
- `model/TemporalModelTests.lean:1-25` — ordinary model test aggregate.
- `model/ModelLint/ImportGraph.lean:17-197` — pure import policy and forbidden-rule dispatch.
- `model/ModelLint/ImportGraphTests.lean:120-182` — direct/transitive import-policy test patterns.
- `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean:1-50` — advanced module currently lacking an overview.

**Optional** (reference as needed):
- `docs/superpowers/specs/2026-08-26-nexus-lifecycle-cleanup-design.md:127-142` — historical ordinary/Experimental facade decision.
- `model/Temporal/Tool/Generated/Regressions.md:1-15` — generated ownership marker; inspect only.

### Acceptance
- [ ] `Temporal.Feature.Nexus` is the documented ordinary entry import and excludes Experimental modules.
- [ ] `Temporal.Feature.NexusTests` imports only the facade, exercises representative Lifecycle/Operations/Observation declarations, and is included by `TemporalModelTests`.
- [ ] Import policy and its focused tests reject both direct and transitive paths from the exact ordinary Nexus facade to the Experimental prefix while leaving explicit Experimental entry points usable.
- [ ] Maintained docs teach one consistent reading order and accurately distinguish facades, internal implementation modules, Observation, System, and advanced Experimental material.
- [ ] CallerClosure has a concise navigation map with no physical, semantic, identity, provenance, or artifact change.
- [ ] `cd model && mise exec -- lake build Temporal.Feature.NexusTests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests TemporalExperimentalTests modelLintTests` passes.
- [ ] `make umpire-check-regression` and `make lint-model` pass, generated views and unrelated dirty files are unchanged, and the final diff matches declared touches.

## Acceptance
- [ ] R1, R6, and R7 task-scoped checks pass.
- [ ] No out-of-scope Observation, Experimental structure, generated output, runtime, or authoring-language change is present.

## Done summary
Added the single ordinary `Temporal.Feature.Nexus` entry facade, a facade-only smoke test, exact direct/transitive Experimental import isolation, and a simple-first documentation path through Lifecycle, the three operations, Observation, System correspondence, and advanced Experimental navigation. AutoClose, Observation, generated views, canonical fixtures, public identities, artifacts, and runtime behavior remain unchanged; all final acceptance, regression, and model-lint gates pass.

baseline: red (`cd model && mise exec -- lake build Temporal.Feature.NexusTests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests` failed pre-edit because the task-created `Temporal.Feature.NexusTests` target did not yet exist); pre-edit regression and model lint were green

stage: impl-review - ran [2026-08-28T23:02:39Z..2026-08-28T23:05:54Z] (SHIP)
## Evidence
- Commits: 03aa539f6d6ecc23bbb759b19b7f1be6a6d649ea
- Tests: baseline: red (cd model && mise exec -- lake build Temporal.Feature.NexusTests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests failed pre-edit because Temporal.Feature.NexusTests did not yet exist); make umpire-check-regression and make lint-model were green pre-edit, cd model && mise exec -- lake build Temporal.Feature.NexusTests modelLintTests, cd model && mise exec -- lake exe modelLintTests, cd model && mise exec -- lake build Temporal.Feature.NexusTests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests TemporalExperimentalTests modelLintTests, cd model && mise exec -- lake build Temporal.Feature.NexusTests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.ImplementationLinkTests.Nexus TemporalModelTests, make umpire-check-regression, make lint-model, exact scope/protected-file audit: 10 declared paths only; AutoClose, Observation, generated regressions, six ordinary canonical fixtures, and caller-closure fixture SHA-256 values unchanged; no Nexus Examples directory; inherited symlink targets equal indexed blobs, GREEN_RECEIPT:unittest:03aa539f, GREEN_RECEIPT:smoke:03aa539f
- PRs: