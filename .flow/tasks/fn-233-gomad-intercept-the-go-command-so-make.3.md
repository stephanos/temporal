---
satisfies: [R1, R5, R8]
---
# fn-233-gomad-intercept-the-go-command-so-make.3 gomad -- <command> wrapper, gomad.toml and auto-derived installation

## Description
**Touches:** tools/gomad/rust/gomad/src/cli/wrapper.rs, tools/gomad/rust/gomad/src/config/repo.rs, tools/gomad/rust/gomad/src/install/derive_cache.rs, tools/gomad/testdata/parity/**.
gomad [--installation FILE] [--seeds N|LIST] [--artifacts DIR] -- <command> [args]: everything after the first -- is the child command line and is never interpreted by Gomad; a bare gomad make without -- is the existing unknown-verb error with a hint to the -- form; discover the real compiler on the original PATH, create a private 0700 shim directory under the cache holding go as the gomad binary, prepend it to PATH, export the expanded GOMAD_SHIM_* contract variables, resolve the child command on the modified PATH (direct go must hit the shim), execute with inherited stdio, return the child status, remove the directory (and any stale one on start). No verb name forwards to a child; gomad -- go ... is the only Go spelling. Read gomad.toml at the module root with exactly the keys runtime_root, derive (temporal|default) and guest; refuse unknown keys, non-module-relative runtime_root, absolute or escaping guest entries. Without an explicit installation, derive one through the install derivation code for the compiler on the original PATH, the configured runtime root, derivation and the package being built; cache it keyed by compiler/runtime identities, derivation, package, complete derived configuration digest and linked host/helper implementation identities; never download; explicit installation always wins and is never rewritten.

## Acceptance
Use the complete derived configuration digest including captured schema bytes and current linked host/helper identities in auto-cache admission. Derive/validate required inputs before lookup. Cover schema mutation, missing schema after caching, changed executable, unchanged reuse and concurrent invocation isolation under the expanded metadata contract.

Direct gomad -- go test hits the shim and captures an intercepted campaign rather than launching the host compiler directly; gomad -- true and gomad -- false return 0 and 1; gomad -- replay runs a child named replay, not the verb; gomad make without -- is the unknown-verb error with the hint; empty command after -- is the usage error; unknown command and shim-directory failures use the existing statuses; from a clean checkout with gomad.toml derive = temporal, gomad -- go test ./tests/gomadfunctional -run ^TestFrontendSystemInfo$ passes with retained evidence and the derived manifest configuration bytes equal install --temporal-checkout/--temporal-package for the same inputs; second run reuses the cached manifest; each gomad.toml refusal has a golden case; byte-identical second regeneration.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
