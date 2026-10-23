# Temporal Make acceptance

This is source automation and an execution plan, not a completed Make or shim
acceptance result. The earlier handwritten Go launcher proof, its one real `-p=2`
two-package run and detached replays are separate evidence. They do not prove
this wrapper path. No golden file is checked in from an offline fixture.

The harness accepts only the registered `TestFrontendSystemInfo` selector in
`./tests/gomadfunctional`. The unchanged Make target has three recipes; all three
package variables explicitly select that same package. This is three sequential
executions of one support cell, not additional functional/NDC/XDC coverage.
The repository config has no guest main packages, so the generator and linter
remain unmatched host `go run` calls.

## Inputs and stage grants

Freeze the integrated final executable, its matching installation, pinned stock
Go 1.27.1 compiler and source checkout. Retain exact SHA-256 values, integrated
source identity, build command/status and the installation's original limits.
The harness checks the supplied file hashes before and after each stage and
never builds a replacement executable, installation or module. Put the compiler
at its real regular path, not a symlink. No `.bin/go` may shadow the wrapper shim.
Remove inherited Gomad controls and GOOS/GOARCH/GOFLAGS/GOROOT/GOWORK/CGO overrides.
Use an isolated checkout without an existing `test.log`; the harness refuses to
overwrite an operator log. The actual Make target still cleans its test outputs.

Set concrete absolute inputs and the exact accepted selector before running:

```sh
PROOF_BINARY=/absolute/frozen/gomad
PROOF_BINARY_SHA256=EXACT_BINARY_SHA256
PROOF_COMPILER=/absolute/stock-go1.27.1/bin/go
PROOF_COMPILER_SHA256=EXACT_COMPILER_SHA256
PROOF_INSTALLATION=/absolute/frozen/temporal.json
PROOF_INSTALLATION_SHA256=EXACT_INSTALLATION_SHA256
PROOF_CHECKOUT=/absolute/frozen/temporal
PROOF_ROOT=/absolute/new/private/make-proof
```

Each invocation executes **one** stage, only after the conductor grants its
resource window. Supply the granted external stage deadline; it does not alter
the installation or guest limits. For example, after authorization for expansion:

```sh
python3 "$PROOF_CHECKOUT/tools/gomad/testdata/parity/interception-make-check.py" \
  --binary "$PROOF_BINARY" --binary-sha256 "$PROOF_BINARY_SHA256" \
  --compiler "$PROOF_COMPILER" --compiler-sha256 "$PROOF_COMPILER_SHA256" \
  --installation "$PROOF_INSTALLATION" --installation-sha256 "$PROOF_INSTALLATION_SHA256" \
  --checkout "$PROOF_CHECKOUT" --accepted-selector '^TestFrontendSystemInfo$' \
  --stage expansion --timeout 60 --output "$PROOF_ROOT/first/expansion"
```

Run each remaining stage with its own newly granted timeout and fresh stage
output directory; do not start the next stage after a failure:

| Stage | Actual commands and acceptance |
| --- | --- |
| `expansion` | Wrapped `make -n functional-test` for default race, explicit race-on and race-off. Retain all three expanded test recipes for each command. Make may run its own expansion-time shell queries even with `-n`. |
| `default-race` | Actual unchanged `make functional-test` with no `TEST_RACE_FLAG` override. Retain its real status, refusal output and fresh `test.log`; require no campaign. |
| `race-on` | Same with `TEST_RACE_FLAG=on`. Preserve its real Make status; require refusal and no campaign. |
| `direct-race` | Wrapped `go test ./tests/gomadfunctional -race -run=^TestFrontendSystemInfo$`. Retain direct shim status 2 independently of Make and require no campaign. |
| `host-passthrough` | Separately run `go run ./cmd/tools/protogen -h`, `go run ./cmd/tools/lintcode -h` and `go env GOOS GOARCH CGO_ENABLED GOFLAGS`, once directly and once wrapped. These native help probes compile/run real repository tools; they are not full generation or lint passes. Require equal statuses and output after normalizing only Go's temporary usage-executable name. |
| `positive` | Actual unchanged `make functional-test` with explicit race-off/sql/sqlite, shuffle-off, the three supported package overrides, exact selector, count1 and five-minute Go test timeout. Require Make exit0 and three independently authenticated successful campaigns under one wrapper identity, distinct driver and launch identities, exact original argv, compiler/installation binding and actual `run-guest` driver chain. |

The positive Make variable is deliberately:

```text
TEST_ARGS=-run=^TestFrontendSystemInfo$$ -v -count=1
```

Make consumes `$$` into the single `$` regex anchor before the shell/Go sees it.
A single `$` in this Make variable corrupts the selector and may consume `-v`.
The harness authenticates the final single-dollar argv from retained origins.

## Evidence and two real regenerations

Every command retains argv/cwd, selected environment controls, producer PID, raw
stdout/stderr, numeric status, timeout disposition and stream hashes. Stage
receipts distinguish rejected/incomplete observations from accepted stage data.
The supplied binary/compiler/installation and unchanged Makefile hashes are
recorded independently. Admission plan publications are authenticated through their canonical receipts and
matching empty-journal preparation campaigns (`complete=False`); only storage-backed
launcher campaigns require completion. Unknown publications and mismatched receipts
refuse. Successful campaigns are checked through the existing
campaign checker and sealed-origin authenticator: preparation, every execution,
manifest/record/payload hashes, journal chaining and exact stored-byte accounting
must pass before projection. Guest records must retain the original 4 GiB memory,
2^50 fuel and 300-second execution limits. Raw evidence remains available alongside
`semantic.json`. Only the authenticated ephemeral shim argv0 is replaced by
`$GO_SHIM`; support/configuration/status/original test arguments stay explicit.

Run the entire granted stage sequence a second time under
`$PROOF_ROOT/second/STAGE`, with the **same** immutable executable, compiler and
installation hashes. Do not reuse the first output directory or copy results.
Require every receipt's `accepted` value to be true, then compare each pair:

```sh
diff -u "$PROOF_ROOT/first/expansion/semantic.json" "$PROOF_ROOT/second/expansion/semantic.json"
diff -u "$PROOF_ROOT/first/default-race/semantic.json" "$PROOF_ROOT/second/default-race/semantic.json"
diff -u "$PROOF_ROOT/first/race-on/semantic.json" "$PROOF_ROOT/second/race-on/semantic.json"
diff -u "$PROOF_ROOT/first/direct-race/semantic.json" "$PROOF_ROOT/second/direct-race/semantic.json"
diff -u "$PROOF_ROOT/first/host-passthrough/semantic.json" "$PROOF_ROOT/second/host-passthrough/semantic.json"
diff -u "$PROOF_ROOT/first/positive/semantic.json" "$PROOF_ROOT/second/positive/semantic.json"
```

These are two actual semantic golden regenerations. Only after raw authentication
and byte-identical comparison may their accepted projections become corpus
fixtures. No fixture producer output is a real Go/guest pass. Replay the retained
positive artifacts with the exact final binary using the existing detached-replay
controller and a separately granted resource window. Inspect and preserve raw
compiler/launcher observations and post-Go temporary-file absence. Source-only
checks, a copied module or the prior manual Go proof cannot replace this gate.

Make's `tee` pipeline does not preserve each Go status. The final verifier only
requires some `^ok` in `test.log` and no `^--- FAIL`; a stale passing log can mask
all three refusals. The harness uses a fresh log, never changes `SHELL`, adds
pipefail or overwrites `COMPILED_TEST_ARGS`, and reports Make's observed status.
The direct race probe retains the shim-level status. No campaign plus a refusal
is insufficient on its own to prove no compiler build: join this receipt to the
shim lane's compiler-call fixture, which must prove rejection before build.
Likewise, host-tool help/output comparisons complement rather than replace the
shim lane's byte-exact original-argv/environment fixture.

## Offline source checks

```sh
python3 tools/gomad/testdata/parity/interception-make-test.py
python3 tools/gomad/testdata/parity/interception-make-check.py --help
git diff --check
```

The offline tests copy the real unchanged Makefile to a temporary directory and
supply an explicitly fake Go process. They test selector expansion, three-recipe
routing, race propagation, truthful fresh/stale log behavior and preservation of
Makefile bytes. They launch no Go compiler or guest and generate no goldens.
