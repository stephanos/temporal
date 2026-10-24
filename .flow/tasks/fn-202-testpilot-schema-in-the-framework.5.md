---
satisfies: [R5]
---
# fn-202-testpilot-schema-in-the-framework.5 ConformanceDescriptor proto

## Description
Add the generic descriptor message: interned states, rows keyed by state and class with ordered results (target, outcome, facts, choice), hole marks, per-result claim bitsets, monitor automata (states, next table, read set, violated set), progress and end marks, fairness classes, abstract evidence roles (correlation keys, ordinals, closing events). Every field commented; attached to Case as an optional field.

**Files:** model/framework/proto/testpilot/v1/descriptor.proto, case.proto, Makefile TESTPILOT_PROTOCOL_PROTOS.

**Quick commands:** `make umpire-check-testpilot-protocol`

## Acceptance
- Comment check passes; Go and Scala bindings generate.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: