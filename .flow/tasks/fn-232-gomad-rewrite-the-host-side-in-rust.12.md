---
satisfies: [R11, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.12 Keep Go overlay, models, generator and corpora green under the Rust binary

**Touches:** tools/gomad/Makefile, tools/gomad/internal/model/generation/protocol, tools/gomad/internal/archcheck/**

## Description
Ensure `make generate`/`make validate` keep emitting Rust codecs for the binary and Go codecs for the oracle; keep the conformance corpora (helper transcript, replay plan, namespace, volume, network, clock) running against the Rust library used by the binary; update archcheck so the Go host packages scheduled for removal are tracked separately from the retained Go reference set.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
`make validate` passes with the Rust binary present; a deliberate codec change fails validation; archcheck distinguishes retained Go sets from host packages slated for removal.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
