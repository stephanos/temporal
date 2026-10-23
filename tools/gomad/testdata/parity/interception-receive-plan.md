# Fresh fn233 receiving route

`interception-receive.py` is a separate local receiving producer and independent
raw reader. It does not add an original96 stage, change the frozen fn232 manifest,
or replace the 81-case shim corpus and its two generations. In particular the
five original `auto-*` definitions remain explicit `-exec=true -count=0` cache
probes. This route adds the missing actual auto-derived Frontend success/reuse
proof rather than changing those probes into guest executions.

The original corpus contains exactly 21 real-input definitions: `guest-test`,
`caller-count`, `boundary-args`, `boundary-double-args`, `boundary-terminator`,
`guest-lookalikes`, `goflags-quoted`, `repeated-effective-flags`,
`empty-effective-exec`, `external-launcher`, `exec-true-count0`,
`multi-package-json`, `guest-run`, `guest-build`, `explicit-excluded-tag`,
`auto-prime`, `auto-reuse`, `auto-schema-mutation`, `auto-schema-missing`,
`auto-changed-executable`, and `concurrent-identities`.

No current-candidate Go execution is recorded by this source change. Historical
9fd36aedd8 manual/P2/hidden replay proofs remain historical. The existing six-case
launcher corpus still owns actual target-failure first-stop/replay/minimization,
report provenance refusal, and infrastructure non-replayability. The existing
callback harness owns compiled-Go marker-before-consumer-close evidence. Native
`short_markers_are_forwarded_before_guest_waits_for_controller_release` owns the
controlled WAT handshake; these endpoints do not claim a new controlled Go guest
handshake from a completed transcript or a successful process exit. Coherent
final Make wrapper gates and both complete shim generations remain separate
required executions after the final candidate is frozen.

## Prerequisites

Supply a private, reviewed `gomad.interception-receive-inputs/v1` JSON file with:

- `binary`: absolute frozen ordinary release executable.
- `fixture_binary`: absolute frozen `determinism-fixture` executable from the same
  joined source. The real public negative probes require the ordinary binary to
  reject the fixture grammar and the fixture binary to recognize it.
- `compiler`: absolute pinned stock Go1.27.1 executable, `checkout`: genuine
  accepted Temporal checkout with its actual `gomad.toml`, `cache`: fresh owned
  XDG cache for the two auto routes, and `installation`: genuine fixture-role
  Temporal installation manifest with that fixture executable identity.
- `environment`: reviewed compiler/cache controls such as `GOMAXPROCS`,
  `GOCACHE`, `GOMODCACHE`, `GOPROXY`, and `GOSUMDB`. The pinned compiler and XDG
  cache cannot be overridden through this map. No `GOMAD_*` input is accepted.
- `accepted_selections`: exactly two reviewed entries, respectively package
  `./tests/gomadfunctional`, selector `^TestFrontendSystemInfo$`, and package
  `./common/tasktoken`, selector `^TestWorkerTaskID$`. Each entry includes an
  existing absolute `receipt` and its `receipt_sha256`.
- `files`: absolute-path to SHA256 map of actual existing static inputs, including
  both executables, stock compiler, fixture installation, checkout `go.mod` and
  `gomad.toml`, selection receipts, and the reviewed source/build/role receipt
  closure. This map contains no unknown future report hashes.

The producer never downloads a toolchain or invents an installation. The same
compiler, cache and real source inputs must remain available for independent
outside-namespace rereading. The private raw command capture includes inherited
OS environment bytes; treat it with the same custody as other qualification
captures. There is no automatic retry or cleanup of a failed attempt.

## Commands after the conductor grants the actual Go lease

Use a new private common parent `RECEIVING` outside the source/toolchain/cache
paths that will later be masked. Substitute actual absolute input/root paths:

```sh
python3 tools/gomad/testdata/parity/interception-receive.py --inputs INPUTS --case auto-first --root RECEIVING/auto-first
python3 tools/gomad/testdata/parity/interception-receive.py --inputs INPUTS --case auto-reuse --root RECEIVING/auto-reuse
python3 tools/gomad/testdata/parity/interception-receive.py --inputs INPUTS --case manual --root RECEIVING/manual
python3 tools/gomad/testdata/parity/interception-receive.py --inputs INPUTS --case parallel --root RECEIVING/parallel
```

Each route has one original 600-second external admission shared by its role
probes, preparation, Go invocation and reader verbs. Original guest bounds remain
4GiB, fuel2^50, 300seconds, output8MiB, evidence159383552bytes and seed1; cleanup
has at most5seconds after expiry. Reader verbs do not reset that admission.
Append `--read-only` to any command to authenticate retained raw evidence without
executing a command. `proof.json` is a convenience output and is never authority
for the independent reader.

`auto-first` requires an empty derived-installation cache and runs the ordinary
wrapper without an explicit installation or caller `-exec`/`-overlay`. `auto-reuse`
requires the independently authenticated sibling first execution, exact manifest
bytes and unchanged manifest nanosecond mtimes. Both require an actual named
Frontend PASS, exact retained guest output plus the single Go package summary,
real successful seed1 and the current compiler/source/schema inventories.

`manual` and `parallel` use the existing genuine fixture-driver exporter and
public `run-guest` launcher, not a native guest stand-in. The second route uses
the original two-package `-p=2` collection, shared wrapper/driver/artifact root,
distinct guest modules/argv/launch identities, and the union of children of every
Go OS thread. At least one retained sample must show both authenticated launcher
PIDs and their exact raw argv concurrently. Missing overlap fails; there is no
retry or timing-based substitute. These are explicitly fixture-role manual
proofs, not bare ordinary-release manual execution claims.

All four routes additionally require actual public inspect bound to the retained
record/path/hash, exact retained-only replay, success-artifact minimize refusal3,
and ordinary public qualify refusal3 for an intercepted `Kind=exec` source.
Successful Frontend execution is never counted as a minimized target failure.

## Hidden-input retained replay

For `interception-receive-hidden.py`, add pinned absolute `bwrap` and `python`
regular executables to `files`; resolve symlink executables before pinning. Add
`hidden_directories` (existing, distinct, nonoverlapping source/compiler/runtime/
cache/preparation directories) and `hidden_installations` (existing pinned
manifest files including `installation`). They must cover **every** compiler and
source inventory path and the separately captured Go executable. They must not
cover replay executables, the receiving roots, the input declaration or retained
artifacts. Provision an evidence root on a filesystem suitable for native replay.

```sh
python3 tools/gomad/testdata/parity/interception-receive-hidden.py --inputs INPUTS --owner-root RECEIVING --root HIDDEN
python3 tools/gomad/testdata/parity/interception-receive-hidden.py --inputs INPUTS --owner-root RECEIVING --root HIDDEN --read-only
```

The namespace has read-only root, separate user/PID/network namespaces, empty
mounts over each original input, and only its private inside-proof directory persistently
writable (with private namespace devices). A retained exact standalone Python child checks all masked locations
before running exactly three fixture `replay ARTIFACT` commands, with no explicit
installation selector and one shared600-second deadline. It imports no source
checkout code. Each raw replay must reproduce success with exact choice replay.
This proves retained-only replay with the inputs unavailable and no harness
restoration; it does not claim a forensic audit of every possible guest write.

## Offline gates

```sh
PYTHONDONTWRITEBYTECODE=1 python3 tools/gomad/testdata/parity/interception-receive-test.py
PYTHONDONTWRITEBYTECODE=1 python3 tools/gomad/testdata/parity/interception-shim-test.py
```

These fixtures check authenticator causality, exact output and process identities,
cache byte/mtime reuse, independent inspect binding, raw capture integrity,
pre-spawn expiry and hidden input coverage. They supply zero actual Go proof or
final qualification credit.
