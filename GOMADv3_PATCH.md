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
