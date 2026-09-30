# Gomad deferred follow-ups

[fn-105](../.flow/specs/fn-105-gomad-follow-ups-deferred-scope.md) owns the tasks and
acceptance criteria. Each item below records why it is deferred and what would
revive it. A trigger must be recorded before implementation; items may remain
open or close as won't-do.

## Architecture consolidation

| Item | Scope and origin | Revival trigger |
| --- | --- | --- |
| D1 | Shared completed-execution assessment across seed, choice, and simulation (`fn-102` R2) | A second consumer or strategy hits the duplication |
| D2 | Shared retention and artifact-input composition, with separate strategy transactions (`fn-102` R3) | A second consumer or strategy hits the duplication |
| D3 | Private executor injection instead of public `Executor`/`ReplayExecutor` (`fn-102` R4); a Go API change | A second consumer or strategy needs the seam |
| D4 | Architecture fitness checks for ownership, host effects, and public signatures, with negative fixtures (`fn-102` R5) | A second consumer or strategy exposes the boundary problem |
| D5 | Architecture/platform/determinism documentation reconciliation (`fn-102` R6) | A second consumer or strategy requires the evidence |

These were deferred as maintenance without a waiting consumer or behavior change.
The [architecture assessment](../.flow/artifacts/fn-102-gomad-architecture-consolidate/architecture-assessment.md)
retains the evidence. Preserve CLI behavior, schemas, canonical bytes, failure
classification and precedence, and replay compatibility when reviving them.

## Capability and CI follow-ups

| Item | Scope and origin | Why deferred | Revival trigger |
| --- | --- | --- | --- |
| D6 | `seeded` and `fixed=<d>` clock ticks, manifest settings and qualified fixtures (`fn-103`) | `forward` addresses known ties; the extra policies are exploration features | A bug class needs deliberate ties or constant quanta |
| D7 | macOS functional smoke job (`fn-101.4`) | Linux supplies the smoke gate and macOS already runs Temporal integration | A darwin-only regression escapes to main |
| D8 | Downstream closure-mode adapter for the signal-handling metrics library (`fn-104` C3/R2) | Linked mode removes the import | A downstream module needs closure-mode preparation or manifests |
| D9 | linux/amd64 downstream packs and qualification (`fn-104`) | The downstream measurement is darwin/arm64 | A downstream gate must run in Linux CI |
| D10 | Downstream seam guide (`fn-104` R4) | Analyzer findings already name the sites | A second downstream module adopts Gomad |
| D11 | Dynamic Linux clock audit with disabled vDSO, seccomp denial, and positive control (`fn-101.3`, pre-amendment R5) | Static inventories cover both platforms; darwin DTrace exercises interception | A linux-only host-clock escape is observed |
| D12 | Identify the linux/amd64 host-timing channel behind the intermittent replay divergence (`fn-106.1`) | About one tier-3 seed-run in 26 diverges on either seed; reverting the FIPS DRBG draw or the mark-start greying did not remove it, and a Rosetta container reproduces it only under load, so each fork iteration costs an hour | A linux/amd64 host is available for buffered per-event runtime logging, or the rate rises |

The constraints in [GOMAD_MILESTONES.md](GOMAD_MILESTONES.md) apply throughout.
New tick policies carry execution identity and the
[COMPAT-5 evidence set](GOMAD3_NEXT_COMPATIBILITY.md#compat-5-targeted-deterministic-adapters-and-io-models).
Downstream packs/adapters bind exact versions; dependency drift reopens their
qualification. On revival, use the origin spec's requirement text as acceptance.
