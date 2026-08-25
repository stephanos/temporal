---
satisfies: [R5, R6, R7]
---
# fn-8-umpire-temporal-dynamic-config.5 Implement typed ConfigView authoring and resolution

## Description
Build the handwritten model-authoring boundary for R5, R6, and Lean parity in R7. Keep the public interface small enough that semantic models never consume raw overrides or arbitrary string lookup.

**Size:** M
**Files:** `model/Temporal/Experiment/Config.lean`, `model/Temporal/Experiment/ConfigTests.lean`, `model/Temporal/ExperimentTests.lean`
**Touches:** [model/Temporal/Experiment/Config.lean, model/Temporal/Experiment/ConfigTests.lean, model/Temporal/ExperimentTests.lean]

### Approach
- Define classifications, typed interpretations, config uses, resolution contexts, sampling points, change effects, structured diagnostics, resolved-entry provenance, and immutable use-keyed views following existing `Except` and semantic-digest conventions.
- Author only the representative real-setting classifications/interpretations from the spec. Bind opaque-default replacements to the imported default identity; reject stale schema/default metadata before decoding.
- Implement the parent spec's binding callback-address interpretation contract: exact special Temporal URLs, HTTP/HTTPS plus host validation, whole-host wildcards, secure-by-default rules, and explicit insecure permission. Malformed canonical rules fail interpretation; do not import the Go converter's raw-entry skipping behavior.
- Validate the complete use and override set before resolution. Build each policy's exact ordered constraints and interleave overrides with constrained defaults at every level; reject duplicate exact constraints and incomplete/illegal contexts rather than choosing by input order.
- Require checked `ConfigUse α` values for typed reads. Include catalog and interpretation digests plus source/matched constraints/context in resolved entries so same-key/multi-consumer views remain distinct and reproducible.
- Consume every valid generated Go parity fixture in Lean tests. Test duplicate rejection separately as Lean structural validation, not as production resolver parity, and add negative/mutation cases for errors, order independence, opaque defaults, typed reads, and view immutability.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Experiment/Semantics.lean:145-250` — structured validation/error conventions
- `model/Temporal/Experiment/Semantics.lean:433-620` — canonical identity and digest patterns
- `common/dynamicconfig/collection.go:314-377` — exact policy/default ordering authority
- `chasm/lib/callback/config.go:71-180` — custom callback-address conversion/validation boundary
- `model/Temporal/ExperimentTests.lean:1-9` — focused test-root imports

**Optional** (reference as needed):
- `common/dynamicconfig/constants.go:1665-1671` — restart-required global example
- `common/dynamicconfig/constants.go:1292-1308` — constrained-default task-queue example

### Key context
The public YAML documentation's constraint list is stale; mirror current repository source. Invalid canonical values and duplicate exact constraints are explicit Lean errors, not production resolver parity or Go converter fallback, because raw/YAML conversion is outside this boundary.

### Quick commands
```bash
cd model && mise exec -- lake build ExperimentTests
cd model && mise exec -- lake build
```
## Acceptance
- [ ] Representative generated keys have explicit non-empty classifications, checked typed interpretations, use identities, sampling points, and change effects; unclassified catalog entries remain allowed but unusable by models.
- [ ] Callback-address interpretation implements the parent spec's exact special-URL, wildcard, scheme/host, and insecure-connection contract; malformed canonical rules fail before model execution.
- [ ] Validation returns stable structured errors for every R5/R6 negative case before planning or transition enumeration.
- [ ] Resolution exactly matches all eight policies and constrained-default interleaving, is source-order independent, and produces immutable use-keyed entries with complete provenance/digests.
- [ ] Duplicate exact constraints are tested as Lean structural-validation failures outside the production Go resolver parity set.
- [ ] Typed reads require the originating checked use and support the same key in multiple contexts/consumers without collision.
- [ ] Every valid generated Go fixture and all Lean-specific negative/order/opaque-default cases pass through the focused test root.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
