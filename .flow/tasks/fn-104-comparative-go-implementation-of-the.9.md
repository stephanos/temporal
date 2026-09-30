# fn-104-comparative-go-implementation-of-the.9 T8 Go Case producer with byte parity for the seven Nexus fixtures

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
caseproducer/ is the Go counterpart of Umpire/Case/Producer.lean with its compile, correlated lowering, projection check and local-name passes; nexuscaller/realization.go ports Temporal/Case/Realization/Nexus.lean. umpire/canonical.go and umpire/lower.go port the canonical encoders behind every Behavior Fingerprint and the predicate lowering (Umpire.Command.enumerateSameStep).

Result: all seven nexusCallerTests-*-case.json fixtures are produced byte-identical in persisted form (nexuscaller/case_test.go), including all four Definition fingerprints, the correlated Property fingerprint, the projection fingerprint, the Program, the correlated Contract and the local-name table. Each produced Case also decodes strictly and prepares, unchanged, under a Profile DeriveProfile derives from it (the environment must name the handler's own task queue).

Parity checks along the way (parity/canonical_test.go): the 590 KB target semantic string, and for all seven Queries the scenario semantic, the lowered witness Property semantic and the Query canonical form, byte for byte against Lean's canonical metadata; 22 fingerprints equal.

Scope notes: the port covers what the Nexus caller set exercises (one instance, same-step Properties, no field relations, no outage-order rule, no structural model-value keys). Those paths reject by name rather than guess. The Model's Lean source path is reused in provenance so bytes can match.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: go test -tags test_dep ./experiments/umpire-go/nexuscaller/ -run 'TestCasesAreByteIdenticalToTheFixtures|TestProducedCasesPrepareUnderTheirDerivedProfile', experiments/umpire-go/run.sh
- PRs: