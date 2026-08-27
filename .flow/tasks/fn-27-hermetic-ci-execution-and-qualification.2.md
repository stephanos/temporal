# fn-27-hermetic-ci-execution-and-qualification.2 Bind generated Go tests to the disposable CI runner

## Description
Run the generated ordinary Go test through the existing invocation-owned runner using fixed loopback bindings, concurrency one, declared phase and Evidence Limits, and deterministic participant/environment cleanup. Accept no external endpoint, credential, arbitrary executable, plugin, or undeclared network authority.

## Acceptance
- [ ] The generated test uses the shared runner and one bounded disposable loopback environment.
- [ ] Cancellation and every failure path kill/reap participants and prove cleanup.
- [ ] Limit N/N+1 and authority-leak fixtures fail at the runner boundary.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
