# fn-155.1 routing source audit

The targeted correction resolves the audit's sole Important source defect. Canonical ownership loading now rejects admitted host directory and ancestor symlinks before Git inventory or lint dispatch.

## Scope and provenance

- Independent fresh source correctness review of the four admitted routing files, including staged changes against candidate HEAD.
- Primary root `P=/Users/stephan/Workspace/skunkworks/gomad/temporal`, HEAD `ee16212a4de034c2120843fb88254fb1fb7883b1` supplied by root.
- Candidate `C=P/.worktrees/fn-155-gomad-syscall-level-io-boundary-from`, observed HEAD `c8b811d5344fb85e347b6db296998dd0feedc4ab`.
- Requested reviewer `gpt-6.1-sol` at high, same GPT family as the writer. Session fallback `jev-unavailable(no_key)` was supplied by root; actual-model telemetry is unavailable.
- Read primary AGENTS.md, Gomad README, MILESTONES.md, task .1 including Host-runner gate routing admission, requesting-code-review skill/template and the Flow prose contract.
- Read the actual architecture and lint consumers, including `Program.Load`/`Import`, `PackageEdges`, canonical Make targets and fixture construction.
- No Go, compiler, build, lint, generator or native execution; no bridges, additional agents, Git/index/HEAD/Flow mutation or product edits.
- The initial shell resolved P instead of C. Those initial primary-tree hashes were discarded. Every source input below was consumed after explicit `cd C`.

## Initial frozen inputs

The following SHA-256 values matched before and after review.

| Candidate path | SHA-256 |
| --- | --- |
| `tools/gomad3/internal/gomadtool/architecture/architecture.go` | `a99690a06c0df5a27bb373b05903177c1f4cc0f70de8ad4365888fbf76eb4d55` |
| `tools/gomad3/internal/gomadtool/architecture/architecture_test.go` | `95a121cec557f4592423f80f36527954e37f057c01b4084c4712429faf2f6e85` |
| `cmd/tools/lintcode/main.go` | `7210b3531ed1c51e0937773024e08b8d56a99ad64c46e625c8beb1d06d14942b` |
| `cmd/tools/lintcode/main_test.go` | `14ca5f0d94a6dd63b2c2065e81e4e800856f61e3d968942785991fbddac7afd7` |

## Strengths

- Architecture owns the single literal list of `vfdpointer` and `vfdnative`; lint reads that declaration rather than maintaining a second authority.
- Classification matches each exact directory, keeps normal Gomad ownership and empty disposition, and preserves rejection of siblings, prefix lookalikes, nested directories and unknown modules.
- Explicit List patterns supply both production/test metadata; existing `Load` consumes this List for production source type/import analysis. Existing owner/edge rules and source exclusions remain unchanged in the reviewed diff.
- Canonical fast/module-wide tests exercise real Go package listing and actual Make routing, with recorded golangci/vet substitutes and controlled tool failures. Package Error/DepsErrors now stop lint dispatch.

## Initial issues

### Critical

None found in the frozen routing correction.

### Important

1. `cmd/tools/lintcode/main.go:73`, `:280`, `:435`; `main_test.go:87`.
   Canonical lint inventories only Git paths matching `*.go` or `go.mod`. Git stores a directory symlink itself and does not enumerate its target files. A new or staged symlink named `tools/gomad3/toolchain/runtime/testdata/vfdpointer` therefore contributes no source entry, and `loadOwnership` never inspects that admitted directory. `regularSource` runs only for supplied sources, so it cannot reject this omitted directory.
   With the symlink and no cached child file paths, `lint-code-gomad3` can lint the other ordinary packages and succeed while omitting this runner. Fast lint likewise has no matching changed source to select. This violates the admission requirement to preserve symlink rejection in canonical gates.
   The directory=true regression directly supplies a fabricated `source.go` path to `coveredPackages`. It proves that the helper rejects an ancestor symlink after receiving a source, but does not prove that canonical inventory finds the symlink.
   Validate admitted host directories and their ancestors independently of Git's Go-file inventory, retaining absent-directory behavior. Add a regression that invokes canonical Make with this directory symlink, requires failure before tool dispatch, and covers both fast and module-wide entrypoints.

### Minor

None found requiring a separate change.

## Parent-retained evidence consumed

All paths below are under `C/.flow/tmp/`. These are root-retained logs, not independently executed gates or reconstructed command receipts.

| Log | Observed result | SHA-256 |
| --- | --- | --- |
| `fn155-routing-architecture-red.log` | Meaningful omitted inventory/import regression failure | `54094f75730e2c7b858e9e65c4c788927cfcd831ab53431883390d5c83a9b389` |
| `fn155-routing-architecture-green.log` | Whole architecture package `ok`, 120.799s | `3ac17f3205eba0e656c9d58230a3fb343a5b7d0ecee1ba1324e52a8bb02f57d8` |
| `fn155-routing-lint-red.log` | Build failure from shadowed source identifier | `f5a3342e09c4d84d2cd72dbdb0f477c37bc50bb45303b84f78a928d86120f6a1` |
| `fn155-routing-lint-red-2.log` | Compiled behavioral RED, uncovered runners, missing dispatch and malformed/import controls; 4.121s | `95a763adc4b3f102412f094579ba1f430fc7716e990dfedda9fba0736314c84d` |
| `fn155-routing-lint-green.log` | Whole lintcode `ok`, 10.136s before final style corrections | `59f9acb2a8731a62c5af7f268985ef5682b0d290f27032275b38f39e8d8aac07` |
| `fn155-routing-lint-green-2.log` | Whole lintcode FAIL, TestGomadDefaultRemainsGenerate clock-skew warning; 8.430s | `d864239a59fe8a475f23b92a62320101251728d2cda20169fb6856e4c179fcff` |
| `fn155-routing-lint-focused-green.log` | Scoped routing tests `ok`, 4.920s | `a002cc29cd2d28201ec9143df3cee7a1ad0033c174bbc167ea905300807838b8` |
| `fn155-routing-symlink-red.log` | Canonical fast/full wrongly succeed for staged runner-directory and testdata-ancestor symlinks; 0.841s | `73f9758eead150de04a4326472614a11bd812b8264beb4020e418d61c8853779` |
| `fn155-routing-symlink-green.log` | Affected routing suite `ok`, 5.150s before lowercase diagnostic correction | `9a56207f14d6da4cc0feb517f12ee0bd09658f0f3a8376e33790e6da3d29c8f9` |

The initial `red.log` is a build failure and supplies no behavioral RED credit. Root supplied `red-2.log` as the compiled retry after correcting the shadowed local identifier; its observed failures establish the behavioral baseline.

## Targeted recheck and final inputs

Architecture inputs retain their initial hashes and were not re-reviewed. Final candidate HEAD remains `c8b811d5344fb85e347b6db296998dd0feedc4ab`.

| Candidate path | Final SHA-256 |
| --- | --- |
| `cmd/tools/lintcode/main.go` | `f7ecb5ea8c1686defa55141fdfbcec7d56212ed6af22972a1802966c5dc6f761` |
| `cmd/tools/lintcode/main_test.go` | `2c61b03aadfc793946a52374930e5b4575624f1f51216c87a1d40e34678160ed` |

- `main.go:286` calls `regularHostDirectory` for every entry of the shared literal before `lint` inventories Git paths at `:73`. The check does not depend on cached or untracked child source paths.
- `main.go:508` walks each admitted directory and every ancestor below root. Missing children continue upward, preserving absent runner behavior while still rejecting a symlinked parent. Existing entries must be directories; Lstat and other errors propagate.
- `main_test.go:82` stages both runner-directory and testdata-ancestor symlinks and invokes actual `repo.runMake` for fast and module-wide gates. Each requires a source-symlink error and empty tool calls. The fabricated-directory-source helper test was removed; the separate canonical file-symlink test remains.
- The affected suite also retains normal package dispatch, missing-import and malformed-list controls. No second hardcoded list or owner/exclusion/policy expansion was added.
- The inspected pre-style main hash was `328001d17dc2fb7a8cb482f6671171e2b49d02bca63fb360bd5bbea18a3a6b64`. Reversing only the final `Gomad` to `gomad` diagnostic change in a read-only stream reproduced that hash exactly. Final hashes matched before and after recheck.
- No additional Critical, Important or Minor source defect was found in this targeted correction. The initial Important finding is resolved for these final source inputs.

## Recommendations and assessment

Root accepted the initial Important finding and the worker supplied the bounded correction. Preserve the full lintcode failure and its unchanged fixture; the affected-suite pass does not replace that package result.

The targeted source audit accepts the canonical directory-symlink correction. Task .1 remains Quick unqualified with its first-platform native requirement open. This report supplies no formal implementation review, SHIP, native pass, Done or merge-readiness verdict. Root's post-lowercase narrow regression and subsequent canonical staged-new-files lint were pending at final dispatch and receive no pass claim here; root owns those receipts separately.
