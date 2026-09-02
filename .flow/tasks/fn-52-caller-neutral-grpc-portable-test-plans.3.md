---
satisfies: [R3, R6, R10]
---
# fn-52-caller-neutral-grpc-portable-test-plans.3 Enforce provenance and result claim scopes

## Description
Implement the external versus model-compiled provenance seam and make result authority explicit for R3 and R6. The plan carries bindings, while the executor host supplies independent trust verification.

**Size:** M
**Files:** `tools/umpire/testplan/**`, `tools/umpire/executor/**`, focused provenance fixtures/tests
**Touches:** [tools/umpire/testplan/**, tools/umpire/executor/**]

### Approach
- Add a narrow host-injected provenance verifier; the semantic plan cannot supply trust anchors or credentials.
- Admit external plans directly as plan-local and require exact verified bindings for model-bound scope.
- Reject missing, invalid, expired, unsupported, or crossed requested model provenance before runtime I/O; never downgrade it silently.
- Carry validated provenance, claim scope, Known Gaps, and unresolved external obligations into every result.
- Ensure required external obligations prevent complete model-bound success without pretending the executor performed them.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/executor/executor.go:48-143` — resident admission and result lifecycle
- `tools/umpire/evaluationcontract/validate.go` — binding validation patterns
- `proto/internal/temporal/server/api/umpire/v1/message.proto:328-343` — current model-bound contract fields
- `.flow/specs/fn-29-bounded-production-canary-execution-and.md:193-231` — canary provenance and claim separation

**Optional** (reference as needed):
- `tools/umpire/artifact/set.go` — artifact identity and relation checks

### Acceptance
- [ ] Any client can submit a valid external plan and receive only plan-local scope.
- [ ] Only independently validated exact model provenance permits model-bound scope.
- [ ] Forgery, trust-anchor injection, expiry, mismatch, missing verifier, and downgrade mutations fail before I/O.
- [ ] Required and advisory external obligations have distinct deterministic effects and remain visible in results.
- [ ] Provenance validation and result-scope mutation tests pass with `-tags test_dep`.

## Acceptance
- [ ] R3 caller-neutral authority and R6 provenance safety are complete.
- [ ] Claim scope and obligations cannot be forged, dropped, or silently downgraded.
- [ ] Focused tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
