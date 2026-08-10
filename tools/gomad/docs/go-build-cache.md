# Gomad Go Build Cache Isolation

## Goal

Make Gomad use a dedicated Go build cache without requiring users to know about
or configure `GOCACHE`. Preserve an explicitly supplied `GOCACHE` so CI and
advanced users can control cache placement.

## Current Behavior

Gomad has a translation cache at `.gomad/cache.sqlite3`, generated source under
`.gomad/translated`, and prebuilt metatest binaries under `.gomad/metatest`.
Translation-time package loading and Gomad-owned `go test`, `go test -c`, and
Delve commands inherit the caller's ambient Go build cache.

Sharing the cache is correct because Go keys build artifacts by their complete
build inputs, and translated packages use distinct import paths. It nevertheless
mixes native and simulated build artifacts, makes cache ownership unclear, and
prevents users from removing all Gomad state by removing `.gomad`.

## Design

Before a command performs translation or compilation, the CLI configures the Go
build cache as follows:

1. If `GOCACHE` exists in the process environment, leave it unchanged.
2. Otherwise, find the current Go module root and set `GOCACHE` to the absolute
   path `<module>/.gomad/go-build`.
3. Let package loading and child processes inherit the resulting environment.

The configuration applies to `translate`, `test`, `build-tests`, `debug`, and
`prepare-selftest`. Commands that do not perform Go build work, including help,
must continue to work outside a Go module.

The CLI owns this policy at one boundary. Individual package-loading and command
execution sites do not accept or construct separate cache settings. This avoids
missing a build path when the CLI gains another Go subprocess.

`GOMODCACHE` remains shared. Module downloads are immutable, checksum-verified
inputs and separating them would add download time and disk usage without
separating generated Gomad artifacts.

## Bootstrap Behavior

When Gomad is invoked with `go run`, the Go command must compile the CLI before
the CLI can configure its environment. That ordinary CLI compilation may use the
caller's normal cache. Once the CLI starts, all translation-time and translated
build work uses the dedicated cache.

Complete bootstrap isolation would require a launcher or an environment setting
outside the process. That extra installation surface is not justified because
the bootstrap compiles ordinary Gomad CLI code, not translated application code.

## Errors and Overrides

An explicitly supplied `GOCACHE` is authoritative, including invalid values; the
Go command will report invalid paths in its normal form. Gomad does not silently
replace an explicit setting.

When `GOCACHE` is absent, failure to find the module root is reported before a
build-oriented command starts. The computed default is absolute, as required by
the Go command. Directory creation remains the Go command's responsibility.

## Testing

Unit tests cover:

- absent `GOCACHE` selects `<module>/.gomad/go-build`;
- an explicit `GOCACHE` is preserved exactly;
- the generated default is absolute;
- a child Go process observes the configured value; and
- failure to locate a module is returned clearly.

Command-level coverage verifies that help does not require a module, while each
build-oriented command configures the cache before its first package load or
subprocess.

## Trade-offs and Failure Modes

The isolated cache duplicates native standard-library and dependency artifacts
that Gomad could otherwise reuse. This increases the first-run cost and disk
usage in exchange for clear ownership, predictable cleanup, and isolation from
normal development builds. Subsequent runs retain normal content-addressed Go
cache performance.

Parallel Gomad processes may share the module-local cache safely because the Go
build cache supports concurrent access. A tenfold increase in Gomad invocations
increases cache traffic and disk use but does not introduce a new coordination
mechanism or correctness dependency. Go's normal cache eviction behavior remains
in effect when commands use this cache.

Crashes can leave unused cache entries but cannot corrupt source or translated
artifacts through this policy. The cache contains derived, reproducible data and
can be discarded with the rest of `.gomad`.

The change does not expand the trust boundary: cache contents remain local Go
build artifacts, explicit environment configuration remains authoritative, and
no new network access or third-party dependency is introduced.
