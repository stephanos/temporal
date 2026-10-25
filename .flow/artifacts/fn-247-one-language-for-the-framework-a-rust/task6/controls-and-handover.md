# Task 6 native installer lane

Native `fetch-apalache [workspace-root]` replaces the Python installer. The command
uses raw OS path arguments, prints the admitted absolute jar path on success,
and reports native `xtask:` errors with status 1. Curl's diagnostic stream is
inherited. All production pins reside in `model/dev/xtask/src/apalache_pins.rs`;
the installer and backend include that source. The conductor explicitly authorized
the narrow backend `src/lib.rs` edit beyond the original Touches to consume it.
The existing backend public constants remain available without a second authority.
The native command has no caller-supplied pin/archive/URL overrides.

## Control transfer

| Previous Python responsibility | Rust control in apalache_install |
| --- | --- |
| Symlink and dangling cache components | symlinks_in_every_component_including_dangling_are_refused (also covers installation, lib and jar) |
| Partial installation and wrong existing jar | partial_wrong_and_nonregular_installations_never_download |
| Archive hash before parsing and jar hash before publication | hash_before_parse_and_exact_regular_member_before_publication (also malformed gzip, wrong exact path and nonregular member) |
| Unsafe symlink/FIFO locks | malicious_lock_types_refuse_without_following_or_blocking (also directory lock) |
| Failed download, private staging and cleanup | download_failure_cleans_private_staging_and_never_publishes |
| Four independent concurrent installers publish once | concurrent_installers_publish_once_and_return_same_path (actual child processes, one logged download, identical admitted path, cached validation) |
| Ordinary native invocation and dynamic downloader output | native_curl_contract_and_download_failure_cleanup (no argument from model directory and explicit root; exact safety flags and structured private archive path) |
| Native usage/status/error | native_dispatch_usage_failure_is_nonzero_and_structured |

No useful Python safety control was retired. Exact Python exception classes,
patching/import APIs, diagnostic strings and byte formatting are retired under
the owner's native-driver amendment. Disposable duplicate/type7/trailer research
cases require replacing the production archive hash; the installer exposes no
such override, so no arbitrary archive grammar or reader-parity promise is added.
Tests build controlled gzip/tar fixtures in-process. Production invokes no system
tar and writes no archive member to disk before jar validation.

The first fixture run was red at the private-staging assertion: observed mode
493 decimal (0755), expected 448 (0700). This was a real implementation defect in
the initial tempfile port; `Builder::permissions(0700)` fixes it before download.
The initial red output was observed through the tool result; subsequent command
logs replaced that temporary log, so no original raw red log is claimed. Native
Clippy initially found missing module docs and a test assertion style issue; both
were corrected. Final logs are source-bound by source-sha256.json.

## Dependencies and checks

Baseline backend controls: 14 passed, exit 0 before edits. The task creates the
apalache_install target, so that target had no pre-edit runnable baseline.
Final native fixture target: 8 passed, exit 0. Backend controls: 14 passed, exit 0;
real backend cases retain their normal optional skip behavior and are not real
Apalache execution evidence. Scoped Clippy for xtask and umpire-apalache,
formatting, and cargo deny check bans passed with exit 0. No full Model gate ran.

`dependency-inventory.json` records before/after package inventories and features,
seven new pinned packages and their R18 bases. `registry-admissions.json` captures
the official cached sparse-index rows and checksum matches. Direct crates.io API
access returned HTTP 403; cached primary registry records supply release dates.
Every new package except adler2 has a release within 12 months. Adler2 2.0.1 is a
finished, widely used checksum primitive; the pinned Rust 1.99.1 standard library's
rust-src Cargo.lock itself uses that exact version and checksum through miniz_oxide.

Before metadata came from the conductor checkout only after verifying its three
Cargo manifest/lock source bytes equal task base 2c3c308. The metadata snapshots
are retained compressed. All six all-platform non-development normal/build
package/feature/edge closure hashes are equal before and after; none contains a
new package. This is a metadata closure proof, not compiled guest publication.
The earlier task2 metadata had inherited feature differences and was rejected as
a baseline rather than used to assert false equality.

## Conductor join obligations

The Make caller resolves the shared .5 bootstrap with
`helper=$(/bin/bash model/scripts/xtask.sh)` then runs
`"$helper" fetch-apalache "$PWD"` from the repository root. Task .5 owns that
script; it is deliberately absent from this independent lane's base. The
conductor must join .5/.6/.7 dispatcher/lib/Make/Cargo/README edits before claiming
that bootstrap caller passes. Source lane tests invoke the compiled xtask directly.

After joining, run the same bootstrap/installer command with no fixture overrides.
The immutable download is:
https://github.com/apalache-mc/apalache/releases/download/v0.62.4/apalache-0.62.4.tgz
Archive SHA-256: b781b53bba32fac14ac2e02b0332e1dc2edc64cb7b850c0608d8142cc7aa2ef4
Jar SHA-256: e5223915507b65cc5f1638bd037ede60f8a4fc1644aff46355779f9b05df9262
The ordinary admitted jar path is
`model/build/apalache/apalache-0.62.4/lib/apalache.jar`; only that jar is published,
alongside the persistent `.install.lock` in the cache. The one joined required
Apalache validation pipeline must consume it. Real pinned installation, joined
bootstrap/feature verification, native macOS execution, canonical guest publication,
review and Flow completion are unclaimed here and owned by the conductor.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(policy: parallel-wave - conductor owns review)
