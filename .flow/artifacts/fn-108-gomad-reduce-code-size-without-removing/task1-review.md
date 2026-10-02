# fn-108.1 implementation review

Raw codex bridge on the untracked working-tree files (commits are forbidden in this run, so no
commit range exists). Model `gpt-5.6-sol` at high reasoning effort, read-only sandbox. Four
rounds; the conductor authorized the fourth after the three-round cap ended at NEEDS_WORK.

Rounds 1 and 2 are reconstructed from the worker's transcript. Their output files were lost
when a session restart cleared the scratch directory. Rounds 3 and 4 are verbatim.

## Round 1 - VERDICT: NEEDS_WORK

1. High, `size-count.sh` and `baseline.md`. Collapsing a gofmt-clean multi-line call into one
   line removes code lines and the mandatory trailing comma, lowering `code` and `codebytes`
   without simplifying anything. Fixed: counting rule v2 excludes commas and semicolons outside
   literals from Go code bytes; re-verified against `go/scanner` on all 585 Go files.
2. High, `size-count.sh`. `--exclude-standard` omits ignored untracked files, and an ignored
   `.go` file still takes part in a package build. Fixed: any ignored path other than
   `tools/gomad3/.toolchain/` and `tools/gomad3/.bin/` fails the script.
3. High, `baseline.md`. Growth in overlay, generated, protocol and other classes was reported
   but never deducted from the production reduction. Fixed: `size-compare.sh` computes a
   residual and fails on a file that changed class.
4. Medium, `api-capture.sh`. The outside-repository check was lexical, an existing BINDIR could
   hold symbolic links, and the write-safety claim ignored the ambient Go caches. Fixed: parent
   directories are resolved physically, BINDIR must not exist, and the claim is corrected. The
   Go caches were not relocated because that forces a full rebuild and module download per
   capture.
5. Medium, `api-capture.sh`. `go list -e` and failing `go doc` commands were masked. Fixed: a
   public package with a load error and any failing `go doc` fail the script. `-e` stays because
   the runtime overlay packages cannot load outside GOROOT.

## Round 2 - VERDICT: NEEDS_WORK

1. High, `size-compare.sh`. Offsetting growth used each class's net change, so shrinking one
   protocol file could hide production code moved into another. Fixed: positive per-file growth
   is summed and a new file contributes its whole size.
2. Medium, `api-capture.sh`. The capture depended on ambient GOENV, GOOS, GOARCH and
   GOEXPERIMENT, the gomad3sim command lacked `GOWORK=off`, and `-mod=mod` could write module
   files. Fixed: one pinned environment for every Go invocation; a run with
   `GOOS=linux GOARCH=amd64 GOFLAGS=-mod=mod` exported reproduced `api-baseline/` exactly.
3. Medium, `baseline.md`. No digest was recorded for `api-capture.sh`. Fixed.

## Round 3 - VERDICT: NEEDS_WORK (verbatim)

1. **High — `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/baseline.md:194`**: Only the three scripts are hashed, while the untracked `size-baseline-files.txt`—which `size-compare.sh` trusts without reconciling its detail rows against the summary—is mutable. Editing those rows can manufacture an R1 pass; the API and gate baselines are similarly unprotected. Before source edits, record and require verification of hashes for both size baselines, a canonical `api-baseline/` manifest, test dispositions, and gate results.

VERDICT: NEEDS_WORK
Fix: `SHA256SUMS` lists the SHA-256 of all 74 baseline files; `baseline.md` records the
manifest digest and makes its verification a prerequisite of the final comparison.

## Round 4 - VERDICT: SHIP (verbatim)

Round-3 finding is resolved: manifest digest `a64ac192…` matches, all 74 files verify, and both size baselines reproduce exactly. No remaining findings; `baseline.md` digest is `b3a8c0dd…` for the tracked done summary.

VERDICT: SHIP
## After the verdict

`baseline.md` gained one file-table row and one clause naming this review record as outside
`SHA256SUMS`. That wording change is applied and unreviewed. No script or baseline output
changed after round 4.
