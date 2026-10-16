---
satisfies: [R6]
---
# fn-180-gomad-replay-time-instrumentation-for.5 Temporal dump and probe demonstrated on a retained failure artifact, docs

## Description
Register at least one Temporal dump (shard and mutable-state summaries) and one probe through `common/gomaddebug` behind gomad WASM build seams (default production/native behavior unchanged). Dumps cannot block on application-held shard/cache/workflow locks: use bounded immutable diagnostic views maintained at existing mutation seams, or prove direct nonblocking access safe with all app goroutines parked. Include freshness/boundary information and test inspection while the relevant application locks are held.

Record and retain a Temporal failure with the final dormant-instrumentation binary, then demonstrate exact stop-and-inspect, heap/stacks, registered summaries and enabled probe output from that artifact without rebuilding or mutating it. Document selectors, separate tape/trace coordinates, requested/resolved time/failure boundaries, `inspected_prefix` versus full failure reproduction, evidence/witness/old-artifact refusals and bounded scan versus conditional bisect. Run integrated qualification per the spec: host-source/lint/architecture and protocol validation, Rust/helper/model checks, full test-wasm and mandatory cooperative runtime qualification after changed runtime identities, then the retained Temporal demonstration. Retain one current-candidate batch handover mapping .1–.5/R1–R6 to identities, exact commands/exits and independent review; native execution remains deferred and no PR/push/CI is authorized.
## Acceptance
- [ ] Temporal dump + probe exist; default server build unchanged
- [ ] Demonstration on a retained Temporal failure artifact recorded in a .flow artifact
- [ ] README documents target selectors, actual stop resolution, evidence refusals and bounded scan/bisect semantics without claiming unconditional binary-search correctness
- [ ] Shard/mutable-state dump remains useful with application locks held; bounded view freshness is explicit and inspection cannot wait on frozen app goroutines
- [ ] Final dormant-instrumentation Temporal artifact is immutable and inspected from retained bytes with compiler/source unavailable; earlier artifacts are refused honestly
- [ ] Integrated current-candidate handover maps all R1–R6, runtime qualification, full WASM checks, source/lint/architecture/identity validation and independent review without claiming native execution
## Done summary
Implemented and qualified Temporal shard/mutable-state dumps and probes with bounded immutable snapshot views, lock-held regressions, freshness reporting and documented retained inspection/search contracts.

The unchanged original final7 three-case gate passed, exit 0, 3301.509 s: shard112.87s, workflow108.21s, onebox3080.41s. The real onebox failure occurred after workflow timers/activity retries completed. Source-free retained exact replay matched the canonical record hash; stopped inspection passed authenticated prefix/witness, useful dumps/probes, stacks/heap, immutable evidence and retained-artifact reopen assertions. All three actual heap profiles decoded with pinned go tool pprof, exit0. Source2012/helper postchecks passed, original processes reaped and source staging cleaned.

Current source e4f63ac483; manifest597b200d8f83acd6f5c9115d46e44c8fcc2dac2957efb771c4db424dfeb646da. Current full WASM, architecture, runtime800recordings/32replays and narrow repair reviews are retained in .flow/artifacts/fn180-current-batch-2026-10-16.md. Broad and repair source review receipts are SHIP; full fn180 completion review remains a separate required gate. Native execution remains deferred; no push/PR/CI.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e4f63ac4832241ab104bb68f92b09b5b7c2c719a, 69b31a401b, 90c1230140, 18882a5f2a
- Tests: Original session17232: go test -p=1 -v -tags test_dep -run ^TestTemporalInspectionRetainedFailure$ -count=1 -timeout=100m ./internal/backends/wasm/backend; exit0,3301.509s; /tmp/fn180-temporal-prefix-final7.log, make -C tools/gomad test-wasm; exit0; /tmp/gomad-fn180-prefix-qualification-7yB87N/test-wasm.log, Current cooperative runtime:800recordings+32exactreplays PASS; /tmp/gomad-fn180-prefix-qualification-7yB87N/runtime-acceptance.json, Pinned go tool pprof actual shard/workflow/onebox profiles exit0, sha256sum --check --quiet /tmp/gomad-fn180-prefix-qualification-7yB87N/source.sha256 exit0
- PRs: