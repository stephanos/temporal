---
satisfies: [R11]
---
# fn-93-simplify-the-lean-model.38 Authoring shorthand for evidence lines and limits (F1, F2)

## Description
Lane F1 and F2. An `evidence:` line may be a bare `x`, meaning `x: x` (syntax ~`Syntax.lean:1671`, handler ~1931-1932; 49 of 60 lines); `limits` `actions:` defaults to `steps:` (~640-645; 20 of 22 blocks). Rewrite Models and tests to use them, and update the quoted AUTHORING.md regions in the same commit.

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean`, `model/Umpire/Command/Tests/**` (a `#guard` per shorthand comparing records with the long form), Models/tests with evidence lines (`model/Temporal/Feature/Nexus/{Caller,Control,Pair}/Model.lean`, `model/Temporal/Feature/Nexus/Tests/Machines.lean`, `model/Temporal/Feature/Workflow/{Start,Outage}/Model.lean`, `model/Temporal/Feature/System/Info/Model.lean`, `model/Temporal/Feature/Nexus/Success/Tests.lean`), limits blocks (22, incl. `Success/RaceSyntaxTests.lean:108` and `Success/Tests.lean:614` which differ and keep `actions:`), `model/AUTHORING.md` (`product`/`protocol` blocks ~292-296, 498-503; `queries` ~672-684; prose ~198, 818, 825)
**Touches:** [model/Umpire/Command/Syntax.lean, model/Umpire/Command/Tests/**, model/Temporal/Feature/**, model/AUTHORING.md]

### Approach
- `tools/umpire/authoring/drift_test.go` requires AUTHORING.md regions to be byte-equal to `-- authoring:` regions of `Caller/Model.lean`; change both in one commit.
- Definition IDs and Fingerprints are the oracle; a `#guard` per shorthand compares elaborated records.
- A bare evidence name that is not an observation fails exactly as the long form does.

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean:635-650,1665-1680,1925-1940`
- `tools/umpire/authoring/drift_test.go`
- `model/Temporal/Feature/Nexus/Caller/Model.lean` (`-- authoring:` regions)

### Quick commands
```sh
cd model && lake build
go test ./tools/umpire/authoring/...
make umpire-check-goldens umpire-check-regression
```

## Acceptance
- [ ] Both shorthands elaborate to records equal to the long forms (`#guard` each)
- [ ] Models and AUTHORING.md use them; drift test green
- [ ] Definition IDs, Fingerprints, fixtures byte-identical


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
