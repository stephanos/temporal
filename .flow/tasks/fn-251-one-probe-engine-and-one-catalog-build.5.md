---
satisfies: [R5, R7]
---
# fn-251-one-probe-engine-and-one-catalog-build.5 Generated annotation dispatch and AnnotationRuntime::with_probes

## Description
Implement **Transport dispatch (B8)**: generate `annotate_<svc>(..)` and add `AnnotationRuntime::with_probes(rt, probe_fn, annotate_fn)`; replace `kettle/transport.rs:13-61`, `temporal/feature/activity/api_transport.rs:19-58`, `temporal/realization/src/annotations.rs:25-40`.

## Acceptance
- [ ] One dispatch path; unknown prefix or method refuses as before; Go host tests pass.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
