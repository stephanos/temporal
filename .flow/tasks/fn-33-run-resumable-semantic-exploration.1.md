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
- Register the fixed `umpire-exploration-bridge` sibling and implement the exact single 4-byte big-endian length plus canonical JSON request/response, 16 MiB frame, eight-item batch, 30-second timeout, two-second terminate/kill/reap, 64 KiB stderr, and handshake/error contract.
- Keep Go-visible metadata limited to admission, bindings, leases, and transport limits.
- Add strict `umpire-campaign-checkpoint/v1` codecs and ArtifactSet relationships that bind—but do not alter—the fn-18 coverage checkpoint, environment input set, leases/attempts/results, parent/generation, versions, bounds, omissions, and provenance.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Exploration.lean` — fn-17 public state/protocol after completion
- `tools/umpire/artifact` — fn-18 admitted set boundary after completion
- `tools/common/artifactio/set.go:475-645` — atomic set publication/recovery pattern

### Acceptance
- [ ] Cross-language frame/envelope goldens pin every field, byte/frame/batch/stderr limit, handshake, exit, timeout, cancellation, and reaping row.
- [ ] Campaign-checkpoint closure binds coverage state/report, input set, artifacts, leases/attempts, parent/generation, bounds, versions, omissions, and provenance with one-at-a-time relationship mutations.
- [ ] Go cannot inspect semantic coverage/corpus contents.
## Acceptance
- [ ] R1/R4 cross-language golden and mutation fixtures pass.
- [ ] Unknown/stale/crossed state rejects before work.
- [ ] No second artifact reader or publisher is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
