# fn-104-comparative-go-implementation-of-the.3 T2 Framework core: finite domains, machines, tables, ids, fingerprints

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
Framework core in experiments/umpire-go/umpire: finite domains by reflection (Values(), Bool, struct product with the last field fastest, sealed-interface sums via umpire.Sum), Lean-exact keys, actions with typed inputs and functional options, machines with generic-method Step0..Step3 bindings, tables in Lean order, Definition IDs, typed examples validated at declaration. Declaration errors are recorded and reported, never panicked. Deviation: Behavior Fingerprints are deferred to T8, where the Case producer needs them; nothing before T8 reads them. Unit tests cover domain order, sum and unenumerable-type rejection, table order, out-of-domain, duplicate binding, missing start, bad example type.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: go test -count=1 -tags test_dep ./experiments/umpire-go/..., golangci-lint v2.13.1 --config=.github/.golangci.yml (0 issues), go-check-sumtype, exhaustive -default-signifies-exhaustive=false on model packages (clean)
- PRs: