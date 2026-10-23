---
satisfies: [R4]
---
# fn-228-runner-and-run-the-one-go-host-loop.4 After .3: Temporal Driver as act-and-observe; live activity slice through umpire explore

## Description
Reduce the retained temporal/* driver to a runner.Driver: send the proposed request over the generated API, collect Observations with poll interval, read timeout and backoff from instruction fields, capture and redaction as field paths, RPC completion and evidence well-formedness checks; surface SDK callbacks from the in-process Go worker. Run the live activity slice (start, retry then completion, cancellation, host refusal) through umpire explore with the Run as the only artifact; make umpire-check-model-slice invokes it. Read timeout without Model acceptance is a DriverFault event; malformed evidence refused before Judge.

### Required integration contract

Integrate the incremental Exchange contract and Model-owned continue_polling signal with the real Temporal Driver. Start uses bootstrap and same-invocation World admission from fn-227; callbacks are serviced while the RPC/poll is active. Prove retry/completion/cancellation and an intermediate read with intervening callback before completion. Record the real Run and replay all protocol events without a live server. Preserve guest embedding/freshness and keep poll timing/backoff as transport mechanics.

### Carried from fn-228.3 (2026-10-23)

fn-228.3 retired four live tests that built the deleted commands; see `.flow/tmp/fn-228.3/retired-coverage.md`. This task owns their live replacements: `umpire explore` run as a separate process with the Temporal Driver, which exits 0 for an Explained Run and leaves the Run as the only artifact; resource provisioning and cleanup in the Temporal Driver; a live control where an unreachable endpoint is a DriverFault and exits 2; and a real callback-bearing Run recorded and then replayed without a server.
## Acceptance
- [ ] Temporal Driver implements runner.Driver without interpreting Temporal messages
- [ ] Live slice passes through umpire explore; Run is the only artifact
- [ ] DriverFault and malformed-evidence controls

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
