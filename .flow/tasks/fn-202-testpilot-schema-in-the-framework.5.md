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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
