# fn-27-hermetic-ci-execution-and-qualification.7 Run the bounded CI portability proof

## Description
Run one exact v2 Nexus Artifact through the disposable CI runner and shared Run Evaluation boundary, then compare its stable semantic meaning with the local result. Cover success, semantic non-success, cancellation, timeout, Limit N/N+1, and cleanup failure without remote or production authority.

## Acceptance
- [ ] The valid CI run proves byte-identical input and equal stable Run Evaluation meaning.
- [ ] Negative outcomes remain inspectable and never become portable-success claims.
- [ ] All resources are closed and the proof is deterministic and retry-safe.

## Done summary
Implemented the pinned hermetic CI qualification path with byte-identical local/CI input proof, direct typed Result semantic parity, retry determinism, and six explicit actual-boundary outcomes: Limit N success, semantic violation, cancellation, timeout, N+1 rejection, and cleanup failure with zero leaked handles. Final Quick gates are green; nonmutating diff lint reports zero task issues and only the inherited `tools/umpire/runtime/errors.go:60` errortype warning, and review returned SHIP after fixing the real-boundary gap and withdrawing the transport-sensitive Lean checksum finding.

stage: impl-review - ran [2026-08-31T22:35:17Z..2026-08-31T23:02:26Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5baaf0c0f2bd5595510f189cb9a1db637b046934, 861d6e024af03cbbc690aafc71f17e69a2888dfe
- Tests: cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests, GOFLAGS=-tags=test_dep CC=$(xcrun --find clang) CXX=$(xcrun --find clang++) SDKROOT=$(xcrun --show-sdk-path) TMPDIR=$(pwd -P)/.flow/tmp/go-tmp mise exec -- go test -count=1 ./tools/umpire/runtime/... ./tools/umpire/runevaluation/... ./tools/umpire/temporal/nexus/..., GOFLAGS=-tags=test_dep CC=$(xcrun --find clang) CXX=$(xcrun --find clang++) SDKROOT=$(xcrun --show-sdk-path) TMPDIR=$(pwd -P)/.flow/tmp/go-tmp mise exec -- go test -count=1 ./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$', GOFLAGS=-tags=test_dep CC=$(xcrun --find clang) CXX=$(xcrun --find clang++) SDKROOT=$(xcrun --show-sdk-path) TMPDIR=$(pwd -P)/.flow/tmp/go-tmp mise exec -- make umpire-check-regression, GOFLAGS=-tags=test_dep CC=$(xcrun --find clang) CXX=$(xcrun --find clang++) SDKROOT=$(xcrun --show-sdk-path) TMPDIR=$(pwd -P)/.flow/tmp/go-tmp mise exec -- go test -count=1 ./tools/umpire/cmd/umpire-gen-tests-go
- PRs:
