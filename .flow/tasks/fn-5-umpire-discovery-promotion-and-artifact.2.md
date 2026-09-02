---
satisfies: [R1]
---
# fn-5-umpire-discovery-promotion-and-artifact.2 Add deterministic Nexus list output

## Description
Project the closed Nexus inventory into one deterministic `list` result and expose it through the
existing inspector executable.

**Size:** S
**Files:** `model/Temporal/Tool/NexusDiscovery.lean`, `model/Temporal/Tool/NexusDiscoveryTests.lean`, `model/Temporal/Tool/Inspect.lean`, `model/Temporal/Tool/InspectTests.lean`
**Touches:** [model/Temporal/Tool/NexusDiscovery.lean, model/Temporal/Tool/NexusDiscoveryTests.lean, model/Temporal/Tool/Inspect.lean, model/Temporal/Tool/InspectTests.lean]

### Approach

- Add a pure list projection over the checked inventory; do not introduce selectors, filtering,
  pagination, aliases, or registry mutation.
- Emit one canonical `umpire-nexus-discovery/v1` JSON value with the four query identities and their
  checked declaration/source/fingerprint/plan summaries in canonical order.
- Route `temporal-model-inspect list` through the pure projection while preserving the existing
  positional scenario invocation byte-for-byte.
- Pin stdout, stderr, LF, status, field order, and authoring-order permutation behavior in focused tests.

### Non-goals

- No generic semantic graph, generated glossary, machine index, complete export command, or broad regression set.

### Investigation targets

**Required:**
- `model/Temporal/Tool/NexusDiscovery.lean` — task `.1` inventory and validation boundary.
- `model/Temporal/Tool/Inspect.lean` — current effect-thin command shell.
- `model/Temporal/Tool/InspectTests.lean` — canonical output/error conventions.
- `model/lakefile.toml` — existing `temporal-model-inspect` registration.

### Quick command

`cd model && mise exec -- lake build Temporal.Tool.NexusDiscoveryTests Temporal.Tool.InspectTests temporal-model-inspect`

## Acceptance
- [ ] `list` emits exactly one canonical four-row JSON value plus one LF, empty stderr, and status 0.
- [ ] Reordered valid inventory input yields byte-identical list output.
- [ ] Invalid inventory state yields empty stdout, one structured diagnostic plus one LF, and status 1.
- [ ] No filters, paging, aliases, output modes, or alternate inventory inputs are exposed.
- [ ] Existing positional scenario inspection outputs and failures remain byte-identical.
- [ ] Existing comments in touched files are preserved.

## Done summary
Added the deterministic `umpire-nexus-discovery/v1` list projection over the checked Nexus inventory and routed only the exact `temporal-model-inspect list` argument through it. Each canonical row exposes the checked Property, Behavior, and Query declaration identity, kind, source, and fingerprint plus the planned ExperimentSpec format and checksum. The existing positional inspector runner and its success and failure bytes remain unchanged.

Focused tests pin canonical field and row order, permutation stability, invalid-inventory failure with no partial stdout, exact LF/stderr/status behavior, rejection of alternate list arguments, and compatibility of the existing positional path. The exact task suite, real executable invocation, full Lean model lint, and diff check pass.

stage: impl-review - ran (Codex SHIP; 0 introduced and 0 pre-existing findings)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 34e8e56192e18951ed9e29fb13c7f0a5946112fa
- Tests: BASELINE_GREEN: cd model && mise exec -- lake build Temporal.Tool.NexusDiscoveryTests Temporal.Tool.InspectTests temporal-model-inspect, cd model && mise exec -- lake build Temporal.Tool.NexusDiscoveryTests Temporal.Tool.InspectTests temporal-model-inspect (95 jobs), cd model && mise exec -- lake -q exe temporal-model-inspect list (status 0; one LF-terminated v1 JSON line; four entries; empty stderr), make lint-model (234 build jobs plus 205 lint jobs), git diff --check, flowctl codex impl-review fn-5-umpire-discovery-promotion-and-artifact.2 --base 8d6176bc5 --receipt /tmp/impl-review-receipt-fn-5-umpire-discovery-promotion-and-artifact.2.json (SHIP)
- PRs: