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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
