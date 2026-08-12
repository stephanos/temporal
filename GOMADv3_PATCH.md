# Gomad v3 patch versus whole-file replacements

## Recommendation

Do not replace every patched Go file with a full-file overlay. The patch is
large because Gomad must intercept many standard-library entry points, not
because most of those files have become Gomad-owned. Replacing all of them
would hide a small semantic delta inside a large fork of the Go source tree.

Use a hybrid rule:

- keep surgical patches for small, stable hooks;
- consider a version-pinned full-file replacement for
  `src/net/tcpsock.go` only; and
- reconsider `src/net/net.go` or `src/os/file.go` only if their Gomad-owned
  portions continue to grow substantially.

`src/net/tcpsock.go` is the one current file on the replacement side of the
line: it is modest in size, has 18 dispersed hunks, and changes about 17% of
the upstream file. A replacement would make that file materially easier to
edit and format. The other high-hunk files still contain less than 5% Gomad
delta, so replacing them would mostly copy upstream code.

Before changing representation, regenerate the current patch from a clean
Go 1.26.4 tree and strengthen validation. The inspected working-tree patch is
not currently applicable in full.

## Current shape

The current working-tree patch has 19 files, 79 hunks, 317 added lines, and 12
removed lines. The corresponding upstream files contain 20,527 lines. A full
replacement of every touched file would therefore check in about 20,832 lines
of patched Go source to represent 329 changed lines. The current patch is 930
lines, while the already separate additive overlay is 1,584 lines.

| File | Upstream lines | Hunks | Delta | Changed/upstream | Recommendation |
| --- | ---: | ---: | ---: | ---: | --- |
| `src/net/tcpsock.go` | 478 | 18 | +82/-1 | 17.4% | Reasonable replacement candidate |
| `src/net/net.go` | 897 | 12 | +42/-0 | 4.7% | Keep patched |
| `src/os/file.go` | 942 | 9 | +43/-2 | 4.8% | Keep patched for now |
| `src/runtime/proc.go` | 8,125 | 8 | +29/-5 | 0.4% | Definitely keep patched |
| `src/runtime/time.go` | 1,532 | 7 | +18/-3 | 1.4% | Keep patched |
| `src/testing/testing.go` | 2,845 | 4 | +8/-0 | 0.3% | Definitely keep patched |

The hunk count alone is misleading. For example, replacing
`src/runtime/proc.go` would take ownership of more than 8,000 upstream lines
to avoid eight small hooks. In contrast, replacing `src/net/tcpsock.go` would
produce a roughly 559-line file and remove 18 frequently edited hunk sites.

The patch has also grown in coherent stages rather than accumulating duplicate
file sections: the committed versions progressed from 2 files/8 hunks, to 4/10,
to 5/19, and then to 15/63. The local filesystem work raises that to 19/79.
This is real interception-surface growth, so changing storage format will not
remove the underlying behavioral obligations.

## What a replacement simplifies

A full-file replacement makes local development easier in several concrete
ways:

- editors, `gofmt`, and import tooling operate on normal Go files;
- adjacent hooks can be refactored without manually repairing hunk headers;
- merge conflicts appear as ordinary source conflicts; and
- malformed unified-diff syntax cannot break that replacement.

It does not simplify the runtime design. The same methods still need to route
to Gomad before host dispatch, preserve normal behavior while Gomad is disabled,
and fail closed for unsupported operations. The existing `net/gomad.go` and
`os/gomad.go` files already hold most of the deep implementation; the patch
hunks are intentionally thin adapters at standard-library boundaries.

## Costs of replacing whole files

### Review and correctness

A patch makes the exact divergence from Go 1.26.4 visible. A replacement makes
the repository copy look authoritative even though almost all of it remains
upstream-owned. Reviewers must generate a separate diff to distinguish Gomad
logic from copied Go logic.

The most dangerous failure is silent upstream drift during a Go upgrade. A
patch normally conflicts near changed upstream code. A replacement can instead
overwrite the new upstream file with an old copy and still compile, omitting a
bug fix, security fix, new invariant, or platform behavior. The globally pinned
source archive prevents drift within Go 1.26.4, but it does not make the next Go
upgrade safe.

### Complexity

The build currently gives `overlay/` a simple invariant: every file is new, and
any collision with upstream source is an error. Allowing arbitrary collisions
would weaken a useful safety boundary. Replacement files need an explicit,
separate contract rather than a general removal of the collision check.

The two source-inspection tests in
`internal/ioprofile/network_patch_test.go` and
`internal/ioprofile/filesystem_patch_test.go` also read
`go1.26.4.patch` directly. A replacement would require those audit gates to
inspect the materialized source or a normalized generated diff instead of
assuming every hook lives in the patch.

### Performance and scale

There is no target runtime or 10x-load benefit: the compiled source is the same.
Hashing and copying a few hundred or even a few thousand extra lines is
negligible compared with rebuilding Go. The relevant scaling cost is human:
the number of copied upstream files that must be audited and rebased on each Go
upgrade.

### Security

The patch representation fails conspicuously when context no longer matches.
A stale replacement can fail silently. That matters here because the hooks are
the boundary preventing modeled I/O from falling through to the host. Any
replacement scheme must preserve explicit base-version verification and
disabled-mode compatibility tests.

## A safe hybrid design

If `src/net/tcpsock.go` is converted, keep replacement semantics distinct from
the additive overlay. For example:

```text
overlay/                         # files absent from upstream
replacement/go1.26.4/src/net/
  tcpsock.go                     # complete patched upstream file
go1.26.4.patch                   # all remaining surgical changes
```

The build contract should be:

1. Snapshot the patch, additive overlay, and replacement tree.
2. Verify the official Go archive checksum.
3. Require every replacement path to appear in a small explicit allowlist.
4. Verify the SHA-256 of each replacement's unmodified upstream base file.
5. Reject a path present in both the patch and replacement tree.
6. Apply the surgical patch, then copy the replacement files.
7. Include all three snapshots and the replacement manifest in the build key.
8. Generate or print the upstream-to-replacement diff for review.

Keep the existing collision rejection for `overlay/`; only the explicitly
versioned replacement tree may collide. This makes the exceptional ownership
obvious and prevents an accidental file copy from becoming a standard-library
fork.

A useful default threshold is to consider replacement only when a file is
under roughly 1,000 lines and either has at least 10 dispersed hunks or a custom
delta around 15% or greater. This is a judgment aid, not a validator rule. It
selects `net/tcpsock.go` today without pulling in `net/net.go`, `os/file.go`, or
large runtime files.

## Immediate patch-validation finding

At the inspected working-tree state:

- `make -C tools/gomadv3 validate` succeeds;
- `git apply --check < tools/gomadv3/go1.26.4.patch` rejects the patch as
  corrupt at line 156; and
- production-equivalent `/usr/bin/patch --dry-run --batch -p1 -F 0` against
  the pinned Go 1.26.4 archive reports one failed hunk in `src/os/file.go`.

Some unchanged lines in the `runtime/rand.go` and `testing/testing.go` hunks
lack the unified-diff context prefix. The system `patch` command accepts those
leniently, but Git does not. Separately, the final `src/os/file.go` hunk does
not apply in the production dry run. Several accepted hunks also require line
offsets, which indicates that the hunk headers were not freshly generated from
the exact pinned base.

`test.sh validate` currently checks header counts, path allowlists, file kinds,
and generated/binary markers; it does not parse the patch or prove that it
applies. `build.sh` catches applicability later, after source extraction, but
the standalone `validate` target gives a false green result.

The immediate fix should be procedural and automated rather than another
manual hunk edit:

1. Materialize the exact checksum-pinned Go 1.26.4 source.
2. Apply the intended Gomad changes to normal source files.
3. Generate `go1.26.4.patch` from that clean base.
4. Add a syntax check such as `git apply --numstat` to `validate`.
5. Before compilation, run an applicability check against the extracted pinned
   source using the same operation that will apply the patch.

That workflow removes most patch-authoring pain while retaining a compact,
auditable delta. If `net/tcpsock.go` remains a frequent editing hotspot after
the workflow is fixed, move only that file to the guarded replacement mechanism.

## Implementation details

Implement this in two separately verifiable changes. First make the existing
patch reproducible and fail validation when it is malformed or stale. Then add
the replacement mechanism and migrate `src/net/tcpsock.go`. Keeping those
changes separate makes it possible to prove that the repaired patch preserves
current behavior before changing its representation.

### Representation invariants

The materialized source tree has three mutually exclusive input classes:

| Input | May exist upstream | Ownership rule |
| --- | --- | --- |
| `go1.26.4.patch` | Yes | Small edits to explicitly allowed upstream files |
| `overlay/` | No | Complete Gomad-owned additive files |
| `replacement/go1.26.4/` | Yes | Complete files listed in a versioned manifest |

No source path may occur in more than one class. The additive overlay keeps its
current collision rejection. A replacement is accepted only when its path is
allowed, its upstream base has the expected digest, and its checked-in contents
are a regular text source file. All validation happens against immutable
snapshots, and the identity of every snapshot participates in the build key.

Use this layout:

```text
tools/gomadv3/
  replacement/go1.26.4/
    UPSTREAM_SHA256SUMS
    src/net/tcpsock.go
```

`UPSTREAM_SHA256SUMS` contains strict records of the form
`<lowercase-sha256><two spaces><relative-path>`. The digest is for the official
unmodified Go 1.26.4 file, not for the replacement. Initially it has exactly
one record for `src/net/tcpsock.go`. The replacement tree digest separately
binds the replacement contents into the build key.

The manifest is both an upgrade guard and an ownership declaration. Keep an
explicit `validate_replacement_path` allowlist in `test.sh` as a second boundary;
initially it accepts only `src/net/tcpsock.go`. Adding another replacement then
requires a visible validator change, a manifest record, and the complete file.

### Phase 1: regenerate and validate the surgical patch

Add `tools/gomadv3/regenerate-patch.sh` with one narrow interface: given an
unbuilt, modified Go 1.26.4 source root, generate `go1.26.4.patch` from the
official checksum-pinned base. The script should:

1. Require an explicit candidate source root and verify its `VERSION` is
   `go1.26.4`.
2. Reuse the archive URL, version, and SHA-256 constants used by `build.sh`;
   move those values into `toolchain-version.sh`, sourced by both scripts, so
   regeneration and production builds cannot silently select different bases.
3. Extract the verified archive into a temporary directory and initialize a
   temporary Git index from the pristine source. Do not commit or use a
   worktree.
4. Copy only modified, allowed upstream files from the candidate tree. Reject
   additions, deletions, symlinks, generated files, binary data, and paths
   owned by `overlay/` or the replacement tree.
5. Run `gofmt` on changed Go files with the explicitly selected bootstrap
   toolchain before producing the diff.
6. Generate a Git-format diff with `a/` and `b/` paths, stable `LC_ALL=C`
   ordering, and no timestamps or absolute temporary paths.
7. Validate the generated temporary patch and replace `go1.26.4.patch`
   atomically only after every check passes.

For the one-time repair, create the candidate tree from the pinned archive,
apply the current patch with the production `patch` command, resolve the
`src/os/file.go` reject against the intended Gomad code, and compare every
changed file with the existing patch before regeneration. This recovery step
must not become part of the normal script: after the repaired patch exists, a
rejected hunk is an error rather than input to an automatic merge.

Extend `validate_patch` in `test.sh` with a real parser check:

```bash
if ! git apply --numstat <"$patch_file" >/dev/null; then
    printf 'gomadv3 patch is malformed\n' >&2
    exit 1
fi
```

Keep the existing header, path, generated-code, and binary checks. Parser
success proves syntax only, so `build.sh` must also run
`patch --dry-run --batch -p1 -F 0` against the freshly extracted source before
mutating it. Apply the patch with the same options only after the dry run
succeeds. This keeps `make validate` fast and offline while making the build
prove applicability to the checksum-pinned source before compilation.

Add negative harness cases for malformed context lines, inconsistent hunk
counts, a syntactically valid but nonapplying patch, and a patch requiring
fuzz. Every case must fail with a specific diagnostic and leave the published
toolchain and `build-key` unchanged.

The Phase 1 completion gate is:

- `git apply --numstat < tools/gomadv3/go1.26.4.patch` succeeds;
- zero-fuzz dry-run and application succeed on a clean Go 1.26.4 archive;
- the rebuilt toolchain passes disabled-mode and Gomad-enabled tests; and
- the regenerated patch contains the same intended interception surface,
  including `src/net/tcpsock.go` for this phase.

### Phase 2: add guarded replacements

Update `build.sh` to accept
`GOMADV3_REPLACEMENT_DIR`, defaulting to
`replacement/$go_version`. Treat it exactly like the existing patch and
overlay inputs:

1. Validate the caller-provided patch, overlay, and replacement inputs.
2. Snapshot all three beneath `.toolchain` before hashing or building.
3. Revalidate the snapshots so a concurrent input change cannot bypass
   validation.
4. Hash replacement paths and contents in bytewise path order. Include the
   manifest in that identity and add the resulting digest to `build_key`.
5. Increment `build_environment` from `canonical-v4` to `canonical-v5` so no
   toolchain produced by the old materialization contract is reused.

Add `validate_replacement` to `test.sh`. It should reject:

- a missing manifest or source tree;
- malformed manifest lines, duplicate paths, digests that are not exactly 64
  lowercase hexadecimal characters, absolute paths, `..`, backslashes, and
  newlines;
- a manifest path without exactly one replacement file, or an unlisted file;
- any path other than the explicit replacement allowlist;
- directories containing symlinks or other non-regular entries;
- NUL bytes or generated-code markers; and
- any path also named by the patch or additive overlay.

Expose patch path enumeration as one helper inside `test.sh` instead of
maintaining multiple AWK parsers. `validate_patch` uses it for the existing
allowlist, and `validate_replacement` uses the same normalized set for overlap
checks. This is the deep boundary for source ownership: callers ask for the
validated path sets without depending on diff-header details.

After extraction, `build.sh` materializes inputs in this order:

1. Verify the SHA-256 of each pristine upstream replacement path against
   `UPSTREAM_SHA256SUMS`. Report the relative path on mismatch.
2. Reject additive-overlay collisions with the pristine tree as today.
3. Dry-run and apply the surgical patch with zero fuzz.
4. Copy the additive overlay, which may still create only absent paths.
5. Copy each replacement over its verified upstream path.
6. Compile and publish only after the complete source tree is materialized.

The base digest check must happen before patching or copying. Patch/replacement
overlap is already invalid, but verifying the pristine file makes the failure
mode explicit and ensures a Go upgrade cannot be hidden by a stale full-file
copy. All failures occur in the disposable work directory; the existing atomic
publication and immutable-key locking behavior remains unchanged.

Update `Makefile` so the manifest and every replacement file are prerequisites
of `toolchain`. Update `README.md` and `ARCHITECTURE.md` to describe three input
classes, their collision rules, the replacement base check, and the expanded
build-key identity.

### Phase 3: migrate `src/net/tcpsock.go`

Start from the Phase 1 materialized source, not by manually reconstructing the
file from patch hunks:

1. Copy the fully patched `src/net/tcpsock.go` into
   `replacement/go1.26.4/src/net/tcpsock.go`.
2. Run the Phase 1 Go 1.26.4 toolchain's `gofmt` and compare the replacement
   with the official source; the diff must contain exactly the Gomad changes
   previously carried by the 18 patch hunks.
3. Record the official file's SHA-256 in `UPSTREAM_SHA256SUMS`.
4. Restore `src/net/tcpsock.go` to its pristine version in the patch-generation
   candidate and regenerate `go1.26.4.patch`.
5. Prove that the new patch has no `src/net/tcpsock.go` header and that the
   materialized source is byte-for-byte identical to the Phase 1 source.

Do not move `src/net/tcpsock_unix.go`: its `SetKeepAliveConfig` interception is
a small platform-specific hook and remains surgical. This also means the
network audit must inspect both the replacement-backed `tcpsock.go` and the
patch-backed platform file after materialization.

Provide a review command, either in `regenerate-patch.sh` or as a Make target,
that prints a normalized diff between every verified upstream base file and
its replacement. It is a review aid only; validation continues to rely on the
manifest and content digests rather than a checked-in generated diff.

### Representation-independent audit tests

The source-inspection tests should verify the source actually compiled, not
the repository representation that supplied it. Add a small test helper in
`internal/ioprofile/toolchain_source_test.go`:

```go
func readToolchainSource(t *testing.T, relativePath string) string
```

The helper reads `.toolchain/build-key`, resolves the immutable build root, and
reads the requested source beneath it. It rejects absolute or escaping paths
and reports setup failures with the standard `testing` API. The nested Gomad
module currently has no Testify dependency, so this change must not add one
only for this helper. This gives the audit tests one stable interface if
another file later crosses between patch and replacement form.

Change `network_patch_test.go` to inspect the materialized
`src/net/tcpsock.go` and `src/net/tcpsock_unix.go`. Preserve the current checks
that each concrete `TCPConn` method reaches `gomadConnection(c.fd)` before host
dispatch. Change `filesystem_patch_test.go` to inspect the materialized files
containing `Hostname`, `Mkdir`, `MkdirAll`, and `Stat`, while preserving the
same `gomadIOEnabled` ordering assertion.

Add builder tests covering:

- the valid single-file replacement;
- upstream base digest mismatch;
- changed replacement contents producing a different build key;
- missing, extra, duplicate, prohibited, binary, generated, and symlinked
  replacement files;
- patch/replacement and overlay/replacement overlap;
- mutation of the caller's replacement directory after snapshotting; and
- failure before publication preserving the previous stable toolchain.

Behavioral tests remain the final authority. Run the existing network and
filesystem fixtures in enabled mode, and run disabled-mode standard-library
tests to prove that the replacement retains upstream behavior whenever Gomad
is inactive.

### Error handling and operational behavior

Diagnostics should identify the input class and relative path, for example
`gomadv3 replacement base checksum mismatch: src/net/tcpsock.go`. Never continue
after a validation, digest, dry-run, copy, or formatting failure. Temporary
files remain under the existing cleanup trap, and publication remains the last
atomic operation.

A crash or killed build can leave only an unreferenced temporary directory or
immutable incomplete build, both handled by the current cleanup and incomplete
build replacement paths. A concurrent build observes the same content-addressed
key and lock. A concurrent repository edit cannot affect an in-flight build
because patch, overlay, manifest, and replacement files are snapshotted before
the key is computed.

At 10x the present replacement count, validation and hashing remain linear in
a few thousand source lines and negligible beside a Go bootstrap. The explicit
allowlist intentionally prevents that count from growing accidentally. There
is no runtime performance or scalability change because the compiled source is
identical; the cost is maintenance during Go upgrades.

The security trade-off remains deliberate: a full-file copy is easier to edit
but more capable of concealing upstream changes. The pinned archive checksum,
per-file upstream digests, exact allowlist, overlap rejection, normalized review
diff, and disabled-mode tests collectively restore the conspicuous failure
behavior that a surgical patch provides by default.

### Verification and rollout

Run each gate before proceeding to the next phase:

```text
make -C tools/gomadv3 validate
make -C tools/gomadv3 toolchain
make -C tools/gomadv3 runner-test
make -C tools/gomadv3 test
make fmt-imports
make lint-code
git diff --check
```

All Go test commands continue to include `-tags test_dep` through the Gomad
Make targets. If a focused test is run directly, it must include that tag.

Land Phase 1 and Phase 2/3 as distinct review units if practical. Phase 1 must
show only patch repair, regeneration tooling, and validation. Phase 2/3 must
show a byte-identical materialized toolchain source for `net/tcpsock.go` before
and after migration. Do not add `net/net.go`, `os/file.go`, or another
replacement until a separate review demonstrates that it meets the size/hunk
threshold and accepts the corresponding upstream-maintenance cost.
