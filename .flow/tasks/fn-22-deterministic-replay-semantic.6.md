---
satisfies: [R3, R4, R5, R6, R7]
---

# fn-22-deterministic-replay-semantic.6 Prove the Case-native negative control and evidence core
## Description
Recompile the fn-21 duplicate-observation control into one generic fn-64 Case with no scenario-specific Go path. Prove two matching violated Runs, complete reduction or irreducibility, and a diagnostic EvidenceCore that omits one labeled non-responsible Observation while leaving the source Run/Verdict unchanged.

**Size:** L
**Touches:** `model/Temporal/Feature/Nexus/Experimental/**`, `tools/umpire/replay/integration_test.go`, `tests/umpire_replay_test.go`

## Acceptance
- [ ] The negative Case uses only public Program instructions, Contract machines, and Temporal Host capabilities.
- [ ] Repeated Runs preserve the semantic violation key and are isolated.
- [ ] EvidenceCore omission is proved without rewriting events, Run, Verdict, or Contract.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
