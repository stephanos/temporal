# Shared Lean Library Migration Design

## Goal

Move the neutral Lean transition and trace-replay primitives out of the Go-oriented
`tools/common/formal` tree and into the repository's primary Lean source tree under `model/Shared`.
The migration is a hard cutover from the `SharedModel.*` module and namespace names to `Shared.*`:
there are no forwarding modules, aliases, duplicated files, or compatibility imports.

The library remains Temporal-independent and Umpire-independent. Gomad, Umpire3, the current
Umpire model, and future Lean models may depend on it without inheriting another model's domain
API.

## Library Boundary

The existing three Lean modules move as follows:

- `tools/common/formal/lean/SharedModel.lean` becomes `model/Shared.lean`;
- `tools/common/formal/lean/SharedModel/Transition.lean` becomes
  `model/Shared/Transition.lean`; and
- `tools/common/formal/lean/SharedModel/TraceReplay.lean` becomes
  `model/Shared/TraceReplay.lean`.

Their declarations and behavior remain unchanged. Only module imports, namespace qualifiers, and
filesystem ownership change. Existing comments are preserved.

`model/lakefile.toml` defines `Shared` as a production Lean library and includes it among the
default targets. `Shared.lean` remains the umbrella module, importing `Shared.Transition` and
`Shared.TraceReplay`. The library must not import `Umpire` or `Temporal`.

Gomad and Umpire3 continue to declare a local `Shared` Lean library whose source directory is the
repository's `model` directory. This preserves their independently buildable Lake projects while
giving both projects one canonical source tree. They do not depend on the complete
`temporal-model` Lake package.

## Migration

Rename the namespaces in the moved sources from `SharedModel` and `SharedModel.TraceReplay` to
`Shared` and `Shared.TraceReplay`. Update all Gomad and Umpire3 imports and qualified references
from `SharedModel.*` to `Shared.*`.

Update the Gomad and Umpire3 Lake library declarations from `SharedModel` to `Shared` and point
their `srcDir` values at `model`. Update the Umpire3 layout test to require the canonical files
under `model/Shared`.

The standalone Lean project under `tools/common/formal` ceases to exist. Delete its `lean`
directory, `lakefile.toml`, and `lake-manifest.json`; retain the independent Go module and its Go
packages. Change the root `gomad-formal` Make target's isolated shared-library build from
`tools/common/formal` to `model`, building target `Shared` explicitly before the Gomad build.

Update `model/ARCHITECTURE.md` and relevant current documentation to list `Shared` as the neutral
foundation beneath `Umpire` and `Temporal`. Historical design documents need not be rewritten
unless they advertise an obsolete current build command or ownership boundary.

## Behavior and Compatibility

The definitions of transition systems, finite runs, reachability, step closure, observations,
trace steps, named trace following, and trace checking remain semantically identical. No new API
is introduced and no proof is weakened.

This is intentionally source-incompatible for Lean consumers. Imports such as
`SharedModel.Transition` and names such as `SharedModel.Runs` stop existing immediately. Every
in-repository consumer moves in the same change.

The Go packages under `tools/common/formal` keep their import paths and behavior. Moving the Lean
sources must not change their module boundary or Go test commands.

## Verification

Build the neutral library directly from the primary model project:

```sh
cd model
mise exec -- lake build Shared
```

Build each consuming Lean project through its normal root:

```sh
cd tools/gomad/formal
mise exec -- lake build

cd tools/umpire3/model
mise exec -- lake build
```

Run the focused Umpire3 layout test and the retained shared-formal Go tests with the repository's
required `test_dep` tag. Run the root `gomad-formal` Make target to verify its updated orchestration.

Finally, search the repository for `SharedModel`, `tools/common/formal/lean`, and the removed
standalone Lean project name `sharedformal`; no current code, build configuration, or runnable
documentation may retain them. Run `git diff --check` and report unrelated failures separately.

## Trade-offs

Keeping `Shared` in the existing `model` Lake project avoids another manifest and toolchain while
still exposing a distinct library boundary. Consumer Lake projects refer directly to the canonical
source directory, matching their previous arrangement and retaining independent builds.

The hard rename makes module ownership obvious and aligns filesystem paths, imports, library name,
and namespaces. Its one-time cost is updating all consumers atomically; compatibility wrappers
would preserve misleading `SharedModel` terminology and create two apparent homes for the same
API.
