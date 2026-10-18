---
satisfies: [R4, R5, R6, R7]
---
# fn-167-gaps-declared-in-the-model-gap-replaces.6 Held sweep: gap-aware lint matching, inheritance and expiry

## Description
Held sweep: consume gap/provenance metadata in the Go lint (R5–R7), preserving every current kind's judgment. Owner authorization remains required.

**Size:** M
**Files:** lint finding/acceptance pipeline, lint CLI and read-only Flow status adapter, gate seam/tests.
**Touches:** [tools/umpire/lint/**, tools/umpire/cmd/umpire-lint/**, tools/umpire/ir/**, model/check/**]

### Approach
- Compute findings with existing predicates first, then match the transported canonical keys. Preserve optional-kind admission and source positions in errors.
- Preserve the existing `capability-waiver` stale-check exemption: validate synthesized metadata against its capability declaration, not against a nonexistent lint finding. Include an `except` and an `overriding` regression; no waiver annotation constructor or new lint kind is introduced.
- Establish inheritance suppression per kind with base/derived controls, including changed assumptions, refinement, phases and reachability. Unsupported equivalence is an explicit diagnostic/design finding, never hidden by equal binding identities.
- Replace source-feature acceptance reads with gap matching and explicit legacy-file compatibility. Prove a source feature containing an acceptance file fails while legacy payloads remain readable.
- Resolve expiry from authoritative Flow state, including committed spec records and applicable shared runtime overrides. Make missing/ambiguous lookup behavior deterministic and test status transitions without remote writes or a new status service.
- Finish the staged capability-waiver consumer/copy-path retirement from task 5.

### Investigation targets
Required:
- `tools/umpire/lint/lint.go:28` — finding pipeline and optional kind.
- `tools/umpire/lint/accept.go:88` — existing key matching and stale entries.
- `tools/umpire/lint/holes.go:711` — table/Property/Query subjects.
- `tools/umpire/lint/kinds.go:149` — aggregate evidence findings.
- `tools/umpire/lint/bindings.go:91` — realization-owned findings.
- `model/check/Gate.scala:1255` — current waiver copy seam.

### Verification
Use paired base/derived fixtures per supported kind; compare raw finding sets before suppression and accepted/unaccepted results afterwards. Include changed-context negative controls, stale gap positions, expired/dangling/ambiguous expiry, legacy retention, source-file refusal and waiver matching. Run `go test -count=1 -tags test_dep ./tools/umpire/lint ./tools/umpire/cmd/umpire-lint` and relevant Scala gate tests. Record R5's unconfirmed cases rather than declaring all kinds proven from a single fixture.
## Acceptance
- [ ] Exact key matching, stale/expiry/refusal/legacy controls and focused Go/Scala suites pass.
- [ ] Raw findings retain each existing kind's judgments; only proven inherited duplicates suppress and changed-context controls remain visible.
- [ ] Authoritative Flow lookup is tested and the staged waiver copy/file seam is retired safely.

## Done summary
Superseded by fn-200 (one-language series, 2026-10-18): native lint judges @gap

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: