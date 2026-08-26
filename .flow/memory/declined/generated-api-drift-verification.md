# Generated API drift verification

Do not add generated Lean API drift verification or GitHub Actions coverage in the current simplification.

The generator is intentionally generation-only for now. Its focused and golden tests remain, while a repository or CI drift gate is deferred until there is a demonstrated need.

## Prior requests

- 2026-08-24 — While simplifying the Lean API generator, explicitly excluded drift verification and all CI workflow work.
- 2026-08-24 — Requested planning the Umpire Temporal dynamic-config design, then confirmed `make umpire-check-dynamic-config` and all CI workflow changes remain excluded.
- 2026-08-26 — Planned the Lean test-suite decomposition follow-up and kept generated API drift verification, CI coverage, and generated-file changes out of scope.
- 2026-08-25 — Requested the next C5 Go-test/documentation projection spec; generated Lean API drift verification and GitHub Actions coverage remain out of scope.
