---
satisfies: [R11]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.12 Keep Go overlay, models, generator and corpora green under the Rust binary

Touches: tools/gomad/Makefile, internal/model/generation/protocol, archcheck

## Description
Ensure `make generate`/`make validate` keep emitting Rust codecs for the binary and Go codecs for the oracle; keep the conformance corpora (helper transcript, replay plan, namespace, volume, network, clock) running against the Rust library used by the binary; update archcheck so the Go host packages scheduled for removal are tracked separately from the retained Go reference set.

## Acceptance
`make validate` passes with the Rust binary present; a deliberate codec change fails validation; archcheck distinguishes retained Go sets from host packages slated for removal.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
