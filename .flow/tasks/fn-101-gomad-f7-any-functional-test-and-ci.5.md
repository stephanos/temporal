---
satisfies: [R3, R4]
---
# fn-101-gomad-f7-any-functional-test-and-ci.5 Make the gomad3 workflow pass on a clean checkout (linux and macOS runners)

## Description
Fork CI run 36453715669 (workflow_dispatch on stephanos/temporal gomad at d6a6d5d-ish) failed three jobs on clean runners: (1) host-tools-linux tests nine tools/gomad3/internal/* packages that no longer exist (moved) — update the step to the current package set; (2) core-linux: TestRunInterceptionExecutesManifestDrivenCompilerCases fails 'create interception test workspace: stat .../tools/gomad3/.toolchain: no such file or directory' when test-harness runs before any toolchain build — the test must create what it needs (or the tier must order correctly) without relying on local state; (3) core (macos-15): compatibility-pack-qualification fails 'resolve pinned go.opentelemetry.io/otel/sdk module: lstat /Users/runner/go/pkg/mod/go.opentelemetry.io: no such file or directory' — adapter/pack resolution must download pinned modules itself (e.g. go mod download of the exact pinned version, checksum-verified) or the Makefile must ensure it, so a clean checkout works. Reproduce each locally as far as possible (fresh GOMODCACHE/ fresh clone in scratchpad for (3); remove .toolchain in a scratch clone for (2)), fix, then dispatch the fork workflow (`gh workflow run gomad3.yml --repo stephanos/temporal --ref gomad` after the conductor pushes) and iterate until the jobs that can pass on hosted runners pass. Record anything that cannot run on hosted runners (e.g. root DTrace).

## Acceptance
- a fork workflow_dispatch run of gomad3.yml on the branch passes host-tools-linux, core-linux, core (macOS), and temporal-integration, or each remaining failure is recorded with cause

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
