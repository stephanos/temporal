# fn-113 R1: pin baseline

Measured at `6be0755fe` (2026-10-02) with `measure-pins.sh`, which reads the committed tree
through `git archive`, so the linux/arm64 development harness in the working tree does not
affect the counts. Raw output: `pin-baseline.txt`. fn-105 task 8 had already landed (15
adapters); fn-110 task 1 measured the patch at `6782b55f4` (1007 lines, 57 overlay files,
17,105 lines), and the runtime work after it grew both.

A manual step is one command invocation or one hand edit of a checked-in file (spec R5
definition). Builds and tests that run only to read a digest from a failure count as
commands.

## Pin classes and counts

| Class | Count at `6be0755fe` | Assessment (2026-10-01) |
| --- | --- | --- |
| Go release | `go1.27.1`, 1 archive SHA-256, 2 platforms | same |
| Runtime patch | 1045 lines, 20 files, 74 hunks; allowlist 20 | 1010 lines, 20 files |
| Runtime overlay | 61 files, 18,423 lines; allowlist 61 | 57 files, 17,105 lines |
| Boundary manifest | 131 intercepts; 132 declaration fingerprints (131 + 1 linux/amd64 override); 131 package fingerprints; 102 reviewed candidates; 1 hook policy | 131 intercepts, 132 fingerprinted entries |
| Host-clock inventory | 25 references (10 darwin/arm64, 15 linux/amd64) | not counted |
| Goroutine creation inventory | 10 sites | not counted (added after) |
| Dependency adapters | 15; 135 SHA-256 literals, 125 distinct, 30 per-platform prepared-source-set pins; 2 adapter templates; 6 modules absent from root `go.mod` | 15 adapters, 129 anchors |
| Compatibility packs | 12 packs, 54 rules, 19 module-version pins over 17 modules, 7 adapter replacements, 672 Go and 58 foreign source digests; 12 requests, 12 reports, 37 generation outputs; 9 darwin/arm64 and 3 linux/amd64 packs | 12 packs, 54 rules, 19 pins |
| Upstream `go.mod` churn | 73 upstream commits, 4 `go` directive bumps (2026-04-01 to 2026-10-01) | same |

Corrections made in `MILESTONES.md#maintenance-cost`: patch and overlay sizes, the adapter
SHA-256 count (129 matched no definition; it is 135 literals, 125 distinct, and the count did
not change between the assessment revision `1d7272e65` and `6be0755fe`), and a new row for
the two test-pinned toolchain inventories.

## Manual steps per bump today

### One adapted module (for example `google.golang.org/grpc`)

1. `go get MODULE@VERSION` in the target module (the bump itself).
2. Hand edit `toolchain/version/version.json` adapter identity (version, sum).
3. `make -C tools/gomad3 generate` (descriptor consumers and upgrade guide).
4. Hand edit `deterministicio/<name>_adapter.go`: version, sum, original inventory, one source
   and one replacement digest per rewritten file, replacement inventory, and the per-platform
   prepared source set (6 to 23 literals per adapter).
5. `go test ./deterministicio -run <Adapter>` repeatedly: preparation stops at the first
   mismatched digest (original inventory, then each source, replacement, replacement
   inventory, prepared set), so about 2 + 2 x rewritten files runs, plus one on the other
   platform for its prepared source set. Changed upstream text also needs hand edits of the
   anchors and replacements, with no tool that shows which anchor moved.
6. Every pack that replaces the module with this adapter (libc and memory: 7 activations in 4
   packs) needs the pack refresh below.
7. `make -C tools/gomad3 validate` and the platform gates on darwin/arm64 and linux/amd64.

Minimum: 1 bump command, 2 hand edits, about 5 to 10 commands, more for anchor repair.
No command reports the affected adapter ahead of the build failure.

### One packed module (for example `github.com/klauspost/compress`)

Per invalidated request (a klauspost bump invalidates 4 packs on 2 platforms, see
`root-bump-report.txt`):

1. `gomadtool compatibility-pack discover --request=... --working-dir=TARGET` on the pack's
   platform.
2. `gomadtool compatibility-pack review --output=reports/ID.md`, then a person reviews.
3. `gomadtool compatibility-pack generate --approve-review=DIGEST`.
4. `make -C tools/gomad3 validate compatibility-pack-qualification` (or `compatibility-pack
   qualify`).

Four commands per request, run on its platform, plus one hand deletion each of the request,
report, pack, and `generation.json` entry for a stale variant nothing selects.

### One Go release (interception fingerprints, toolchain inventories, patch, overlay)

1. Hand edit `toolchain/version/version.json` (release, archive digest, manifest version,
   patch path, allowlists) and `deterministicio/boundary/manifest.json`.
2. `gomadtool patch-materialize`, a manual rebase of each rejected hunk, then
   `gomadtool patch-regenerate --candidate-root=...`.
3. Hand port of changed overlay files.
4. `gomadtool boundary-generate --refresh`, review of changed fingerprints.
5. `make -C tools/gomad3 generate`.
6. `make test-toolchain` on each platform, then hand edits of
   `toolchain/clock_inventory_test.go` and `toolchain/goroutine_inventory_test.go`.
7. `make upgrade-dossier GOMAD3_BASELINE_REF=...`, rerun with
   `GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256=...` after review.

At least 8 commands and 4 hand-edited files, plus a variable number of patch and overlay
edits. Go releases are outside fn-113 (fn-110 and the upgrade dossier own them); the pin
impact report only marks their pins unknown when a candidate needs a newer Go.
