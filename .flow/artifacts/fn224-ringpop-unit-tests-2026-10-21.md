# fn-224.2 — ringpop unit tests under the cooperative profile

Candidate: branch `fn224-ringpop` at `3925d83499` (fn-224.1 unstub). Raw evidence: `/home/agent/wt/evidence/fn224/ringpop/` (commands, explore JSON, retained successes, replay outputs).

## Identities

| Input | Identity |
| --- | --- |
| gomad binary (fn-221 sweep build of `1d7b37f00a`) | `sha256:407263f9be00…` |
| Stock compiler go1.27.1 | `sha256:1675694ef690…` |
| Helper | `sha256:b25439140f0c…` |
| Model | `sha256:73c1a4f3f64d…` |
| Installation | `gomad install --temporal-checkout <fn224> --temporal-package ./common/membership/ringpop --memory-bytes 4GiB --fuel 2^50` (derived profile `gomad.wasi-cooperative/v1`, NetworkV2 v3, `ReadStepNanos=0`, tags `disable_grpc_modules,gomad_wasm_slim,sqlite3_dotlk,test_dep`) |

Early proof point: `^TestRpoSuite$/^TestCompareMembers$` seed 0 prepared and passed; no refused import.

## Classification (seeds 0, 1, 2; `--choices --keep-successes=all`)

The spec's "eleven tests" are two suite wrappers plus nine leaf methods; every leaf ran by exact selection. Each retained success's stdout contains the leaf's own `--- PASS` line.

| Selection | Seeds | Outcome | Replay |
| --- | --- | --- | --- |
| `TestRpoSuite/TestCompareMembers` | 0,1,2 | pass ×3 | exact ×3 |
| `TestRpoSuite/TestCompareMembersWithDraining` | 0,1,2 | pass ×3 | exact ×3 |
| `TestRpoSuite/TestMonitor` | 0,1,2 | pass ×3 | exact ×3 |
| `TestRpoSuite/TestScheduledUpdates` | 0,1,2 | pass ×3 | exact ×3 |
| `TestRingpopSuite/TestHostsMode` | 0,1,2 | pass ×3 | exact ×3 |
| `TestRingpopSuite/TestInvalidBroadcastAddress` | 0,1,2 | pass ×3 | exact ×3 |
| `TestRingpopSuite/TestRingpopMutualTLS` | 0,1,2 | pass ×3 | exact ×3 |
| `TestRingpopSuite/TestRingpopServerTLS` | 0,1,2 | pass ×3 | exact ×3 |
| `TestRingpopSuite/TestRingpopInvalidTLS` | 0,1,2 | pass ×3 | exact ×3 |

27/27 pass; 27/27 `gomad replay` report `reproduced=true diagnostic=false result=success choice-replay=exact`. No refused import, capacity, timeout or guest failure.

## Stalls

None. No harness change and no `ReadStepNanos` declaration (NetworkV2 admission requires 0 anyway). `TestMonitor` logs one ringpop bootstrap retry ("join duration of 3.408s exceeded max 2s") in virtual time and then passes; it is ringpop's own retry path, not a stall.

## Boundary (R4)

No change to gomad models, config, helper protocol or testcore; this task adds only this artifact.

## Rerun

`/home/agent/wt/evidence/fn224/ringpop/run.sh <label> 0,1,2 '<selector>'`, `all.sh` for the nine selections, `replay-all.sh` for the replays.
