---
satisfies: [R2, R6]
---
# fn-153-checked-property-examples-pilot.1 Add optional illustration carriers, admission and identity pins

## Description
Extend Property metadata and its reader admission. Freeze the predecessor's behavior and identity evidence before any pilot schema or declaration edit. This task leaves production Model declarations unchanged.

**Size:** M
**Files:** `proto/internal/temporal/server/api/umpire/v1/claim.proto`, generated `api/umpire/v1/claim*.go`, `tools/umpire/ir/validate.go`, `tools/umpire/ir/illustrations_test.go` (new), existing schema/admission identity fixtures
**Touches:** [proto/internal/temporal/server/api/umpire/v1/claim.proto, api/umpire/v1/claim*.go, tools/umpire/ir/validate.go, tools/umpire/ir/*_test.go, tools/umpire/ir/testdata/**]

### Approach
- Re-anchor against the integrated Batch 5 closure before coding. Record the actual schema, Property binding and export interfaces left by fn-141. Freeze complete transition/check answers, Definition IDs, Behavior Fingerprints and Case bytes in the task's ignored proof directory; pin sources and producer inputs. Reuse exact unchanged baseline receipts where permitted by MILESTONES.
- Extend the Property carrier at `claim.proto:17` with an optional ordered illustration collection. Reuse concrete finite Value, action-class and Position carriers; include label, explanation, expected satisfied/violated value, resulting state/outcome/facts and optional before-state. Avoid a second expression language or witness-search API. Generate Go mirrors through existing schema tooling.
- Add independent nested admission through `validate.go:1002`. Check actual declared state/outcome/fact/input domains, including nested values, rather than only structural Conforms. Validate per-Property labels, expectations, explanation, ownership and before-state shape from the parent API contract. Allow an individually owned Property on a refining machine; reject composed owners without claiming unsupported machine refinement.
- Compare omitted and empty illustration collections with the frozen canonical baseline. Metadata must not enter Property semantic identity, Query identity, table construction or Case generation. Use `internal/engine/canonical.go:249` and existing `PropertyOrigin` tests as patterns, not a weakened identity normalizer. Populated metadata may alter lifted carriers but not behavioral fingerprints.

### Investigation targets
**Required:**
- `proto/internal/temporal/server/api/umpire/v1/claim.proto:17` - Property and origin carrier precedent
- `tools/umpire/ir/validate.go:1002` - Property admission and resolved owners
- `tools/umpire/interp/machine.go:572` - structural conformance versus owning-domain membership
- `tools/umpire/internal/engine/canonical.go:249` - explicit behavioral canonicalization
- `tools/umpire/ir/identity_test.go` - independent identity test patterns
**Optional:**
- `tools/umpire/ir/origins_test.go` - inert located metadata
- `tools/umpire/ir/schema_test.go` - schema closure and wire tests

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/ir -run 'Property|Illustration|Origin|Identity|Schema'
```
Run the established proto generation and freshness check if the schema changes; preserve the descriptor ledger's existing contract.

## Acceptance
- [ ] Optional carriers round-trip concrete values and nested positions; absent and empty metadata preserve the original canonical bytes and identities.
- [ ] Located tests independently reject wrong owner/action, out-of-domain and malformed nested values, invalid facts/outcomes, missing or extraneous before-state, blank/duplicate labels, whitespace-only explanations and unspecified/unknown expectations.
- [ ] Individually owned refining-machine Properties remain eligible; composed owners reject explicitly.
- [ ] Frozen source/producer input pins and complete behavior/identity/Case baseline are available for the final original-to-pilot comparison; no candidate-generated baseline is substituted.
- [ ] Schema mirrors are fresh and focused reader/schema checks pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
