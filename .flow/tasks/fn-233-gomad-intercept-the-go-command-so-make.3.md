---
satisfies: [R1, R5, R8]
---
# fn-233-gomad-intercept-the-go-command-so-make.3 gomad <command> wrapper, gomad.toml and auto-derived installation

## Description
Touches: gomad/src/cli/wrapper.rs, gomad/src/config/repo.rs, gomad/src/install/derive_cache.rs, golden corpus.
gomad [--installation FILE] [--seeds N|LIST] [--artifacts DIR] -- <command> [args]: everything after the first -- is the child command line and is never interpreted by Gomad; a bare gomad make without -- is the existing unknown-verb error with a hint to the -- form; resolve the command on PATH, create a private 0700 shim directory under the cache holding go as the gomad binary, prepend it to PATH, export the six GOMAD_SHIM_* contract variables, exec with inherited stdio, return the child status, remove the directory (and any stale one on start). No verb name forwards to a child; gomad -- go ... is the only Go spelling. Read gomad.toml at the module root with exactly the keys runtime_root, derive (temporal|default) and guest; refuse unknown keys, non-module-relative runtime_root, absolute or escaping guest entries. Without an explicit installation, derive one through the install derivation code for the compiler on the original PATH, the configured runtime root, derivation and the package being built; cache it keyed by compiler digest, runtime identity, derivation and package; never download; explicit installation always wins and is never rewritten.

## Acceptance
gomad -- true and gomad -- false return 0 and 1; gomad -- replay runs a child named replay, not the verb; gomad make without -- is the unknown-verb error with the hint; empty command after -- is the usage error; unknown command and shim-directory failures use the existing statuses; from a clean checkout with gomad.toml derive = temporal, gomad -- go test ./tests/gomadfunctional -run ^TestFrontendSystemInfo$ passes with retained evidence and the derived manifest configuration bytes equal install --temporal-checkout/--temporal-package for the same inputs; second run reuses the cached manifest; each gomad.toml refusal has a golden case; byte-identical second regeneration.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
