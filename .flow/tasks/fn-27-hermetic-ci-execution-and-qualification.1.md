---
satisfies: [R1, R6]
---
# fn-27-hermetic-ci-execution-and-qualification.1 Pin the byte-identical v2 Artifact for ordinary CI tests

## Description
Freeze the exact canonical v2 `ExperimentSpec` already used by the local Nexus path as the sole CI semantic input. Generate an ordinary Go test from that admitted Artifact without recompiling or reconstructing its definitions. Check the exact bytes, format version, Artifact Checksum, Definition IDs, Behavior Fingerprints, Limits, Known Gaps, query, Properties, Observation program, and Implementation Link before runtime IO.

## Acceptance
- [ ] CI and local tests consume byte-identical canonical v2 Artifact bytes.
- [ ] One-byte, version, checksum, fingerprint, closure, or generated-output drift fails before runtime IO.
- [ ] No CI Evaluation Profile, provenance schema, or semantic copy is introduced.

## Done summary
Made the aggregate regression gate portable across Darwin's logical and physical temporary roots. Generated an inspectable Run Evaluation subject pin from the exact canonical ExperimentSpec bytes and reject byte, version, checksum, fingerprint, closure, or generated-output drift before runtime I/O.

All task and spec Quick commands pass. Diff-scoped lint and vet pass; the repository-wide `make lint-code` remains inherited red with 1,375 unrelated findings.

stage: impl-review - ran [2026-08-31T09:02:36-0700..2026-08-31T16:10:01.594624Z]
## Evidence
- Commits: 5b74f05150b7094e642b666c2cd097d6428192ee, b211253a91be3223529ce69089dc94c905248426
- Tests: cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests, mise exec -- go test -count=1 ./tools/umpire/runtime/... ./tools/umpire/runevaluation/... ./tools/umpire/temporal/nexus/..., mise exec -- go test -count=1 ./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$', mise exec -- make umpire-check-regression, mise exec -- .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --new-from-rev=HEAD --config=.github/.golangci.yml, mise exec -- go vet -tags test_dep ./tools/umpire/runevaluation ./tools/umpire/cmd/umpire-gen-tests-go ./tools/umpire/temporal/nexus
- PRs: