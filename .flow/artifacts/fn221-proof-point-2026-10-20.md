# fn-221 preliminary proof point

Observation on one candidate only. These wrapper-level results are not a support or qualification claim and do not satisfy the complete method-level sweep.

The frozen candidate was `95f2d9695b61e2312f087de3f4fdd6982f9529aa`, with a clean tracked worktree at both inventory headers. The retained sidecars bind compiler, helper, launcher, harness, model, runtime, engine and guest configuration identities. Both runs used the throwaway configuration, seed 0, one guest at a time, a 5-minute execution timeout, a 20-minute preparation allowance, a 30-minute overall admission budget per package and a 2-GiB exemplar bound.

| Selection | Parsed / go-test-list wrappers | Observed pass | Not run | Sweep / verify exits |
| --- | ---: | ---: | ---: | --- |
| `./tests/gomadfunctional` | 1 / 1 | 1 | 0 | 0 / 0 |
| `./tests`, cap 10 wrappers | 148 / 148 | 1 | 147 | 0 / 0 |

`TestFrontendSystemInfo` passed in 294.19 seconds, including cold preparation and engine compilation. `TestAcquireShardSuite` passed in 308.14 seconds. The root package then lacked the harness's full 25-minute reservation for another test; nine more selected wrappers were journaled as not run, and the remaining 138 wrappers were excluded by the cap. This does not establish that ten functions ran.

Both package fixtures started. The spec's stop-before-full-sweep condition was therefore not triggered. The original harness inventories top-level functions only: its root wrapper count omits receiver methods, even though a wrapper may execute them. A source audit subsequently found 1,296 receiver test method declarations under the selected host file set. Corrected enumeration and a fresh method-level proof are required before the full sweep.

The original harness also reserves budgets without propagating an overall context deadline, and its preparation cache is cleared per function. These preliminary results must not be relabeled as evidence for either correction. The later harness propagates the invocation deadline and records budget cutoffs as not run; compilation cache reuse and method inventory are being verified separately.

The host was `dev-build`, Linux/aarch64, kernel `7.0.14`, 12 CPUs and 21,242,953,728 bytes of RAM. Other work ran concurrently; initial load average was 3.89. Go and Cargo concurrency were capped at two. These wall times are execution observations with host-load limitations and supply no performance comparison or qualification evidence.

Machine-readable records: [gomadfunctional](fn221-proof-point-gomadfunctional-2026-10-20.json), [root wrappers](fn221-proof-point-root-2026-10-20.json). They retain the original `gomad-wasm.sweep/v1` shape; the corrected method inventory uses v2.

Raw commands, build logs, digests, host measurements, JSONL journals and the original executable are retained at `/home/agent/wt/evidence/fn221/proof-point/20261020T152622Z`. The runner is `/home/agent/wt/evidence/fn221/proof-point.sh`. Both package invocations and their explicit verifier invocations returned 0; the script returned 0. Timestamp-specific scratch was removed after evidence retention. `QUALIFICATION.md` was not changed.
