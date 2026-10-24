---
satisfies: [R3, R5]
---
# fn-191-gomad-sometimes-assertions-and-a-site.3 Typed semantic record section, catalog identity binding, inspect output, legacy replay test

## Description
gomad-evidence: add the semantic section to execution-record v2 as a typed struct with deny_unknown_fields (catalog hash, per-incarnation sorted hit IDs, completeness, Fail hits with ID, ordinal, virtual time, symbolized frames through evidence/symbolize). Bind the catalog hash into prepared-target identity; replay and resume refuse a mismatch at preparation. inspect prints the section; inspect --choices marks decisions adjacent to a Fail hit. Supported-lineage artifacts without the section report 'no catalog'. Retain an authenticated pre-change gomad executable and matching inputs; the legacy replay test runs the old artifact on it while the new executable refuses the incompatible identity. Depends on .2.

## Acceptance
Record goldens for the section; catalog mismatch refused at preparation with a named reason; inspect output goldens including adjacent-decision marks; 'no catalog' on a pre-change artifact; legacy replay test passes on the retained executable and the new executable refuses; no identity check bypassed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
