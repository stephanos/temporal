# fn-108 implementation baseline

Recorded 2026-10-01 by task fn-108.1, before any fn-108 source edit. The final task (fn-108.7)
reruns `size-count.sh` and `api-capture.sh` unchanged and compares against the files in this
directory.

## Revision and working tree

| Item | Value |
| --- | --- |
| Baseline revision | `6782b55f49a0317b230e827ea2a63a37d116d502` (`wip`, 2026-10-01T08:45:44-07:00) |
| Branch | `stephanos/gomad`, up to date with `stephanos/gomad` on the remote |
| Planning-time revision | `d4d800fb47`, two commits behind the baseline; superseded by a conductor decision |

The task text asked for a clean tree at `d4d800fb47`. The conductor replaced that with the
current HEAD, which already contains the fn-105/fn-111 work of 2026-10-01, including the D14
runtime fix and the D13 generator change.

The working tree is not clean. `git status --short` lists 80 paths before this task wrote its
artifact directory: 5 modified tracked files and 75 untracked files, all under `.flow/`,
`.plans/GOMAD_MILESTONES.md`, and `docs/research/gomad/`. The full list is
[git-status-baseline.txt](git-status-baseline.txt). Nothing is staged.

The size inventory equals the baseline revision's:

- `git diff --stat HEAD -- tools/gomad3 tools/gomad3sim tools/gomad3integration` is empty.
- `git status --short --untracked-files=all -- tools/gomad3 tools/gomad3sim tools/gomad3integration`
  is empty, so no untracked file exists under the three directories.
- The 718 paths counted by `size-count.sh` equal
  `git ls-tree -r --name-only HEAD -- tools/gomad3 tools/gomad3sim tools/gomad3integration`.
- Both checks were repeated after the gate runs below with the same result.

## Host and toolchains

| Item | Value |
| --- | --- |
| Platform | `Darwin arm64` (`uname -sm`), macOS 26.6.2 (25G83) |
| `go` on the default PATH | `go version go1.27.0 darwin/arm64` (`/opt/homebrew/bin/go`) |
| `go` used for every command here | `go version go1.27.1 darwin/arm64`, from `$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin` prepended to PATH |
| `tools/gomad3/.toolchain/bin/go` | `go version go1.27.1 darwin/arm64` (patched toolchain) |
| Toolchain build key | `8d28bd4486f0b6300e8d25efd4caf8cb6ccbf000e96dbd26b1d8f53bf5f251bc` |
| Counting tools | `git version 2.54.0 (Apple Git-157)`, `/usr/bin/awk` version 20200816 |

`make -C tools/gomad3` needs go1.27.1 as the host `go`, so the default PATH `go` (1.27.0) cannot
run the gates. Every gate and capture below ran with the PATH override in the table.

## Counting rule (v2)

`sh .flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/size-count.sh` takes no
arguments, reads the working tree, and writes only to stdout. It runs from any directory inside
the repository. `SIZE_COUNT_DETAIL=1` appends one row per file.

**Inventory.** Every path under `tools/gomad3`, `tools/gomad3sim`, and `tools/gomad3integration`
listed by `git ls-files --cached --others --exclude-standard`, sorted in the C locale, that
exists as a regular file. This is the tracked inventory plus untracked files that git would
track. The task text specified tracked files only. fn-108 forbids commits and staging, so a
helper file added by a later task stays untracked and a tracked-only inventory would never count
it. The build-output roots `tools/gomad3/.toolchain/` and `tools/gomad3/.bin/` are ignored by
git and never count. Any other ignored path under the three directories makes the script fail,
because an ignored `.go` file still takes part in a package build and would escape both the
count and `git status`. At the baseline the two inventories are identical (718 paths) and the
two build-output roots are the only ignored paths. The script exits nonzero on a symbolic
link, a non-regular file, a path with a character outside `[A-Za-z0-9._/+@-]`, an unreadable
file, or an empty inventory.

**Classes.** Each file belongs to exactly one class. The first matching row wins.

| Class | Rule |
| --- | --- |
| `generated-go` | `*.go` with a line matching `^// Code generated .* DO NOT EDIT` before the `package` clause |
| `test-go` | remaining `*_test.go`, and remaining `*.go` with a `/testdata/` path component |
| `overlay-go` | remaining `*.go` under `tools/gomad3/toolchain/runtime/overlay/` |
| `production-go` | every other `*.go`; this is the authored production code R1 measures |
| `protocol-input` | `*.tmpl`, `*.patch`, `*.json`, `*.s`, `*.sh`, `*.mk`, `Makefile` |
| `other` | every remaining file (`*.md`, `*.txt`, `go.mod`, `go.sum`, `.gitignore`, `clock_audit.d`) |

The task text listed five classes. `other` makes the classes cover the whole inventory, so code
moved into a `.txt` fixture or an embedded text file still appears in a row.

**Units.** The script reports four numbers per scope and class.

- `files`. Number of files.
- `physical`. Lines as read by awk, blank and comment lines included.
- `code`. Lines that carry at least one code byte.
- `codebytes`. Non-whitespace bytes outside comments. Whitespace is space, tab, CR, FF, and VT.
  In Go files a comma or semicolon outside a literal is not a code byte.

For Go files a small lexer removes `//` comments and `/* */` comments. It tracks interpreted
strings, rune literals, and raw strings, so comment markers inside a literal stay code and the
content of a multi-line raw string counts. A file that ends inside a block comment or a raw
string makes the script fail. Commas and semicolons are excluded because line layout alone
decides whether a trailing comma or an explicit semicolon exists. Joining a gofmt-formatted
multi-line call into one line deletes its trailing comma, and joining two statements adds a
semicolon. With both excluded, `codebytes` of a Go file is the same for every layout of the same
token sequence. For every other file a line is comment-only when its first
non-whitespace bytes are the line-comment prefix of its kind (`//` for `*.tmpl` and `*.s`, `#`
for `*.sh`, `*.mk`, and `Makefile`, none for the rest). Every other non-blank line is code.

**Cross-check.** An independent Go program built on `go/scanner` counted, for all 585 Go files,
the lines that carry a token and the non-whitespace bytes of every token other than commas and
semicolons. Its per-file `code` and `codebytes` equal the script's for every file. A scratch
repository exercised the gaming cases with these results for one 20-line fixture file.

| Change to the fixture | physical | code | codebytes | `size-compare.sh` |
| --- | --- | --- | --- | --- |
| Original | 20 | 12 | 172 | |
| All comments deleted | 16 | 12 | 172 | FAIL, residual code +0, codebytes +0 |
| Statements and a multi-line call joined | 9 | 5 | 172 | FAIL, residual code -7, codebytes +0 |
| One unused function deleted | 18 | 11 | 148 | PASS, residual code -1, codebytes -24 |
| The same function moved into a `*.tmpl` file | 18 | 11 | 148 | FAIL, residual code +0, codebytes +0 |

The same scratch run showed that an untracked new file is counted, a file under an ignored
build-output root is not, and a symbolic link, an unterminated comment, and an ignored `.go`
file outside the build-output roots are refused. A `Code generated` header added to a production
file is reported by `size-compare.sh` as a class change and fails the comparison.

## Baseline size table

Output of `size-count.sh` at the baseline, stored as [size-baseline.txt](size-baseline.txt). The
per-file listing is [size-baseline-files.txt](size-baseline-files.txt). Two consecutive runs
produced identical bytes.

```
# fn-108 size-count, counting rule v2 (see baseline.md)
scope                    class                   files  physical      code  codebytes
tools/gomad3             production-go             227     54902     50984    1721260
tools/gomad3             overlay-go                 35     12852     11462     318423
tools/gomad3             test-go                   254     40258     37645    1268801
tools/gomad3             generated-go               19      4514      4124     213092
tools/gomad3             protocol-input             52      9003      8529     540030
tools/gomad3             other                      76      7209      5741     409596
tools/gomad3             all                       663    128738    118485    4471202
tools/gomad3sim          production-go              25      8285      7640     249064
tools/gomad3sim          overlay-go                  0         0         0          0
tools/gomad3sim          test-go                    22      5306      5038     155764
tools/gomad3sim          generated-go                0         0         0          0
tools/gomad3sim          protocol-input              0         0         0          0
tools/gomad3sim          other                       0         0         0          0
tools/gomad3sim          all                        47     13591     12678     404828
tools/gomad3integration  production-go               0         0         0          0
tools/gomad3integration  overlay-go                  0         0         0          0
tools/gomad3integration  test-go                     3       189       166       5041
tools/gomad3integration  generated-go                0         0         0          0
tools/gomad3integration  protocol-input              4      5397      5397     123215
tools/gomad3integration  other                       1       200       177       9757
tools/gomad3integration  all                         8      5786      5740     138013
total                    production-go             252     63187     58624    1970324
total                    overlay-go                 35     12852     11462     318423
total                    test-go                   279     45753     42849    1429606
total                    generated-go               19      4514      4124     213092
total                    protocol-input             56     14400     13926     663245
total                    other                      77      7409      5918     419353
total                    all                       718    148115    136903    5014043
total                    protocol-input:tmpl        12      2726      2376      89746
total                    protocol-input:patch        1      1007       956      26354
total                    protocol-input:json        37     10314     10314     532722
total                    protocol-input:s            1        35        25        693
total                    protocol-input:sh           3       134       114       3022
total                    protocol-input:make         2       184       141      10708
```

R1 baseline: `total production-go` is 252 files, 63187 physical lines, 58624 code lines, and
1970324 code bytes.

## Comparison rule for the final task

The final task runs both commands from the repository root and stores their output here.

```
SIZE_COUNT_DETAIL=1 sh .flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/size-count.sh > final-files.txt
sh .flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/size-compare.sh \
  .flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/size-baseline-files.txt final-files.txt
```

`size-compare.sh` prints the change per class and exits 0 only when the R1 size condition
holds. It computes a residual in code lines and in code bytes.

- The residual starts from the `production-go` change.
- It adds the growth of every file in `overlay-go`, `generated-go`, `protocol-input`, and the
  non-Markdown files of `other` that grew or is new. Growth is summed per file. A file that
  shrank earns nothing and never cancels growth in another file. Code moved from production
  into the runtime overlay, a generated file, a template, a patch, a schema, a script, or a text
  fixture therefore leaves the residual where it was.
- The condition holds when the residual is negative in both units and no file changed class.
  A file that changed class fails the comparison and is listed with both classes.

`test-go` and Markdown changes are printed and do not enter the residual. New characterization
tests and the R5 documentation are required work. A `test-go` reduction never offsets a
production increase and needs its own justification under R8.

R1 is met only when `size-compare.sh` exits 0 and these checks, which the scripts cannot make,
also hold.

1. The final task uses the three scripts and every stored baseline output byte-identical to
   the versions recorded here. [SHA256SUMS](SHA256SUMS) lists the SHA-256 of all 74 files in this
   directory other than this record, the manifest, and the review record `task1-review.md`: the scripts, both size baselines, every
   `api-baseline/` capture, the gate logs and results, the test dispositions, and the status
   listing. Before any comparison the final task runs `shasum -a 256 SHA256SUMS`, which must print
   `a64ac192433c03e4c7813027dd8a9832790a6fd34af0217468231ab6a1a64327`, and then
   `shasum -a 256 -c SHA256SUMS` in this directory, which must report every file `OK`. The
   directory is untracked, so the manifest is the only guard against a later edit. The fn-108.1
   done summary in `.flow/tasks/` repeats the manifest digest and carries the digest of this
   record.
2. The `generated-go` file list is unchanged, or each added file is the output of a
   `gomadtool *-generate` command that `make -C tools/gomad3 validate` checks.
3. No Go code moved out of the inventory. `git status --short` and
   `git diff --stat 6782b55f49a0317b230e827ea2a63a37d116d502` show no added Go lines outside the
   three directories that replace code removed inside them.
4. `gofmt -l` over the inventory's Go files prints nothing. It prints nothing at the baseline.
5. Every increase in a non-production class is explained in the final evidence, including the
   Markdown and test increases that the residual does not count.

Limits of the rule, stated so the final task does not over-read it:

- Compiler directives (`//go:build`, `//go:embed`, `//go:linkname`, `//go:generate`) are
  comments to the lexer. Adding or deleting one changes `physical` only.
- Shorter identifiers lower `codebytes` and leave `code` unchanged. The condition requires both
  to fall, and R8 forbids renaming public names.
- `codebytes` is layout-independent for a fixed token sequence. Edits that change tokens while
  preserving meaning, such as merging `var` declarations or removing redundant parentheses, do
  change it. The script cannot tell those from simplification. Review of the diff decides.
- Growth is measured per file. Code moved into an offsetting file while the same file loses at
  least as much other content shows no growth. Review of that file's diff decides.
- The comparison detects a class change only for a path present in both listings. A file renamed
  into another class appears as a removal and an addition, and the residual offsets it when the
  destination is an offsetting class.
- Code moved into a Markdown file or under `testdata/` leaves the residual lower. Neither place
  is compiled into a production package, so that move is a deletion of production code and is
  judged under R2 and R8.
- The script counts every inventory file regardless of build constraints, so platform-specific
  files count on every host.
- In non-Go files a trailing comment on a code line counts as code, and `/* */` comments in
  `*.tmpl`, `*.s`, and `clock_audit.d` count as code.
- `physical` equals `wc -l` except for a file without a trailing newline, which counts one more.

## Public surface capture (R8)

`sh api-capture.sh OUTDIR BINDIR` wrote [api-baseline/](api-baseline/). Neither directory may
exist. BINDIR receives the two binaries and must resolve outside the repository after `..` and
symbolic links in its parent are resolved. The script deletes nothing and writes its own files
only under the two directories. The Go commands it runs use the ambient Go build and module
caches. Every Go invocation runs with `GOENV=off`, `GOFLAGS=-mod=readonly`, `GOWORK=off`,
`GOTOOLCHAIN=local`, `CGO_ENABLED=0`, `TZ=UTC`, and with `GOOS`, `GOARCH`, and `GOEXPERIMENT`
unset, so the capture is the host-platform view whatever the caller exported. A run with
`GOOS=linux GOARCH=amd64 GOFLAGS=-mod=mod` exported reproduced `api-baseline/` exactly. It exits nonzero when a public package fails to load or `go doc` fails. Five further
runs into scratch directories produced trees identical to `api-baseline/` (`diff -r`).

- `api-baseline/go-doc/packages.txt` lists the 19 packages of the nested module outside
  `internal/`, `cmd/`, and the runtime overlay that have non-test Go files. One
  `go doc -all` capture exists per package, from `tools/gomad3/.toolchain/bin/go` with
  `GOWORK=off`. `gomad3sim.txt` is `go doc -all ./tools/gomad3sim` from the repository root.
- The task text also named `simulation/...`. `tools/gomad3/simulation` holds only schema
  templates and JSON and has no Go package. The module root package holds only
  `architecture_test.go`. Neither has a capture.
- `api-baseline/cli/` holds 39 captures: `gomad` with no arguments, `gomad --help`, and
  `--help` for each of its 15 commands; `gomadtool` with no arguments, `gomadtool --help`,
  `--help` for each of its 14 commands, `compatibility-pack` with no subcommand, and `--help`
  for its 5 subcommands. Each file records the command line, the exit status, stdout, and
  stderr separately. Both binaries were built from the baseline source with the patched
  toolchain.
- The captures are the darwin/arm64 view. `go doc` omits declarations excluded by build
  constraints on this host. No capture contains an absolute host path.

The capture covers exported declarations, doc comments, flag names, flag defaults, usage text,
stream routing, and help exit statuses. It does not cover behavior, JSON schemas, canonical
bytes, or `HostError.Reason` values. Those stay with the tests and the fixed-input evidence of
the later tasks.

## Gate dispositions

All gates ran on darwin/arm64 at the baseline revision on 2026-10-01 (18:00 to 18:04 UTC),
sequentially, with go1.27.1 on PATH. Logs are in [gate-logs/](gate-logs/). No log contains
`(cached)`, so every `go test` executed.

| Gate | darwin/arm64 | Evidence | linux/amd64 |
| --- | --- | --- | --- |
| `make -C tools/gomad3 validate` | pass (exit 0) | `gate-logs/validate.log`; includes `TestHostPacksBindCurrentProfile` | not run (host unavailable); expected to fail, see findings |
| `make -C tools/gomad3 test-harness` | pass (exit 0) | 3 packages `ok`; 50 top-level tests pass | not run (host unavailable) |
| `make -C tools/gomad3 test-host` | pass (exit 0) | 45 packages `ok`, 1 without test files; 975 top-level tests pass, 21 skip | not run (host unavailable) |
| `make -C tools/gomad3 world-test` | pass (exit 0) | 3 packages `ok` under `-race`; 36 top-level tests pass | not run (host unavailable) |
| `go test -tags test_dep ./tools/gomad3sim/...` (repo root) | pass (exit 0) | 1 package `ok`; 55 top-level tests pass | not run (host unavailable) |
| `make gomad3-integration-test` | pass (exit 0) | 1 package `ok`; 3 top-level tests pass | not run (host unavailable) |

[test-dispositions-darwin-arm64.tsv](test-dispositions-darwin-arm64.tsv) lists gate, package,
top-level test, and result for all 1140 tests. It comes from a second run of the five `go test`
recipes with `-json` added, each of which exited 0. The 21 skips are 19 subprocess helper tests
in `runner` and `runner/internal/execution`, `TestMemberlistSuppliedTCPConsumer`, and
`TestRegenerateMatchesCheckedPatchForPinnedArchive`. The final task compares its own listing
against this file. A test that disappears or changes from pass to skip needs an explanation
under R8 and R9.

The gates left the three directories unchanged. `git diff --stat HEAD` and the untracked listing
for them stayed empty after the runs.

Not run in this task on either platform, and therefore without a baseline disposition here:
the remaining tiers of `make -C tools/gomad3 test` (`test-toolchain`, `intercept-test`,
`overlay-test`, `test-builder`, `test-live-capability`, `test-runtime`, `test-upstream`),
`clock-audit`, `compatibility-pack-qualification`, `core-qualification`,
`make gomad3-smoke-qualification`, `make gomad3-qualification`, and
`make gomad3-tests-qualification`. R9 names the full Gomad gates, the Temporal smoke selection,
and the affected qualification suites for the final task. The final task must take their
baseline dispositions from the fn-105 evidence recorded at this revision or state that no
baseline exists.

## Pre-existing findings and owners

| Finding | State at the baseline | Owner |
| --- | --- | --- |
| D12, linux/amd64 replay divergence. About one tier-3 seed-run in 26 is nondeterministic or diverges on replay. The F5 and F6 suites are `intermittent` on linux and the linux gates accept `nondeterministic` and `replay_divergence` for them. | Open. Not reproducible on this host. | `fn-105-gomad-follow-ups-deferred-scope.12` (todo) |
| D14, darwin/arm64 `TestSignalWorkflowTestSuiteChasm` replay divergence. | Fixed in the baseline revision (toolchain key `8d28bd44…`); the generated manifest expects darwin/arm64 `qualified`. The same code runs on linux/amd64 and was not run there. | `fn-105-gomad-follow-ups-deferred-scope.14` (done) |
| Stale linux pack `modernc-libc-xsys-v047-linux-amd64`. It binds profile digest `sha256:96487435…` while the linux profile golden is `sha256:84b27e62…`. | Open. `make -C tools/gomad3 validate` is expected to fail on linux/amd64 in `TestHostPacksBindCurrentProfile`. Rediscovery needs a linux/amd64 host. | Recorded as open work by `fn-105-gomad-follow-ups-deferred-scope.26` (done); no open task owns it |
| Host-clock escapes and the DTrace clock audit (`.plans/GOMAD_MILESTONES.md`, "Open findings"). | Open, investigated under D21. The audit needs a root run; CI supplies it. | fn-105 (D11, D21) |

fn-108 changes none of these dispositions. A cleanup task that sees D12 on linux records it for
fn-105.12 and does not treat it as new evidence.

## Files in this directory

| Path | Content |
| --- | --- |
| `baseline.md` | this record |
| `size-count.sh` | counting script, rule v2, SHA-256 `9d3afaa0ed1d0579fa2b2947a8525b0b33a0f23e0e0f7e650b1febe6bd87f53d` |
| `size-compare.sh` | R1 comparison of two per-file listings, SHA-256 `79b611cedf16308831c671ac9de77633e98fe4ce59423ce079d226c82e67f8c8` |
| `size-baseline.txt`, `size-baseline-files.txt` | baseline table and per-file listing |
| `api-capture.sh` | public-surface capture script, SHA-256 `697cd934fcc6d65aafbc03944983d9438fd04970c20c75051ec4177d7f22042c` |
| `api-baseline/` | baseline output of `api-capture.sh` |
| `task1-review.md` | review record of this task, four rounds; written after the manifest and not listed in it |
| `SHA256SUMS` | SHA-256 of every other file here except `baseline.md` and `task1-review.md`; its own digest is `a64ac192433c03e4c7813027dd8a9832790a6fd34af0217468231ab6a1a64327` |
| `git-status-baseline.txt` | `git status --short` at the baseline, without this directory |
| `gate-logs/` | one log per gate and `results.txt` with exit codes and UTC times |
| `test-dispositions-darwin-arm64.tsv` | per-test results of the five `go test` gates |
