---
satisfies: [R3, R4, R5, R7, R8]
---
# fn-18-versioned-umpire-artifact-boundary.8 Admit complete artifact sets with exact closure

## Description
### Umpire4 reconciliation (normative)

Complete-set closure includes the complete current `ExperimentSpec`, `ParticipantProgram`, and the reserved replay, verification, and qualification receipt envelopes when present. Legacy v1 may enter only through strict compatibility admission or a named complete migration; set admission never infers missing executable meaning.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Implement R7's exact manifest and cross-document closure as one inert admitted-set boundary, separate from migration and filesystem publication.

**Size:** M
**Files:** `model/Umpire/Artifact/Set.lean`, `model/Umpire/Artifact/Tests/Set.lean`, `tools/umpire/artifact/set.go`, `tools/umpire/artifact/set_test.go`
**Touches:** [model/Umpire/Artifact/Set.lean, model/Umpire/Artifact/Tests/Set.lean, tools/umpire/artifact/set.go, tools/umpire/artifact/set_test.go]

### Approach
- Define and encode the exact manifest member/relationship shapes and identity view.
- Admit every member through its strict family decoder one at a time, then validate all artifact-only relationship kinds, member uniqueness, binding/path/digest agreement, and exact required closure.
- Resolve query/Property references only against ExperimentSpec fields and program/mapping/profile/capability references only against RuntimeConfiguration fields; never look for semantic pseudo-artifacts.
- Reject missing, extra, duplicate, stale, mixed-version, unresolved-semantic-reference, and cross-boundary-inconsistent sets atomically.
- Add valid full-chain/report-checkpoint sets and one-at-a-time member, relationship, path, digest, semantic-reference, and closure mutations.

### Investigation targets
**Required** (read before coding):
- Tasks `.4`–`.7` typed artifact codecs
- parent spec `Normative v1 wire contract` ArtifactSet and `Artifact Sets, Migrations, and Publication`
- fn-4 embedded semantic identities and fn-17 report/checkpoint bindings

### Acceptance
- [ ] Exact manifest and member bytes round-trip cross-language.
- [ ] Every required artifact relationship and embedded semantic reference is closed exactly once.
- [ ] Query, Property, Observation-program, and mapping are never treated as standalone artifact families.
- [ ] Set validation is inert, bounded, fetch-free, and independent of publication.
## Acceptance
- [ ] R7 set admission and R3–R5 cross-artifact closure are implemented.
- [ ] Full-chain positive and negative set suites pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
