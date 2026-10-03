# Task 10 (R15/S3): validated installation description

Base revision: `331b75bb6` (branch `gomad-fn109`).

## Decision

A new leaf package, `toolchain/installation`, owns the installation layout. It imports only
the standard library, so `target` and `deterministicio` can import it without reaching the
builder in `toolchain`.

- `Layout` (`At(root)`): unvalidated locations. These are `bin/go`, `bin`, `build-key`,
  `builds`, `builds/<key>`, `locks`, `locks/<key>.lock`, `downloads` and `adapters`. The
  builder publishes this layout, and closure review, module-cache queries and the adapter
  cache read their locations from it. None of these callers needs a complete installation.
- `Build`: the locations inside one build: GOROOT, `bin/go`, `target-cache` and
  `prepared-targets`.
- `Description` (`Describe(root)`): a Layout whose launcher is executable, whose build key is
  well formed and whose named build is present. It supplies the pinned build key and
  `PinnedBuild()`. The validation text moved unchanged from `target.readToolchainIdentityWith`,
  including the repair guidance.
- `target` keeps a private `pinnedToolchain` value: the `ToolchainIdentity` from `go env`
  plus the Description. Preparation (exec and go), linked review, the prepared-target cache
  and the target build cache read locations from that value. Public `ToolchainIdentity`
  values and comparisons are unchanged.
- Resolution stays in `toolchain.ResolveInstallation`. Its executable-relative fallbacks use
  `installation.CheckoutDirectory`. Resolution remains lenient, so `doctor` still reports an
  incomplete installation as a failed check instead of failing to resolve.
- Architecture: `ownerMayImport` allows exactly `toolchain/version` and
  `toolchain/installation` for `target` and `deterministicio`. `TestExactModuleEdges` requires
  `toolchain`, `target` and `deterministicio` to import `toolchain/installation`, and requires
  that package to import no module package. `listHostPackages` and `make test-host` include it.

## Preserved identity

For an absolute root, every location is byte-identical. `filepath.Abs(Join(root, x))`
equals `Join(Abs(root), x)`. The adapter replacement location
`<root>/adapters/<base>@<version>-<inventory16>` is pinned by
`TestAdapterRegistryPublishesReplacementAtStableToolchainLocation` for absolute and relative
roots. That test passed before the edit and after it. A relative root now resolves once
against the working directory for the target build cache and the prepared-target cache, which
previously used a relative path. They name the same directory. Neither path enters a binary
or a recorded identity.

## Left in place (outside Touches)

These developer and maintenance tools under `internal/gomadtool/conformance`, `cmd/gomadtool`,
`upgrade` and `toolchain/patch_regenerate.go` still name the checkout's `.toolchain`. Two of
them read its `build-key`. All of them are harness or maintenance code for the repository's
own checkout, not consumers of a supplied installation. They are also outside the task's
Touches list. `toolchain/build.go buildComplete` still joins `bin/go` under a GOROOT, because
it also inspects the unpublished source tree.
