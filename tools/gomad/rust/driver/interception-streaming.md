# Controlled compiled-Go output receiving

This repository gate closes the marker → controller action → guest exit contract
using the existing synchronous `OutputConsumer` API. It changes no product verb,
Go fixture, original 81-case shim declaration or its 21 real-input definitions.
The native WAT control test and the public compiled-Go closed-consumer harness
remain separate required checks.

Build `main_cli_build.py --role interception-harness` using the same immutable
source, static plan and GNU envelope as the ordinary main build. This closed role
selects only `--example interception-streaming-replay`, with empty host/helper
features. Its actual Cargo artifact, raw commands, build scripts and publication
use the existing independent typed build reader. The optional frozen
`interception_streaming` binding is `{binary, build_receipt}`.

After the conductor releases the Go/CPU execution window, use genuine completed
build receipts and an ordinary cooperative installation with the original 4 GiB,
2^50-fuel limits:

```sh
python3 tools/gomad/rust/driver/interception_streaming.py \
  --main-build /absolute/main/main-build.json \
  --harness-build /absolute/streaming/main-build.json \
  --installation /absolute/ordinary-installation.json \
  --root /absolute/fresh-streaming-proof
python3 tools/gomad/rust/driver/interception_streaming.py \
  --root /absolute/fresh-streaming-proof --read-only
```

The producer first runs one actual ordinary `explore` of the unchanged
`./cmd/gomad/testdata/seedfree -- stream` fixture, seed 1, with the original
`test_dep` tag plus the installation’s unchanged tags. It preserves the original
five-minute execution and ten-minute overall bounds, 8 MiB output and 8 MiB
choices. The same 600-second admission covers source/build reopening,
preparation, baseline execution, controlled replay and receiving checks. Actual
native preparation observations authenticate the compiler, original Go source,
recipe, selected module and producer PID. Missing prerequisites refuse; no
module, receipt or successful observation is manufactured.

The harness opens that successful artifact through production replay preflight.
The actual stdout callback accumulates a bounded forwarded stream, delivers the
registered marker to the controller and blocks on a bounded release channel.
The controller receives it, checks execution has not completed, synchronously
writes and fsyncs the release witness, then releases the callback. The guest can
continue only when its actual `fd_write` callback returns. A normal exit and
byte-exact forwarded/retained stdout and stderr are required, alongside exact
WASI and choice replay. No guest stdin or new host capability is introduced.

The independent reader reopens both typed builds, actual process journals,
complete baseline campaign/artifact, raw preparation observations, marker,
release witness and replay streams. It checks shared source/compiled identities,
actual child PID, kernel monotonic ordering and original resource bounds.
`proof.json` is a convenience output and is never authoritative. `--read-only`
executes no compiler, harness, CLI or guest.

Focused Python tests use synthetic protocol counterexamples and a tiny ordinary
Python process. They establish no actual Go execution or fn233 acceptance.
