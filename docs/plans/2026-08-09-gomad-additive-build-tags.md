# Additive GoMaD Build Tags

## Problem

GoMaD currently loads source packages with its private `gomad` tag and compiles translated tests with its private `linkname` tag. Callers cannot add project-specific build tags, so a Temporal package selected with `test_dep` can differ between a native Go build and GoMaD translation.

## Design

Add a `-tags` option, with the same comma-separated value format as the Go tool, to the user-facing `translate`, `test`, `build-tests`, and `debug` commands. Tags supplied by users are additive to GoMaD's private tags:

- Package loading uses `gomad`, the user tags, and `race` for race builds.
- Generated Go compilation uses `linkname` and the user tags. Race compilation continues to use Go's `-race` option.
- Users never need to know or provide GoMaD's private `gomad` or `linkname` tags.

Store normalized user tags in `gomadtool.BuildConfig`. Centralize parsing, deduplication, and effective-tag construction there so translation and compilation cannot accidentally diverge. Preserve the user's first-seen order while removing duplicate and empty tags.

Propagate the same configuration through package loading, `go test`, `go test -c`, and Delve build flags. Existing behavior remains unchanged when `-tags` is omitted.

Include normalized user tags in the translation cache identity because build constraints change the loaded package graph and selected files. Keep translated and precompiled-test directory names stable so existing metatesting lookup remains compatible; as today, the most recently prepared test binary for a build configuration occupies the fixed path.

## Errors

Accept the Go tool's tag syntax rather than introducing a separate validator. Invalid tag combinations therefore fail through the existing package-load or Go-command error path, retaining the underlying diagnostic.

## Tests

- Unit-test tag normalization and effective tags for defaults, user tags, duplicates, and race mode.
- Verify command construction passes user tags alongside `linkname` without exposing `gomad` at generated-build time.
- Add a translation fixture with mutually exclusive build-tagged files to prove user tags affect package loading.
- Update the GoMaD test workflow to prepare translated Temporal tests with `-tags=test_dep` and retain `-tags=linkname,test_dep` for the native outer test process.

Run focused GoMaD tests with `-tags test_dep`, then the repository's formatting and lint targets as available.

## Tradeoffs and Failure Modes

Centralizing the merge adds a small API surface to `BuildConfig` but prevents phase-specific tag logic from drifting. Stable artifact paths avoid changing the metatesting protocol, at the cost of not supporting concurrent preparations with different user-tag sets in the same checkout; that limitation already exists for other configuration changes that share the path.

No new dependencies or security boundaries are introduced. Tag processing is linear in the number of supplied tags, and the cache key prevents stale translated packages when tag selection changes.
