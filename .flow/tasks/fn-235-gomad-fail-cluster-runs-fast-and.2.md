# fn-235-gomad-fail-cluster-runs-fast-and.2 Measure and remove superlinear World inspection commitment cost in long cluster runs

## Description
Found in fn-224.3 (DIAGNOSIS.md): a three-node ringpop scenario slows from ~3 s wall per virtual second toward ~8 (84 s virtual at 5 min, 132 s at 15 min). Suspected site: guest debug exchange per scheduling decision (runtime_inspect_wasm.go.txt:99) makes the helper rebuild the World inspection commitment (cluster/handoff/broker.rs:1227) by hashing full history; the cache is per World revision and every brokered operation bumps it. One perf sample (of the sweep helper, not this scenario) showed 74% SHA-256 there. First measure on the fn-224 ringpop scenario (wall per virtual second at 10/60/120 s, profile with a private copy of the helper and perf --no-buildid-cache), then compute the commitment incrementally or only when inspect reads it. Commitment bytes and replay identity must not change unless a new identity is declared. Performance claims follow fn-185 measurement rules.

## Acceptance
- [ ] Retained measurement of wall per virtual second over a long three-node run, before and after, same host and window
- [ ] Profile attributes the cost to a named site before any change
- [ ] Wall per virtual second stays flat (within measured spread) across the run after the fix; commitment and replay identities unchanged or explicitly rotated
- [ ] Golden scenario corpora stay green

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
