# Gomad F4: close the ./tests capability closure on darwin/arm64

## Goal & Context
<!-- scope: business -->

Milestone F4 of `.plans/GOMAD_MILESTONES.md`. On linux/amd64 the closure of `./tests` under the
`gomad` build tag reports zero blockers. The acceptance is stated for darwin/arm64, where the
darwin packs pin a stale profile digest and cannot be regenerated from Linux, and the darwin
prepared source-set pins of the fx, SDK, and otel adapters were computed rather than observed.

## Architecture & Data Models
<!-- scope: technical -->

Compatibility packs follow discover, review, `generate --approve-review`, check, qualify. The
darwin packs (`temporal-functional-compute-darwin-arm64`, `modernc-libc-xsys-v047` darwin, and a
new darwin counterpart of `temporal-functional-tests-linux-amd64` if the analyzer demands one) are
regenerated on this host. Adapter prepared source-set pins for darwin are confirmed or corrected
from a darwin review.

## Edge Cases & Constraints
<!-- scope: technical -->

- Closure mode is the support claim; linked mode is only evidence.
- A pack names the workload it unlocks; no pack admits `os/exec`, `os/signal`, or `os/user`.
- Exact facts only; the darwin pack admits only what the darwin analyzer reports.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `gomad analyze --capability-mode=closure --format=json` with build tags
  `disable_grpc_modules,gomad,test_dep` on `go-test ./tests` reports zero `unsupported_target`
  findings on darwin/arm64.
- **R2:** Every pack is listed where the qualification set needs it and
  `make -C tools/gomad3 compatibility-pack-qualification` passes on darwin/arm64; `make validate`
  passes.
- **R3:** The darwin adapter source-set pins are observed on darwin (confirmed or corrected).
- **R4:** The eleven unsupported leaf cases in `temporal.json` flip to `qualified` or carry a
  blocker that is not a forbidden import on darwin/arm64 (the manifest records the darwin first
  blocker evidence).
- **R5:** The server builds and the stock functional suite sample still passes with seams at
  their defaults (`go build ./...`, the named stock suite).

## Boundaries
<!-- scope: business -->

- Qualifying workflow-executing suites (F5, F6).

## Decision Context
<!-- scope: both -->

Discovery reviews the host platform, so darwin packs can only be produced here.
