# Package layout of model/umpire: role directories, clear names, a crate map (model)

## Conversation Evidence

> user (2026-10-24): "is the package layout in temporal/model/umpire/ appropriate? reasoanble? clear?"
> assistant: the shape matches the architecture, but: two annotation families with overlapping names (`annotations` vs `catalog*`); inconsistent nesting; three confusable test crates; the hand-written Go `runner` inside the generated protobuf tree; stale descriptions; `reference/` twice.
> user: "yes queue recommendations"
> user: "can we be more aggressive evne in a lcnear design?" then "show me full proposes layout"
> assistant: proposed the role-directory layout below, with a facade crate, the annotations/catalog merge, back ends grouped, and the Go module split.
> user: "yes, update fn-244"

## Goal & Context

Make `model/umpire` readable from its paths: crates grouped by role, one nesting rule, names that show which crates authors use, generated code separated from hand-written code, and Kettle in one place. No behaviour change.

## Target layout

```
model/umpire/
├── README.md                    crate map: each crate's job, author-facing or not
├── author/
│   ├── umpire/                  umpire                 NEW facade: re-exports dsl, suite, catalog derive
│   ├── dsl/                     umpire-dsl
│   │   └── macros/              umpire-dsl-macros
│   └── suite/                   umpire-suite
├── front/                       umpire-front
├── ir/                          umpire-ir
│   └── derive/                  umpire-ir-derive
├── backends/
│   ├── native/                  umpire-checker         (crate name kept)
│   ├── process/                 umpire-backend-process shared bounded subprocess, scratch and pinned-JDK support (added 2026-10-24)
│   ├── tla/                     umpire-tla             was umpire-lift; TLA+ binding and emission only
│   ├── tlc/                     umpire-tlc
│   └── apalache/                umpire-apalache        no dependency on umpire-tlc
├── protocol/
│   ├── proto/                   .proto sources
│   └── runtime/                 umpire-runtime
├── catalog/                     umpire-catalog         merged with umpire-annotations
│   ├── syntax/                  umpire-catalog-syntax
│   ├── macros/                  umpire-catalog-macros
│   └── build/                   umpire-catalog-build
├── hosts/go/                    Go module umpire.local/model
│   ├── host/                    package host           wasm Host (was the module root package)
│   ├── runner/                  package runner         Runner, Run, Driver, publish
│   └── gen/umpire/…/v1/         generated only
├── reference/kettle/            umpire-reference-kettle, plus its Go server and Kettle host tests
│                                (own Go module with a replace directive onto hosts/go)
└── dev/
    ├── testing/                 umpire-testing
    └── fixtures/                umpire-fixtures
```

## Architecture & Data Models

- Moves are directory moves plus Cargo path updates. Crate renames are limited to `umpire-lift` -> `umpire-tla` and the merge of `umpire-annotations` into `umpire-catalog`; every other crate name stays.
- The facade `umpire` re-exports what a Model author needs (`dsl`, `suite`, catalog derive). Fixtures, samples and the Temporal Models switch to it where it reduces their dependency lists; nothing is forced.
- Process supervision used by TLC and Apalache moves out of the emitter into its own crate `umpire-backend-process` at `backends/process` (today `umpire-tla::process`: `Limits`, `Refusal`, `Scratch`, `java_command`, `pinned_java`, `run_process`, `run_process_cancelled`, `bounded_text`); `umpire-tla`, `umpire-tlc` and `umpire-apalache` depend on it; `umpire-apalache` stops depending on `umpire-tlc`; `umpire-tla` keeps binding, emission and typed value admission only. Owner, 2026-10-24: the emitter is not the honest owner of process supervision, and fn-240's rIC3 runner must not depend on a TLA+ emitter to run a child process.
- Go: the wasm Host moves from the module root to package `host`; the Runner moves to `hosts/go/runner`; generated protobuf packages move to `hosts/go/gen/...`; the generated-code freshness check covers exactly `gen/` with no exception. Kettle's Go server and Kettle-specific host tests move under `reference/kettle/` as their own Go module.
- Every consumer is updated: `model/temporal` crates and Go packages, `tools/umpire`, `common/testing/testpilot`, `tools/canary`, Makefile targets, scripts, the source-boundary scanner, guest build inputs, MIGRATE.md's path table, AGENTS.md project structure and the module map.

## Acceptance Criteria

- **R1:** The Rust crates sit in the target directories; `umpire-lift` is `umpire-tla`; `umpire-annotations` is merged into `umpire-catalog`; subprocess supervision is `umpire-backend-process` at `backends/process` with `tla`, `tlc` and `apalache` depending on it and `umpire-tla` free of process code; `umpire-apalache` no longer depends on `umpire-tlc`; the facade `umpire` exists and is used by at least the shared fixtures. Errors: behavioral, wire and semantic goldens byte-identical; snapshots that embed crate paths or names may change, each listed; leftover old paths or names in active source, manifests, scripts and current docs fail the scoped check (archived code and historical Flow evidence excluded).
- **R2:** The Go module is split into `host`, `runner` and `gen`; the freshness check has no special case; Kettle's Go server and tests live under `reference/kettle`; all importers updated; Go tests, the live slice and gaps pass on the rebuilt guest.
- **R3:** `model/README.md` has the crate map; Cargo descriptions are current; MIGRATE.md, AGENTS.md and the module map name the new paths.

## Boundaries

- No behaviour, Model, DSL surface or protocol change.
- `umpire-checker` keeps its crate name; `dev/` crates stay separate.

## Decision Context

2026-10-24, owner ("yes" to the follow-up): `process.rs` leaves `umpire-tla` for its own crate `umpire-backend-process`, folded into .1 while it is in progress; the crate-map wording names one job per crate. Pure move, no behaviour change.

2026-10-24, the owner approved the aggressive layout and both open choices as recommended (keep the `umpire-checker` crate name; keep the `dev/` crates separate). Placement: the Rust layout (.1) runs as one serialized freeze after fn-238 and fn-242 close and before fn-237.1 starts; the Go split (.2) runs after fn-228.6 and before fn-228.7, so the fn-228 write-up names the final paths. The guest is rebuilt because guest identities hash crate lockfile closures.
