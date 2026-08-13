# Gomad v3 Temporal coupling audit

**Date:** 2026-08-12

**Scope:** The current working tree under `tools/gomadv3`, its root Make/CI
integration, and the root module dependencies used by Gomad's qualification
tests. Generated toolchain contents under `.toolchain` were excluded.

## Bottom line

The deterministic runtime, `world`, record/artifact formats, replay engine,
process supervisor, read-only mounts, and standard-library boundary are already
largely application-neutral. Production Gomad code imports no Temporal service
package and encodes no workflow, activity, namespace, persistence, or Nexus
concept. `tools/gomadv3/go.mod` also has no third-party requirements.

The strongest coupling is instead at the target-policy and product boundary:

1. every `go-test` target silently receives Temporal's `test_dep` build tag;
2. the security-sensitive capability exceptions contain exact versions and
   sums from Temporal's dependency graph, including `go-isatty`;
3. `modernc.org/libc` is special-cased through capability validation, release
   metadata, profile construction, and diagnostics rather than being an
   isolated adapter;
4. the complete test/release gate and runner installation assume the Temporal
   checkout layout.

Those issues make the supported target set and the way Gomad is installed
depend on its first consumer. They are more important than the remaining
Temporal names in module paths, tests, and documentation.

## Priority overview

| Priority | Finding | Impact | Effort |
| --- | --- | --- | --- |
| P0 | Implicit `test_dep` changes every external `go-test` target | High | Low |
| P0 | Capability exceptions encode Temporal-selected dependencies | High | Medium |
| P1 | Core test and upgrade gates require the Temporal parent repository | High | Medium |
| P1 | Public qualification can omit a vendor-neutral workload corpus | High evidence gap | High |
| P1 | Runner discovery assumes the checkout's `.bin`/`.toolchain` layout | High | Medium |
| P1 | The generic adapter model collapses to one `modernc.org/libc` special case | Medium-High | Medium-High |
| P2 | The generic module owns a Temporal qualification manifest and default | Medium | Low |
| P2 | The module identity is rooted under `go.temporal.io/server` | Medium extraction cost; no runtime impact | Low-Medium |
| P3 | Generic unit fixtures use Temporal package names and conventions | Low | Low |
| P3 | Compiler test fixtures are shipped in the production interception table | Low; not Temporal-specific | Medium |

## Findings

### P0: `go-test` silently enables Temporal's `test_dep` tag

**Evidence**

- `tools/gomadv3/internal/target/target.go:511-527` adds `test_dep` whenever the
  target kind is `go-test`, even when the caller supplied no build tags.
- `tools/gomadv3/internal/target/target_test.go:171-203` explicitly pins this
  behavior as `TestPrepareGoTestAlwaysAddsTestDependencyTag`.
- The Temporal root defines `test_dep` as part of its test tags at
  `Makefile:51`, and the legacy wrapper repeats it at `Makefile:182-187`.
- `tools/gomadv3/README.md:290-295` currently describes preserving
  `-tags test_dep` as part of the Gomad contract.

**Why this is coupling**

`test_dep` is a Temporal build convention, not a Go or DST convention. A
general-purpose test runner must not change a target's selected source files
unless the user asks it to. An unrelated repository may use the same tag for
different code, so Gomad can silently compile extra implementations, change
behavior, introduce dependencies, or fail a build. The prepared-target
provenance then records a tag that the caller never selected.

**Recommended boundary**

Make `go-test` tag-neutral: `normalizeBuildTags` should validate, sort, and
deduplicate only supplied tags. Temporal's root wrapper should explicitly pass
`--build-tag=test_dep`. Retain tests proving that an empty tag set remains empty
and that explicit tags are preserved.

This is the first change to make: it is localized, removes observable
application coupling, and does not weaken deterministic isolation.

### P0: capability exceptions mirror Temporal's dependency closure

**Evidence**

`tools/gomadv3/internal/target/capability.go:179-208` contains two kinds of
dependency-specific exceptions:

| Exception | Pinned versions | Corresponding current consumer |
| --- | --- | --- |
| `github.com/mattn/go-isatty` | `v0.0.20`, `v0.0.21` | adapter fixture and Temporal root (`go.mod:194`) |
| `github.com/remyoudompheng/bigfft` | one pseudo-version | adapter fixture and Temporal root (`go.mod:207`) |
| `golang.org/x/sys` | `v0.41.0`, `v0.47.0` | adapter fixture and Temporal root (`go.mod:231`) |
| `modernc.org/memory` | `v1.11.0` | adapter fixture and Temporal root (`go.mod:246`) |
| `modernc.org/sqlite` | `v1.51.0` | Temporal root (`go.mod:98`) |
| `modernc.org/libc` | the Gomad-pinned adapter version | Temporal root and the adapter itself (`go.mod:244`) |

The table is not merely documentation:

- `capability.go:218-240` detects a locally replaced `modernc.org/libc` and
  treats the table as the adapter's trusted closure.
- `capability.go:354-417` normally rejects packages importing `syscall`,
  `os/exec`, `os/signal`, `os/user`, `plugin`, `runtime/cgo`, or any
  `golang.org/x/sys/*` package, and rejects foreign/assembly sources.
- `capability.go:507-525` and `:588-605` bypass those checks for any package in
  an exact allowlisted module/version/sum when the libc adapter is active.
- `tools/gomadv3/internal/target/capability_test.go:128-143` codifies an
  accepted closure containing the same packages.

A separate global exception in `capability.go:280-303` recognizes exact source
files and linkname directives from
`github.com/modern-go/reflect2@v1.0.3-0.20250322232337-35a7c28c31ee`, the version
selected by Temporal at `go.mod:199`. This exemption is not scoped to the libc
adapter.

**Why this is coupling**

The generic target validator decides support according to the dependency
versions selected by Temporal and Gomad's current SQLite/libc fixture. A normal
dependency upgrade can therefore require editing core Gomad source, while an
otherwise equivalent external target using another version fails. Conversely,
the exception is at module granularity, so the trusted surface is wider than
the exact package capability that motivated it.

`go-isatty` is the clearest symptom: terminal detection has no inherent place
in a DST's core capability policy. It appears because it is in a currently
qualified dependency closure.

**Recommended boundary**

Keep the fail-closed behavior, hashes, versions, and sums, but move exceptions
out of `internal/target` into immutable, versioned compatibility manifests:

- the generic capability engine owns package-closure discovery and generic
  rules for host escape, foreign code, and `go:linkname`;
- each foreign-runtime adapter owns the exact modules, packages, source
  fingerprints, linkname directives, and exceptional capabilities it needs;
- a downstream Temporal compatibility profile owns additional MVS variants
  needed only by Temporal's dependency graph;
- the selected policy identity is stored in provenance and replay artifacts.

Adding or upgrading an application dependency should update a compatibility
manifest and qualification evidence, not a map in the generic validator. An
unknown version should still fail closed with an actionable message naming the
missing compatibility pack; it should not be accepted broadly.

### P1: the adapter abstraction is `modernc.org/libc`-specific end to end

**Evidence**

- `tools/gomadv3/internal/version/descriptor.go:25-47` models adapters as a
  slice, but `:191-204` requires that the slice contain `modernc.org/libc`.
- `descriptor.go:341-356` discards the general collection when generating Go
  and emits singular `ModerncLibcVersion` and `ModerncLibcSum` constants.
- `tools/gomadv3/internal/ioprofile/profile.go:27-64` has one profile with one
  build-overlay callback; its inventory at `:68-78` always advertises the
  modernc boundary.
- `tools/gomadv3/internal/ioprofile/libc_adapter.go:21-28`, `:47-109`, and
  `:145-181` detect, version-check, source-hash, rewrite, and replace one exact
  modernc libc release.
- `tools/gomadv3/internal/doctor/doctor.go:28-50` exposes a singular `Adapter`,
  and `:58-88` always reports modernc libc as compatible, including for targets
  that never use it.
- The capability validator repeats the same module identity in
  `tools/gomadv3/internal/target/capability.go:179-192`.

The adapter implementation itself is not Temporal-specific: it is a valid
generic bridge for pure-Go packages built on modernc libc. The coupling is that
one dependency family is mandatory core product structure because it was
needed by the initial Temporal/SQLite corpus.

**Recommended boundary**

Extract a deep foreign-runtime-adapter module with a small contract along these
lines:

- immutable identity and compatibility policy;
- target metadata detection;
- build preparation/overlay production;
- declared deterministic capabilities and inventory entries.

Generate a keyed adapter table from release metadata, allow zero or more
adapters, record the exact selected set in the I/O profile/artifact identity,
and have `doctor` report a collection. Ship the current modernc adapter as an
optional built-in. Do not make adapters dynamically loadable inside a target;
the current reproducibility and replay guarantees require a closed, recorded
composition.

This is a strategic refactor rather than a prerequisite for keeping modernc
support. It becomes valuable now because the existing special case has already
spread across four packages and generated artifacts.

### P1: core qualification requires the Temporal parent repository

**Evidence**

- `tools/gomadv3/test.sh:5-13` defines the repository root as two directories
  above the Gomad module.
- Its default `test-upstream` tier at `test.sh:1530-1589` invokes the parent
  Makefile's `gomadv3-run` and `gomadv3-test` targets and the sibling
  `tools/gomadv3roottestdata` fixture.
- `tools/gomadv3/internal/ioprofile/sqlite_toolchain_test.go:16-36` walks four
  directories upward and builds `tools/gomadv3/root_testdata/io_sqlite` from
  the Temporal root module, borrowing its `modernc.org/sqlite` dependency.
- `tools/gomadv3/internal/qualificationgen/main.go:23-27` defaults to the
  Temporal manifest and a working directory two levels above the Gomad module.
- `tools/gomadv3/qualification/temporal.json:1-72` targets named tests under
  `./tests` and `./common/*`, plus the parent repository's `./schema` tree.
- `tools/gomadv3/internal/upgradegen/main.go:41-50` makes the parent-dependent
  upstream tier part of every upgrade dossier.
- `.github/workflows/gomadv3.yml:39-51` obtains the bootstrap Go version from
  the root `go.mod`, not `tools/gomadv3/go.mod`, and runs the nested module from
  the Temporal workflow.
- The workflow's path filters at `.github/workflows/gomadv3.yml:5-21` do not
  include the sibling `tools/gomadv3roottestdata` fixture used by the complete
  gate, so changing that fixture alone does not trigger the gate.

**Why this is coupling**

Copying or publishing `tools/gomadv3` does not produce a self-contained
`make test` or release qualification. Core tests borrow the parent module's
dependencies and wrappers, while downstream Temporal compatibility is mixed
with upstream Go compatibility.

**Recommended boundary**

- Put every core fixture under Gomad in a self-contained module with explicit
  `go.mod`/`go.sum` inputs. In particular, the SQLite fixture must not borrow
  Temporal's root `go.mod`.
- Keep Go toolchain disabled-mode/upstream tests in the core gate.
- Move the root wrapper checks, `test_dep` coverage, unchanged Temporal suite,
  and functional-suite corpus into a Temporal-owned downstream integration
  workflow.
- Let the upgrade dossier ingest an optional external corpus report rather
  than executing parent-repository gates itself.
- Run Gomad's standalone CI from `tools/gomadv3/go.mod`; retain a separate
  Temporal consumer job.

The desired split is "Gomad qualifies Gomad; Temporal qualifies its use of
Gomad." A fresh Gomad checkout should pass its complete core gate without a
Temporal checkout.

### P1: public qualification can omit a vendor-neutral workload corpus

**Evidence**

- `tools/gomadv3/internal/upgrade/dossier.go:175-216` loads retained-corpus
  evidence, but sets `Qualified` using only the supported host and internal
  gate results.
- `dossier.go:371-389` treats a missing corpus as `not-configured`. A configured
  corpus is now strongly validated as a canonical
  `gomadv3.qualification-set-report/v1` report and must itself be qualified,
  which is a sound ingestion boundary; absence nevertheless does not affect
  the dossier verdict.
- `.github/workflows/gomadv3.yml:50-58` generates and supplies that corpus only
  for scheduled and manually dispatched runs. Pull-request and push dossiers
  can therefore report `qualified=true` without corpus evidence.
- The only checked-in qualification manifest is
  `tools/gomadv3/qualification/temporal.json:1-72`. It is versioned and bounded,
  but all five entries are Temporal repository workloads; there is no
  application-neutral baseline.

**Why this is coupling**

The release verdict can prove Gomad's internal fixtures and one platform while
omitting workload evidence entirely. When that evidence is present, the only
shipped corpus makes Temporal the de facto application model without defining
a portable baseline for other Go projects. This is a qualification gap rather
than a runtime dependency, but it encourages future capability work to follow
Temporal's next blocker.

**Recommended boundary**

Add a versioned Gomad corpus with representative generic concurrency,
filesystem, network, persistence, and third-party-library workloads, each with
semantic oracles rather than repeatability alone. Reuse the existing
qualification-set schema and validation, but require this neutral corpus for a
public `qualified=true` release. Record Temporal's suite inventory separately
as downstream dogfooding evidence and require it only for the Temporal
integration verdict.

This is higher effort than moving existing fixtures: the corpus needs stable
ownership and must avoid turning current library popularity into new core
allowlists.

### P1: installation and resource discovery assume a source checkout

**Evidence**

- `tools/gomadv3/cmd/gomad/main.go:506-521` derives the toolchain root by taking
  the executable's grandparent and appending `.toolchain`. This works for
  `tools/gomadv3/.bin/gomad` only.
- `main.go:216-240` repeats the executable-grandparent assumption for `doctor`.
- `tools/gomadv3/build.sh:5-17` advertises `GOMADV3_TOOLCHAIN_DIR`, but the
  runner never reads that setting.
- `tools/gomadv3/internal/target/target.go:186-213` prescribes
  `make -C tools/gomadv3 toolchain` in runtime failures.
- Generated maintenance guidance in
  `tools/gomadv3/internal/version/descriptor.go:321-337` also hard-codes the
  current path in the Temporal monorepo.

**Why this is coupling**

A copied binary, `go install`, package-manager installation, or custom
toolchain directory cannot reliably run `doctor`, `explore`, `qualify`,
`resume`, or `replay`. This is checkout coupling rather than Temporal domain
coupling, but it directly blocks the stated general-purpose tool goal.

**Recommended boundary**

Create one installation/resource resolver used by every command. It should
support an explicit CLI/environment toolchain root, a packaged/bundle
manifest, and a documented adjacent-resource fallback. The builder and runner
must honor the same location, and `doctor` should produce repair instructions
for the resolved installation instead of a repository-relative Make command.

Keep resource resolution separate from target working-directory resolution so
running Gomad in an arbitrary consumer module never depends on where its binary
was built.

### P2: the generic module owns the Temporal qualification default

**Evidence**

- `tools/gomadv3/qualification/temporal.json:1-72` is a concrete inventory of
  named Temporal tests, packages, expected dependency failures, and the root
  repository's schema directory.
- `tools/gomadv3/internal/qualificationgen/main.go:23-27` makes that manifest,
  the parent repository, and Gomad's checkout-local binary/toolchain paths its
  defaults.
- `.github/workflows/gomadv3.yml:50-68` runs the Temporal set on scheduled and
  manual Gomad CI and publishes it beside the generic upgrade dossier.
- `tools/gomadv3/README.md:116-133` presents the Temporal set as Gomad's checked
  representative workload set.

`internal/qualificationset` itself is reusable and application-neutral: it
validates a manifest, invokes the public CLI, checks semantic expectations, and
publishes a canonical aggregate. The coupling is the Temporal-specific default
and corpus ownership, which give the generic module responsibility for a
downstream consumer's repository paths, named tests, and qualification
lifecycle.

**Recommended boundary**

Keep the qualification-set engine in Gomad, but require an explicit manifest
and working directory rather than defaulting to Temporal. Move
`qualification/temporal.json` and its scheduled execution to a Temporal-owned
integration location that invokes Gomad through its public CLI. Gomad
documentation can link to downstream evidence, but its core release gate
should not know the consumer's source layout.

### P2: module ownership and publication remain Temporal-shaped

**Evidence**

`tools/gomadv3/go.mod:1` declares
`module go.temporal.io/server/tools/gomadv3`. Representative self-imports appear
in `tools/gomadv3/internal/runner/runner.go:18-26`,
`tools/gomadv3/internal/replay/replay.go:15-23`, and
`tools/gomadv3/world/child/child.go:13-14`.

This is not a dependency on Temporal server code: all such production imports
stay inside the Gomad module, and its `go.mod` has no `require` directives. A
Temporal-owned module can still be general-purpose. The cost is distribution:
publishing Gomad as an independently versioned neutral project requires a
module-wide import rewrite and makes its public `world` packages appear to be
part of Temporal Server.

**Recommended boundary**

Do not churn the module path until an independent repository and release name
exist. At extraction time, move to that stable neutral module path in one
mechanical change and keep Temporal as a downstream consumer. This is lower
priority than removing behavioral and gate coupling.

### P3: generic unit-test fixtures use Temporal identities

Examples include:

- `tools/gomadv3/internal/ioprofile/profile_test.go:79-108` uses
  `go.temporal.io/server/tests.test`, a Temporal test-suite name, and
  `test_dep` to test generic identity matching;
- `tools/gomadv3/internal/record/record_test.go:246-247` uses
  `go.temporal.io/server/common/timer.test`;
- `tools/gomadv3/internal/runner/runner_test.go:986-994` builds its generic fake
  preparer around `./tests`, `test_dep`, and the Temporal module path;
- `tools/gomadv3/internal/romount/config_test.go:10-24` uses
  `/go.temporal.io/server/schema` as the canonical mount target.

These literals do not change production behavior. They make generic invariants
harder to recognize and can conceal accidental assumptions during extraction.
Replace them opportunistically with `example.test/project`, `./pkg`, a neutral
explicit tag, and `/workspace/schema`. Reserve real Temporal names for the
downstream integration suite.

### P3: compiler conformance fixtures leak into the shipped toolchain

This is not Temporal coupling, but it is the same class of corpus-to-product
leakage:

- `tools/gomadv3/boundary/manifest.json:4105-4188` stores compiler test package
  paths such as `gomadv3.test/intercept` and
  `gomadv3.test/interceptfail/*` beside production boundary metadata.
- They are emitted into the unconditional production table at
  `tools/gomadv3/overlay/src/cmd/compile/internal/gomadintercept/spec_go126.go:149-161`.
- `tools/gomadv3/overlay/src/cmd/compile/internal/gomadintercept/intercept.go:48-61`
  selects entries solely by package import path.

An unrelated module using one of those import paths can trigger Gomad's
compiler rewrite or a deliberate compiler-failure case. Keep conformance cases
in a separate test manifest and emit them only in a test toolchain/overlay.

## Recommended ownership split

| Owner | Belongs here |
| --- | --- |
| Gomad core | deterministic runtime/toolchain patch, generic capability engine, runner, World, records/artifacts/replay, standard-library I/O boundary, adapter contract, self-contained fixtures and CI |
| Built-in adapter | modernc libc source rewriting, exact adapter identity, required dependency/source/linkname policy, adapter-specific qualification |
| Temporal integration | `test_dep`, root Make wrappers, Temporal-selected compatibility variants, unchanged Temporal tests, functional-suite corpus, Temporal CI consumer job |

The split should preserve a strict dependency direction: Temporal integration
depends on Gomad and optional adapters; Gomad core must not depend on Temporal
integration data.

## Safety and scale constraints for decoupling

- Do not replace exact dependency exceptions with a permissive `x/sys`, foreign
  source, or `go:linkname` allowlist. The current fail-closed posture is a
  security and reproducibility property worth preserving.
- Compatibility manifests and selected adapters must be immutable, versioned,
  included in target/profile identity, and verified again during replay.
- Missing or changed adapters must fail before target execution; there must be
  no fallback to live host behavior.
- A data-driven policy should use indexed module/package lookup. At a 10x larger
  dependency closure, lookup overhead should remain negligible relative to the
  existing `go list` and source hashing work.
- Keep adapter policy narrow. A manifest should authorize the exact package and
  exceptional capability needed, not every package sharing a module.

## Reviewed constraints that are not Temporal coupling

- Pinning Go 1.26.4 and qualifying only `darwin/arm64` are version/platform
  scope decisions, not Temporal assumptions.
- Pure-Go/`CGO_ENABLED=0`, no race detector, no subprocesses/signals/plugins,
  and fail-closed unsupported host I/O are current deterministic capability
  limits. A general-purpose tool may be deliberately non-exhaustive.
- The existence of a modernc libc adapter is useful generic functionality. Its
  mandatory, cross-cutting special treatment is the issue.
- Gomad-owned schema names, environment variables, wire magic, fixed virtual
  clock, empty in-memory filesystem, loopback network, artifact layout, and
  replay rules contain no Temporal domain assumptions.
- `world`, `protocol`, `internal/record`, `internal/artifact`,
  `internal/replay`, `internal/romount`, and `internal/process` are
  application-neutral apart from the module import prefix.

## Suggested sequence

1. Remove implicit `test_dep`; pass it only from Temporal integration.
2. Make core fixtures/gates standalone and move the Temporal manifest/default
   to downstream integration.
3. Define a required vendor-neutral qualification corpus and keep Temporal's
   results as downstream evidence.
4. Extract capability exceptions into versioned adapter/downstream manifests
   while preserving fail-closed validation.
5. Generalize adapter metadata and diagnostics around an immutable adapter
   collection.
6. Centralize installation/toolchain discovery and add a supported standalone
   distribution layout.
7. Rename the module only when the independent repository/release boundary is
   chosen; neutralize incidental unit fixtures in the same extraction work.
