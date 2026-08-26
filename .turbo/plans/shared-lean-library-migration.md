---
status: draft
---

# Plan: Shared Lean Library Migration

## Context

The domain-neutral transition-system and trace-replay Lean modules currently live inside the
Go-oriented `tools/common/formal` module even though their consumers are independent Lean projects.
Move those sources into the primary Lean tree as `model/Shared`, make the module path, library name,
and namespace consistently `Shared`, and remove the obsolete standalone Lean project from
`tools/common/formal`.

This is a hard cutover. Gomad and Umpire3 must consume the same canonical files from `model`, with
no `SharedModel` compatibility modules or aliases. The retained Go packages under
`tools/common/formal` remain unchanged.

## Pattern Survey

### Analogous Features
- `model/lakefile.toml:4` — The primary model Lake project already hosts multiple sibling production libraries, including neutral `Umpire` alongside Temporal-specific `Temporal`.
- `model/Umpire.lean:1` — A root module acts as the umbrella facade for the matching `model/Umpire/` module tree.
- `tools/gomad/formal/lakefile.toml:4` — Gomad independently exposes shared external Lean sources as a local `lean_lib` through `srcDir`.
- `tools/umpire3/model/lakefile.toml:12` — Umpire3 uses the same external-source-library pattern while retaining its own Lake project and targets.
- `Makefile:1032` — The existing Umpire regression gate models a hard namespace/path cutover by rejecting removed trees and scanning live Lean sources and build metadata for obsolete names.

### Reusable Utilities
- `tools/common/formal/lean/SharedModel/Transition.lean:5` — `TransitionSystem` — Defines the domain-neutral state, action, initial-state, and step relation consumed by Gomad and Umpire3.
- `tools/common/formal/lean/SharedModel/Transition.lean:11` — `Runs` — Represents finite executions and provides the shared run vocabulary.
- `tools/common/formal/lean/SharedModel/Transition.lean:25` — `stepStarRefl`, `stepStarSingle`, `Runs.append`, `Runs.empty`, `Runs.firstStep`, and `Runs.uncons` — Supply reusable transition-system proofs rather than consumer-specific duplicates.
- `tools/common/formal/lean/SharedModel/Transition.lean:63` — `Observation` and `TraceStep` — Provide neutral trace record structures.
- `tools/common/formal/lean/SharedModel/TraceReplay.lean:3` — `followNamed` — Replays named actions over an abstract successor function.
- `tools/common/formal/lean/SharedModel/TraceReplay.lean:14` — `check` — Evaluates a Boolean property over the states reached by named replay.
- `tools/umpire3/model/Umpire3/Transition.lean:5` — Umpire3 aliases and proof wrappers preserve an Umpire3-facing transition API while delegating its implementation to the shared library.
- `tools/umpire3/model/Umpire3/TraceReplay.lean:9` — Umpire3’s replay adapter specializes the neutral replay functions to `FiniteView` and adds soundness proofs.

### Convention Anchors
- Module/path/namespace alignment: Lean library names, root modules, directory trees, imports, and namespaces agree, as demonstrated by `model/Umpire.lean:1`, `model/Umpire/Core.lean:4`, and `model/lakefile.toml:7`.
- Umbrella facade modules: A top-level `<Library>.lean` imports focused `<Library>.*` modules; `tools/common/formal/lean/SharedModel.lean:1` and `model/Umpire.lean:1` both follow this structure.
- Independent consumer projects: Gomad and Umpire3 retain separate Lake manifests and default targets while mapping a shared library source directory into each project (`tools/gomad/formal/lakefile.toml:1`, `tools/umpire3/model/lakefile.toml:1`).
- Consumer-owned toolchains: `model/lean-toolchain:1` pins Lean 4.33.1, while `tools/gomad/formal/lean-toolchain:1` and `tools/umpire3/model/lean-toolchain:1` pin Lean 4.28.0; external shared sources are therefore elaborated within each consumer’s toolchain.
- Explicit layout invariants: `tools/umpire3/layout_test.go:50` verifies the independent Umpire3 model project and currently asserts the canonical locations of both shared Lean modules.
- Neutral dependency direction: `model/ARCHITECTURE.md:85` establishes that reusable libraries do not depend on Temporal and that Temporal-specific semantics adapt reusable APIs.
- Mixed-language boundary: `tools/common/formal/go.mod:1` defines an independent Go module whose `model`, `trace`, and `conformance` packages coexist with, but do not import or execute, the Lean library.
- Build orchestration boundary: `Makefile:300` currently builds the standalone shared Lean project before building Gomad’s independent formal project.

### Proposed Alignment
Follow the existing `model/Umpire` module-placement and facade conventions while retaining the established local-`srcDir` consumption pattern for Gomad and Umpire3. The main constraint left by those patterns is cross-version compatibility: the neutral sources must remain independently elaborable under both the primary model’s Lean 4.33.1 toolchain and the consumers’ Lean 4.28.0 toolchains.

## Implementation Steps

1. **Move and rename the neutral Lean modules**
   - Move `tools/common/formal/lean/SharedModel.lean` to `model/Shared.lean`, preserving its
     umbrella-module role and changing its imports to `Shared.TraceReplay` and `Shared.Transition`.
   - Move the `Transition.lean` and `TraceReplay.lean` children into `model/Shared/`, preserving
     declarations and existing comments while renaming their namespaces from `SharedModel` to
     `Shared`.
2. **Establish the canonical `Shared` Lake library**
   - Add the sibling production library `Shared` to `model/lakefile.toml` and include it in the
     default targets so the primary model build validates the neutral library.
   - Rename Gomad's shared library declaration to `Shared` and point its `srcDir` at `model` in
     `tools/gomad/formal/lakefile.toml`.
   - Rename Umpire3's shared library declaration to `Shared` and point its `srcDir` at `model` in
     `tools/umpire3/model/lakefile.toml`.
   - Remove `tools/common/formal/lakefile.toml` and `tools/common/formal/lake-manifest.json` because
     that Go module no longer owns a Lean project.
3. **Cut all consumers over to `Shared.*`**
   - Update Gomad's `VirtualTime` model and tests to import and qualify `Shared.Transition` and
     `Shared.TraceReplay`.
   - Update Umpire3's transition facade and trace-replay adapter to delegate to `Shared.*`, keeping
     the existing Umpire3-facing API and proofs intact.
   - Update `tools/umpire3/layout_test.go` to assert the canonical source paths under
     `model/Shared`.
4. **Update orchestration and architecture documentation**
   - Change `gomad-formal` in the root `Makefile` to build target `Shared` from `model` before
     building Gomad's formal project.
   - Update `model/ARCHITECTURE.md` so `Shared` is documented as the neutral foundation beneath
     the independent `Umpire` and `Temporal` libraries, including the dependency map and build
     commands.
5. **Remove obsolete ownership and verify the hard cutover**
   - Ensure no Lean source tree remains under `tools/common/formal` and no live source or build
     configuration refers to `SharedModel`, `tools/common/formal/lean`, or `sharedformal`.
   - Confirm the retained Go module still contains only its Go packages and passes its focused
     tests.

## Verification

- Run `cd model && mise exec -- lake build Shared`; expect the neutral library to elaborate under
  Lean 4.33.1 with no dependency on `Umpire` or `Temporal`.
- Run `cd tools/gomad/formal && mise exec -- lake build`; expect the Gomad model and tests to
  elaborate against `model/Shared` under Lean 4.28.0.
- Run `cd tools/umpire3/model && mise exec -- lake build`; expect every Umpire3 library and default
  executable dependency to elaborate against the same sources under Lean 4.28.0.
- Run `go test -count=1 -tags test_dep ./tools/umpire3`; expect the relocated layout assertions and
  existing Umpire3 Go tests to pass.
- Run `cd tools/common/formal && GOWORK=off go test -count=1 -tags test_dep ./...`; expect the
  retained shared Go packages to pass unchanged.
- Run `make gomad-formal`; expect the root orchestration target to validate `Shared` and Gomad from
  their new locations.
- Search live Lean, Lake, Go, Make, and current documentation for `SharedModel`,
  `tools/common/formal/lean`, and `sharedformal`; expect matches only in historical design records
  that intentionally describe the migration.
- Run `git diff --check`; expect no whitespace errors.

## Context Files

- `docs/superpowers/specs/2026-08-26-shared-lean-library-migration-design.md` — Approved ownership,
  compatibility, and verification decisions.
- `tools/common/formal/lean/SharedModel/Transition.lean` — Canonical neutral transition types and
  proofs to preserve during the move.
- `tools/common/formal/lean/SharedModel/TraceReplay.lean` — Canonical neutral replay functions to
  preserve during the move.
- `model/lakefile.toml` — Primary Lean project and sibling library conventions.
- `tools/gomad/formal/lakefile.toml` — Gomad's established external-source library mapping.
- `tools/umpire3/model/lakefile.toml` — Umpire3's established external-source library mapping.
- `tools/umpire3/model/Umpire3/Transition.lean` — Largest qualified-reference cutover and existing
  consumer-facing compatibility layer.
- `Makefile` — Root build orchestration for the standalone shared library and Gomad.
