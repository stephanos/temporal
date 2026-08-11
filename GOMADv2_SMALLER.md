# Making Gomad Smaller

## Scope

This review covers handwritten code under `tools/gomadv2`. Generated translation
output under `tools/gomadv2/.gomad` is excluded. The goal is to remove locally
maintained mechanisms when an already-used library provides the same semantics,
without weakening Gomad's determinism or crash-simulation behavior.

## Priorities

| Priority | Area | Existing library | Estimated reduction | Risk |
| --- | --- | --- | ---: | --- |
| 1 | Translation cache and hashing | `github.com/rogpeppe/go-internal/cache` | 180-220 lines including tests | Low-medium |
| 2 | Package graph traversal | `golang.org/x/tools/go/packages` | 15-25 lines | Low |
| 3 | Console log formatting | `log/slog` `TextHandler` | 300-400 lines | Medium |
| 4 | Simulated testing implementation | Go's standard `testing` package | More than 1,000 lines | High |
| 5 | Per-key initialization and parallel work | `github.com/rogpeppe/go-internal/par` | 10-70 lines | Low-medium |
| 6 | Checksum hashing | `hash/fnv` | 20-30 lines | Low-medium |

The cache replacement is the best first change. The logging and testing changes
offer larger reductions, but they also affect user-visible behavior or core
simulation semantics.

## 1. Replace the SQLite translation cache

The current cache is split across:

- `tools/gomadv2/internal/translate/cache/cache.go`, which implements a SQLite
  key/value store, locking, access timestamps, and eviction;
- `tools/gomadv2/internal/translate/cache.go`, which implements content hashing
  and adapts translated package results to the cache; and
- `github.com/mattn/go-sqlite3`, which makes the Gomad tool itself depend on
  CGO.

`github.com/rogpeppe/go-internal` is already a direct dependency. Its `cache`
package provides the needed operations:

- `cache.Open` for a filesystem-backed cache;
- `Cache.GetBytes` and `Cache.PutBytes` for translated results;
- `Cache.Trim` for access-based eviction;
- `cache.NewHash` for action IDs; and
- `cache.FileHash` for source and executable inputs.

The translated result can remain gob-encoded and gzip-compressed. Cache keys can
remain fixed-size arrays instead of being converted to hexadecimal strings.
`GOMADCACHE` would identify a cache directory such as `.gomad/cache` rather than
a SQLite file.

This removes the custom database package, most of the custom hasher, its database
tests, and the SQLite dependency. It also gives the translation cache explicit
cross-process coordination without maintaining another locking layer.

### Compatibility and failure handling

- Treat missing or corrupt entries as cache misses and regenerate them.
- Return permission, disk, and other operational errors instead of silently
  treating every error as a miss.
- Expect one cold run because the on-disk format changes.
- Decide whether the library's five-day retention policy should replace the
  current seven-day policy.
- Verify concurrent Gomad processes sharing `GOMADCACHE`, plus termination
  during `PutBytes`; the cache is derived data and must remain safely disposable.

## 2. Use `packages.Visit` for import graphs

`tools/gomadv2/internal/translate/main.go` contains two recursive import-graph
walkers:

- `collectImports`; and
- `findAllDepFiles`.

Both can use `golang.org/x/tools/go/packages.Visit`, which already handles
deduplication. `collectImports` still needs its skip predicate and deterministic
sort, while `findAllDepFiles` still needs root filtering and sorting. This is a
small reduction, but it removes generic graph-walking code with little behavioral
risk.

`packages.PrintErrors` could also replace the package-error printing loop, but
only if printing errors from the entire dependency graph is intended. That is a
behavioral change rather than a purely mechanical reduction.

## 3. Format console logs at the `slog.Handler` boundary

`tools/gomadv2/internal/prettylog/prettylog.go` is a 415-line JSON-to-console
formatter based on zerolog. Gomad currently serializes each `slog.Record` as
JSON, then parses the JSON back into a map to print it.

A smaller design would fan out records before serialization:

- keep `slog.NewJSONHandler` for captured logs and deterministic checksumming;
- use `slog.NewTextHandler` for console output; and
- use `slog.HandlerOptions.ReplaceAttr` for time, level, and source formatting.

This can delete most or all of `internal/prettylog`, its golden tests, and the
`go-isatty` dependency. It also avoids JSON encoding and decoding on the console
path.

The standard text handler does not reproduce the existing colors, fixed field
order, base64 decoding, or multiline traceback formatting. If those are part of
the CLI contract, retain the custom formatter or first agree on a simpler output
contract. Do not accept this reduction solely on line count.

## 4. Translate the standard `testing` package

The files under `tools/gomadv2/internal/testing` contain roughly 1,430 lines copied
or adapted from Go's `testing` package. The active compatibility policy skips
`testing` and `testing/internal/testdeps`, and the implementation already notes
that translating `testing` would avoid the copy.

The target design should translate the Go toolchain's `testing` package and keep
a small Gomad adapter responsible for:

- invoking one selected test inside the simulator;
- connecting test logging to Gomad logging;
- defining the supported test flags and entrypoint behavior;
- mapping time and goroutine operations onto translated standard-library code;
  and
- explicitly rejecting unsupported fuzzing, coverage, or benchmark paths.

This is the largest potential reduction and would automatically track the Go
version selected by the translator. It is also the highest-risk change because
`testing` depends on runtime behavior, process-wide state, `testdeps`, parallel
tests, race reporting, and exit handling.

Prototype this separately. The prototype is successful only if the behavior,
script, metatesting, and race suites pass without broadening shared mutable state
or introducing scheduler-dependent output. Run it with substantially more
parallel tests to expose hidden process-global assumptions.

## 5. Reuse `go-internal/par` selectively

`internal/gomadtool.GetPathForPrecompiledTestBinary` maintains a mutex-protected
map of `sync.Once` values. `go-internal/par.Cache` directly represents an action
computed once per key and can replace this bookkeeping with a single cache.

`internal/translate/workqueue.go` could also use `par.Work`: seed it with nodes
whose dependencies are satisfied, then add newly ready nodes from completed
work. This would remove the channel, submission loop, and `WaitGroup` machinery.

The work-queue replacement is less compelling than the per-key cache. `par.Work`
uses an `interface{}` API and randomized work selection, while the current
`buildInParallel` function is a reasonably deep module with domain-specific
dependency semantics. Add direct cycle, fan-in, fan-out, cached-result, and
high-concurrency tests before considering this replacement. Independently of a
library change, the unused `workQueue.deps` field can be removed.

## 6. Use `hash/fnv` for checksums

`tools/gomadv2/gomadruntime/fnv64.go` implements FNV-1 locally. `hash/fnv.New64`
provides the same algorithm and exposes `Sum64`. Gomad would retain only the
integer-to-byte encoding helper because that encoding is part of the checksum
contract.

This is a modest reduction. Benchmark it before adopting it: storing the standard
implementation behind `hash.Hash64` can add allocation or interface-dispatch
cost in a hot scheduler and logging path. Existing checksum fixtures must remain
byte-for-byte identical.

## Generator cleanup

`tools/gomadv2/internal/simulation/gensyscall/main.go` can use
`golang.org/x/tools/go/packages` instead of invoking and decoding `go list`
itself. This removes some process and JSON plumbing.

Moving its generated Go fragments to `text/template` may improve readability,
but it does not necessarily reduce repository size: the generated declarations
still have to exist as template text, and the template data model adds code.
Treat templates as a maintainability refactor, not a primary code-reduction
project.

## Code that should remain custom

The following packages encode Gomad's defining semantics and should not be
replaced merely to reduce line count:

- `gomadruntime` channel, map, semaphore, scheduler, timer, and randomness code;
- `internal/reflect`, which bridges reflection to simulated maps and channels;
- `internal/simulation/fs`, which models copy-on-write data and partial disk
  persistence;
- `internal/simulation/network`, which models connectivity, delay, ordering, and
  failure; and
- translated standard-library hooks.

General-purpose runtime, filesystem, and network libraries do not provide the
same deterministic or crash-consistency behavior. These are deep modules whose
implementations are larger than their interfaces for a reason. Existing
low-level library reuse, such as `container/heap` in both timer queues, is already
appropriate.

## Suggested sequence

1. Replace the SQLite cache and benchmark cold, warm, and concurrent translation.
2. Convert the two package graph walkers to `packages.Visit`.
3. Replace the per-package `sync.Once` map with `par.Cache`.
4. Decide whether standard `slog` console output is an acceptable UX change.
5. Prototype translating `testing` behind the current public behavior tests.
6. Consider the smaller FNV and generator cleanups only after measuring their
   effect.

For every phase, run focused unit tests with `-tags test_dep`, the script and
behavior suites affected by the change, generated-output checks where relevant,
and the repository lint target. Cache and logging changes should additionally be
tested under abrupt process termination and at least ten concurrent invocations.
