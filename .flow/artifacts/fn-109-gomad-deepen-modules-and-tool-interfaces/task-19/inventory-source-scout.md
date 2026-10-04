# Package-discovery source inventory

Read-only follow-up from `/root/architecture_fitness_scout`, requested Codex
thinking scout `gpt-6.1-sol/high`. The single judge returned
`Tier: session (jev-unavailable(no_key))`; the existing agent was not rerouted.
Execution model metadata was not independently observable. This inventory is
source evidence for task 19, not implementation or platform qualification.

## Discovery boundaries

All paths below are relative to `tools/gomad3`. Read-only
`go list -e -mod=readonly -tags test_dep -json=… ./...` returned 74 package
records for each qualified source set: 20 overlay packages, the test-only
module root, and 53 other records. Each listing reported eight errors, all
inside the overlay; no included non-overlay package reported an error. Filter
the exact overlay boundary before rejecting errors on included packages.

The required checked-in source exclusions are:

| Exact source tree | Why it is outside host-package ownership |
| --- | --- |
| `toolchain/runtime/overlay` | Patched GOROOT inputs, including legitimate standard-library internal imports |
| `cmd/gomad/testdata` | CLI execution fixtures: campaign, seedfree, watchdog |
| `deterministicio/testdata` | Adapter fixture modules and grpcdns test source |
| `internal/compatibilitypack/testdata` | xsys compatibility fixture module |
| `internal/gomadtool/conformance/testdata` | Runtime/compiler/boundary fixtures and adapter modules |
| `testdata` | External Runner consumer; future architecture fixtures also belong here |
| `qualification/corpus` | Separate core qualification workload module |

`qualification/corpus/go.mod` names `gomad3.core.corpus`; Makefile's
`core-qualification-set` explicitly selects this working directory. It is not
a host package accidentally omitted by the checker.

## Nested modules

Inventory modules independently of enclosing fixture-tree exclusions. Each
checked-in module entry must match its `go.mod` and source beneath it; an
unexpected new module must require explicit classification even inside an
already excluded fixture tree. The thirteen current entries are:

- `deterministicio/testdata/cactusstatsd/go.mod`
- `deterministicio/testdata/hashicorpmetrics/go.mod`
- `deterministicio/testdata/memberlist/go.mod`
- `deterministicio/testdata/pebble/go.mod`
- `deterministicio/testdata/sentry/go.mod`
- `deterministicio/testdata/sockaddr/go.mod`
- `deterministicio/testdata/sprig/go.mod`
- `deterministicio/testdata/validator/go.mod`
- `internal/compatibilitypack/testdata/xsys/go.mod`
- `internal/gomadtool/conformance/testdata/go.mod`
- `internal/gomadtool/conformance/testdata/libc_adapter/go.mod`
- `internal/gomadtool/conformance/testdata/sqlite_adapter/go.mod`
- `qualification/corpus/go.mod`

Four other current testdata roots contain only non-Go evidence:
`cmd/gomad/internal/cli/testdata`, `runner/internal/campaign/testdata`,
`runner/testdata`, and `target/testdata`. They need no Go-source exclusion
today. New source there must be classified; a generic any-testdata exemption
would conceal that change.

## Root and optional trees

The module root currently has six Go files, all external tests:
architecture, typed-command ownership, Runner consumer, network ownership,
filesystem ownership, and simulation gate selection. Platform metadata lists
them in XTestGoFiles, with no GoFiles or CgoFiles. Own the test harness rather
than excluding its directory. Future production root source must enter
production ownership/edge checks or fail pending explicit classification.

Only `.toolchain` and `.bin` are optional generated roots, supported by
`.gitignore`; installation.go also defines `.toolchain` as the installation
root. The scout inspected immediate entries only, not their contents. Their
absence must succeed on a clean checkout. No `.gomad` root was present and no
source-based justification was found for exempting it. No other hidden or
underscore source directory was found outside the optional roots.

Normalize module-relative paths and match exact directory components and
descendants: `toolchain/runtime/overlay-extra` and `.toolchain-extra` must not
match. Inventory Go source and nested go.mod files independently of go-list
metadata, pruning only those two optional generated roots. Required source
exclusions must match source, so missing source or modules fail with a specific
stale-exclusion diagnostic. Unclassified hidden/underscore source, fixture
roots and nested modules also fail; discovery omissions are not exemptions.

## Platform and matcher controls

Both qualified source sets select Unix helpers. Darwin artifact files include
`rename_noreplace_darwin.go`; Linux selects `rename_noreplace_linux.go` and
`rename_sysnum_linux_amd64.go`. Execution selects
`descriptor_dup_other_unix.go` on Darwin and `descriptor_dup_linux.go` on Linux.
Use build-selected metadata and preserve IgnoredGoFiles distinctions; names
alone do not establish selection.

Specific negative controls must cover removed required source, removed
required go.mod, unexpected nested modules inside an excluded fixture tree,
hidden/underscore source roots, prefix lookalikes, new production root source
and ordinary ownerless roots. Positive controls must cover absent and populated
optional roots, current test-only root, classified fixtures/modules and both
platform file selections. Each checks the actual production matcher/checker
and its diagnostic, not merely fixture existence.

The scout performed no source/artifact/Flow/Git writes, tests, builds,
generation or CLI bridges. The conductor retained this digest and independently
read the corpus module, canonical Makefile selection and optional-root ignore
entries. Writing-for-agents groups the boundaries with their failure controls.
