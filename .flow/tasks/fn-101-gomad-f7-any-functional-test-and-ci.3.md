---
satisfies: [R5]
---
# fn-101-gomad-f7-any-functional-test-and-ci.3 Pin host-clock references statically on both platforms

## Description
Amended 2026-09-29 (user decision): no dynamic Linux audit. Add a toolchain-tier test that inventories every
standard-library reference to the host clock in the patched GOROOT for each qualified platform against a
reviewed, classified allowlist, plus an AST check that the clock entry points test `gomadEnabled` first.
Escapes it finds are recorded with findings, not fixed here.

## Acceptance
- `make test-toolchain` runs the inventory on darwin/arm64 and linux/amd64; a mutated count fails it
- the host-tools job (stock Go, no toolchain) skips it

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
