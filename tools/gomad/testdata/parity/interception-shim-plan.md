# Public shim and wrapper acceptance

This is source automation. No final CLI, Go compiler, guest, auto-installation or
golden regeneration has run through this harness. The 81 declared cases cover
fn233.2/.3; the existing interception launcher and Make harnesses remain their
own owners. Do not replace their actual runtime proofs with these fixtures.

Run `interception-shim-check.py --help` for the executable interface. It runs
exactly one named case per invocation, into a new private output directory, with
an explicitly supplied final executable SHA-256 and conductor-granted external
deadline. It creates no installation or Go source fixture. A failed case retains
its raw outputs and a rejected receipt and is not retried automatically.

## Synthetic compiler cases

The `host`, `wrapper`, `config` and generic `refusal` cases use the **actual final
Gomad executable**, with an explicitly fake compiler on the original PATH in a
new minimal module. Auto mode permits host routing and pre-build refusal without
manufacturing an installation or claiming stock-Go admission. The fake compiler
is a generated executable Python fixture, not an accepted toolchain. Its captures
are marked `synthetic_compiler: true` and `installed_admission: false`.

A host intermediary records its complete inherited environment as hex-encoded
OS bytes, then invokes bare `go`. The fake compiler externally records all
argument/environment bytes and its actual executable path. The checker requires
exact equality with the intermediary's argv/environment, including empty values,
spaces, shell metacharacters and invalid UTF-8. No shell reparses these values.
`go env GOOS` returns the fixture's actual host platform. All non-guest subcommands
and unmatched `run`/`build` have separate cases; generator/linter repository help
probes remain in the Make lane and are not duplicated here.

Each generic guest refusal requires status2, empty stdout, one stderr line naming
the refused option, **zero compiler captures**, and zero campaigns. Zero captures
includes metadata calls such as `go list` or `go env`, rather than just absence of
a build command. Both argv and GOFLAGS channels cover race, MSan, ASan, coverage,
coverage profile/package and fuzzing. CGO_ENABLED=1 and incompatible GOOS/GOARCH
are separate cases. Refusal tests never accept a fake installed admission.

Wrapper cases exercise true/false statuses, a child literally named `replay`,
child arguments resembling Gomad flags, empty/missing commands, missing `--`,
private-directory failure and direct `go` resolution through the shim. Config
cases cover unknown and duplicate keys, malformed syntax, missing/wrong-type
runtime root, absolute/escaping/symlink runtime paths, wrong derive type/value,
and wrong-type/absolute/escaping guest entries.

For example, only after the final executable is frozen and the conductor grants
the bounded synthetic case:

```sh
python3 tools/gomad/testdata/parity/interception-shim-check.py \
  --binary /absolute/frozen/gomad --binary-sha256 EXACT_SHA256 \
  --case refuse-race-argv --timeout 30 \
  --output /absolute/private/shim-first/refuse-race-argv
```

The fake compiler does not execute any Go subcommand. The offline unit suite
below does not invoke Gomad at all; it validates the byte fixture and authenticators.

## Real guest and cache cases

Every `real` case additionally requires `--real-inputs FILE`. Inputs must be
reviewed for the exact granted operation before execution. A missing accepted
selection, installation, compiler or baseline blocks the case; never fill a gap
with synthetic success bytes. Use existing registered packages/main programs
only. No new Go fixture source, toolchain download, flag weakening or expanded
suite support follows from this plan.

The real input document has this shape (replace every placeholder):

```json
{
  "schema": "gomad.interception-shim-real/v1",
  "case": "guest-test",
  "binary_sha256": "EXACT_FINAL_BINARY_SHA256",
  "compiler": "/absolute/pinned/go1.27.1/bin/go",
  "compiler_sha256": "EXACT_COMPILER_SHA256",
  "checkout": "/absolute/accepted/checkout",
  "installation": "/absolute/genuine/temporal.json",
  "installation_sha256": "EXACT_INSTALLATION_FILE_SHA256",
  "installation_bindings": [
    {"import_path": "go.temporal.io/server/tests/gomadfunctional", "path": "/absolute/genuine/temporal.json", "sha256": "EXACT_INSTALLATION_FILE_SHA256"}
  ],
  "cache": "/absolute/private/shim-first/cache",
  "argv": ["test", "./tests/gomadfunctional", "-run=^TestFrontendSystemInfo$", "-v", "-count=1"],
  "status": 0,
  "campaigns": 1,
  "outcome_classes": ["success"],
  "limits": {"memory_bytes": 4294967296, "fuel": 1125899906842624, "wall_millis": 300000},
  "environment": {},
  "accepted_selections": [
    {"package": "./tests/gomadfunctional", "selector": "^TestFrontendSystemInfo$", "receipt": "/absolute/accepted/selection-receipt.json", "receipt_sha256": "EXACT_ACCEPTED_RECEIPT_SHA256"}
  ]
}
```

The accepted-selection receipt is an explicit conductor input, verified by its
exact file hash; it is not inferred from a successful compile. Each concrete
package and exact selector must occur in the actual command. Real source,
compiler and installation paths stay outside disposable output directories.
The input file is retained verbatim. Commands retain complete argv/environment
bytes, producer PID, raw stdout/stderr, status and stream hashes. Evidence roots
are private because complete environments may contain private operator values.

Plan discovery distinguishes storage-backed launcher campaigns, unfinished zero-guest
preparation campaigns, and published admission plans. Preparations require an empty
journal and are authenticated with `complete=False`; launchers must complete. Each
published admission needs its canonical receipt, exact copied plan, record/module/
provenance commitments and a matching preparation campaign. Publications are never
opened as independent campaign directories or silently skipped.

Build publications under `build-launchers/INVOCATION/ID/plan.json` use the same
receipt authentication plus the exact `inputs.build_launcher` marker contract.
The checker binds module digest, selected raw installation/catalog membership,
original invocation/recipe commitments, actual original Go PID/compiler/driver
registration, namespace invocation ID and lossless original receipt command.
Unknown namespaces or marker/mapping changes refuse. A later launched build
artifact has its own renewed origin and matching preparation/admission pair;
the original build origin is not substituted for that new runtime origin.

Every real execution authenticates preparation, campaign plan, hash-chained
journal, record/manifests, retained payloads, sealed origin/provenance and exact
stored-byte accounting before projection. Original argv must match the actual
case declaration, and the rewritten driver's effective launcher/count are checked
before the guest-argument boundary. Each retained record must match the supplied
accepted limits, bounded by the original 4GiB/2^50/300-second ceilings. Nothing
raises those limits. Shared-driver installation mappings must equal the complete
ordered supplied import-path/manifest-digest list, without duplicates or unknown
members, and their canonical digest must equal the origin commitment. The selected
provenance retains that mapping after wrapper-private files disappear.

| Real case | Required proof shape |
| --- | --- |
| `guest-test` | Registered package/selector, one actual successful campaign. |
| `caller-count` | Explicit `-count=2` preserved effectively; still one launcher campaign. |
| `boundary-args`, `boundary-double-args`, `boundary-terminator` | Respectively `-args`, `--args`, `--`; explicit accepted test arguments remain on their proper side and the effective Gomad launcher precedes the boundary. |
| `guest-lookalikes` | `-args -tags=guest-only -exec=guest-only -count=guest-only` must reach the guest. The test binary's unsupported-flag failure is an actual target failure with driver status1, not a shim build-option refusal. |
| `goflags-quoted` | A reviewed effective GOFLAGS value with actual quoting, for example `-p=1 "-tags=test_dep,disable_grpc_modules"`, plus a successful registered execution. |
| `repeated-effective-flags` | Include `-race=true -race=false`; preserve the effective disabled policy and actual success. |
| `empty-effective-exec` | Include `-exec=false -exec=`; the effective empty caller exec must allow Gomad injection and actual execution. |
| `external-launcher` | Use `-exec=$EXTERNAL_LAUNCHER` and count1. Go must invoke the generated nontrivial Python launcher on its real compiled module. The launcher copies and authenticates its inherited invocation/recipe bytes and records module hash/size before Go removes it. Require no Gomad execution campaign; external success is not a guest pass. |
| `exec-true-count0` | Literal `-exec=true -count=0`, successful pinned-Go compilation, no actual guest/campaign. |
| `multi-package-json` | One Go invocation with two already accepted concrete packages, `-p=2 -v -json`, exact accepted `-run` selector and two successful retained campaigns. Require two Go JSON package-pass events and selected test-pass events, one wrapper/driver identity and distinct launch identities. Use the previously accepted FrontendSystemInfo/WorkerTaskID pair only after granting this wrapper-specific run. |
| `guest-run` | Existing accepted main package explicitly listed in that scratch checkout's `guest`; actual successful launcher evidence. |
| `guest-build` | Existing accepted registered main, `build -o $BUILD_OUTPUT`; require executable launcher and actual `.wasm` bytes. Invoke the published launcher after the wrapper exits, retaining a successful campaign. This stage includes that separate actual guest and needs its resource grant. |
| `explicit-excluded-tag` | A genuinely admitted explicit manifest with a stricter `excluded_build_tags` policy and a caller requesting one of those tags. Require status2/no campaign. The fixture must retain all existing compiler/source/configuration admission; do not manufacture an installed fake compiler. |
| `concurrent-identities` | Supply two distinct `commands` arrays of accepted `go test` invocations. A barrier releases both under one wrapper. Require both actual driver statuses0, two successful campaigns, one wrapper identity, two distinct driver identities and two distinct launch identities. Prewarm the admitted inputs sequentially under separate grants; no concurrent heavy preparation is authorized by this plan. |

The external case additionally requires `launcher_arguments`: the exact ordered
Go-translated arguments expected after the temporary module operand, established
from the pinned driver contract and approved with the input declaration. The
capture must contain exactly one invocation. Its canonical original caller argv
(including `go` and every original flag in order), rewritten driver argv, effective
external launcher, compiler bytes/path, ordered installation mapping and actual
manifest bytes must match the supplied inputs. Canonical invocation and recipe
bytes are retained with their digests and inherited wrapper/digest bindings. The
fixture hashes compiler/source inventories while private generated inputs still
exist, then the checker compares those observations to all recipe/input bindings.
Wrong metadata cannot pass merely because an external process ran and no Gomad
campaign exists. These observations remain caller-owned compilation evidence,
never a Gomad guest execution/admission claim.

The external-launcher path and build output placeholders are expanded directly as
OS arguments; no shell interpretation is introduced. `commands` is used only by
the concurrent case and must contain exactly two different test argv lists.

The generic fake-compiler refusals supply raw zero-call proof. The explicit
excluded-policy case cannot replace the genuine pinned compiler with a fake:
join its receipt to the shim owner's compiler-call-counter test or a separately
provided process audit proving the policy rejects before metadata/build. No
campaign on its own is not sufficient no-build proof. The current host has no
`strace`; this harness does not invent an audit result.

## Auto-cache stages

Cache probes remain explicit `go test -exec=true -count=0` compile-only operations
using pinned real Go. Omit the explicit installation. Use one private cache per
regeneration and a conductor-owned scratch checkout whose required schema/config
inputs can be staged independently of the main source. The harness never edits
or removes source/schema files itself. It snapshots them before/after and refuses
a concurrent mutation during its stage. Required source mutations are separate
conductor actions with preserved original bytes and receipts.

1. `auto-prime`: derive through the actual wrapper, then independently authenticate
   every manifest, complete compiler/runtime inventory, linked executable binding,
   configuration preimage and captured schema payloads.
2. `auto-reuse`: add `baseline` and `baseline_sha256` naming the first stage's exact
   `real-authenticated.json`. Require unchanged manifest bytes **and mtime**, with
   the same current schema/executable binding.
3. `auto-schema-mutation`: stage one authorized schema-byte change in the scratch
   checkout. The baseline remains the prime receipt. Require changed schema bytes,
   an additional authenticated cache entry, and its captured payload inventory
   exactly equal to current source schema bytes.
4. `auto-schema-missing`: stage unavailable schema in the scratch checkout while
   preserving the earlier cache. Require input status2, no guest, unchanged cached
   entries and no fallback to retained schema bytes. Compare against the matching
   pre-removal baseline receipt; retain/restore source bytes under conductor control.
5. `auto-changed-executable`: supply a separately reviewed genuine changed executable
   and matching `binary_sha256`; also supply `previous_binary` for authenticating
   old entries and the baseline receipt. Require different executable bytes and
   a new matching entry binding both current executable and current schema.
   No replacement binary is built or byte-patched by this harness.

`real-authenticated.json` retains exact cache-file hashes, configuration/executable
bindings and captured source-schema inventories. The checker does not normalize
away a changed configuration or authenticate an old entry against the wrong
executable. The second regeneration uses its own empty private cache, repeats the
same controlled source-state sequence and preserves the first generation's cache.

## Two genuine golden regenerations

Run each of the 81 named cases once into `FIRST/CASE`, then independently again
into `SECOND/CASE`, always using the same immutable candidate for that case and
fresh outputs. Every actual compiler or guest stage needs its own resource grant.
A missing accepted main-package fixture or changed-candidate input leaves that
case pending and prevents golden publication; do not skip the case or substitute
an earlier source candidate. Cache probes share only their own generation's
private cache and immutable baseline receipts.

Only after all cases and their raw-authentication receipts pass:

```sh
python3 tools/gomad/testdata/parity/interception-shim-check.py \
  --compare /absolute/private/shim-first /absolute/private/shim-second \
  /absolute/private/new-shim-golden.json
```

Comparison requires the complete declaration, accepted receipts, every retained
raw file hash, cache-entry hashes, semantic hashes and **byte-identical** paired
semantic files. It writes a new golden only after those checks. Dynamic PIDs,
wrapper/driver/launch IDs and raw stored-byte widths remain authenticated in raw
evidence; projections retain their count/equality/distinctness relationships.
Compiler/binary/configuration identities, semantic argument bytes, statuses and
Go JSON action/test/package results remain explicit. Unit tests never produce
this file. This lane has no Go-host reference implementation and claims no
qualification or broader test support.

## Offline source validation

```sh
python3 tools/gomad/testdata/parity/interception-shim-test.py
python3 tools/gomad/testdata/parity/interception-make-test.py
python3 tools/gomad/testdata/parity/interception-check-test.py
python3 tools/gomad/testdata/parity/interception-shim-check.py --help
git diff --check
```

These checks launch only Python fake processes and Make's already-owned offline
fixture; no Go/Cargo/final executable or actual guest is executed.
