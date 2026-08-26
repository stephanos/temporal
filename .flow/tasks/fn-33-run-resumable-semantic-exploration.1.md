---
satisfies: [R1, R4]
---
# fn-33-run-resumable-semantic-exploration.1 Freeze the exploration bridge and checkpoint closure

## Description
Define the bounded Go/Lean campaign protocol and fn-18 checkpoint relationships for R1/R4.

**Size:** M
**Files:** `model/Umpire/Exploration/**`, `model/Temporal/Tool/ExplorationBridge.lean`, `model/Temporal/Tool/ExplorationBridgeTests.lean`, `model/lakefile.toml`, `tools/umpire/campaign/**`, `tools/umpire/artifact/**`
**Touches:** [model/Umpire/Exploration/**, model/Temporal/Tool/ExplorationBridge.lean, model/Temporal/Tool/ExplorationBridgeTests.lean, model/lakefile.toml, tools/umpire/campaign/**, tools/umpire/artifact/**]

### Approach
- Register the fixed `umpire-exploration-bridge` sibling and implement one canonical compact-JSON request/response per invocation behind a four-byte unsigned big-endian length.
- Cap `initialize` and `next-batch` request/response frames at 16 MiB. Cap `observe` requests at 72 MiB and responses at 16 MiB. Every batch caps at eight; each ExperimentSpec remains at most 1 MiB and each full fn-18-admitted Result at most 8 MiB.
- Pin 30-second invocation timeout, two-second terminate/kill/reap, 64 KiB sanitized stderr, handshake/error behavior, and independent N/N+1 item/member/aggregate-frame fixtures for every operation.
- Let the Lean bridge validate each full admitted Result closure, compute one opaque checked admission identity plus reproduction-tuple digest, and construct fn-17's domain-neutral `ExplorationObservation/v1`; Go-visible metadata stays limited to admission, bindings, leases, and transport limits and never supplies semantic/evidence vocabulary, coverage, corpus, priority, or mutation feedback.
- Add strict `umpire-campaign-checkpoint/v1` codecs and ArtifactSet relationships that bind—but do not alter—the fn-18 coverage checkpoint, environment input set, leases/attempts/results, parent/generation, versions, bounds, omissions, and provenance.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Exploration.lean` — fn-17 public state/protocol after completion
- `tools/umpire/artifact` — fn-18 admitted set boundary after completion
- `tools/common/artifactio/set.go:475-645` — atomic set publication/recovery pattern
## Acceptance
- [ ] Cross-language frame/envelope goldens pin every field, operation-specific byte limit, item/member limit, stderr limit, handshake, exit, timeout, cancellation, and reaping row.
- [ ] Eight maximum-size Results fit the 72 MiB observe request; the ninth, a Result over 8 MiB, and an aggregate over 72 MiB each reject before a state transition.
- [ ] Initialize/next-batch N/N+1 fixtures enforce 16 MiB frames, eight items, and the 1 MiB ExperimentSpec member ceiling.
- [ ] Campaign-checkpoint closure binds coverage state/report, input set, artifacts, leases/attempts, parent/generation, bounds, versions, omissions, and provenance with one-at-a-time relationship mutations.
- [ ] Go cannot inspect semantic coverage/corpus contents or derive semantic observations.
- [ ] R1/R4 cross-language golden and mutation fixtures pass; unknown/stale/crossed state rejects before work; no second artifact reader or publisher is introduced.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
