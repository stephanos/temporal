# World stdlib normalization probe

World can retain its existing JSON bytes with a small domain-owned `json.Marshal` → `Decoder.UseNumber` → `Encoder.SetEscapeHTML(false)` → trim-one-newline normalization for its current valid typed values. The focused probe found zero byte mismatches in 5,029 comparisons. Select this approach for planning, conditional on preserving rejection of invalid Go strings and all existing decoder and framing checks. The normalization alone is not a drop-in replacement for `canonicaljson.CanonicalJSON` because it silently replaces invalid UTF-8.

## Evidence boundary

The probe used an archive of HEAD `0bff0019baf1753cb04fedf9672ace0b5f00dcf7` on branch `gomad`, extracted with `set -o pipefail; git archive HEAD tools/gomad3 | tar -x -C /tmp/gomad-fn153-world-probe-VgkVTz`. Archive extraction returned 0. The source remains in that scratch directory. The repository's `tools/gomad3` status stayed clean and HEAD stayed unchanged.

The ordinary compiler identified itself as `go version go1.27.1 linux/arm64`; `uname -s -m` returned `Linux aarch64`. This is portable source evidence, with no patched runtime, native qualification, soak, process transport, generator, lint, broad host gate or independent-review claim. Existing native owners and the fn-155 delivery priority remain unchanged. Research routing requested `gpt-6-astra/high`; root supplied the session fallback receipt `jev-unavailable(no_key)`, and this worker did not rerun that decision.

The test command comes from the ordinary `world-test` Go test convention in `tools/gomad3/Makefile:163`, narrowed to one disposable test and without the race option. `tools/gomad3/go.mod:3` pins Go 1.27.1. The actual successful command ran at `2026-10-10T13:26:22Z` and printed its scratch module working directory before testing.

```sh
cd /tmp/gomad-fn153-world-probe-VgkVTz/tools/gomad3 && set -o pipefail && {
  date -u '+%Y-%m-%dT%H:%M:%SZ'
  pwd
  /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go version
  env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED GOENV=off GOFLAGS= GOWORK=off GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=0 /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -count=1 -tags test_dep -run '^TestFN153WorldStdlibProbe$' -timeout=90s -v ./world
} 2>&1 | tee /tmp/gomad-fn153-world-probe-VgkVTz/probe-output-corrected-cwd.txt
```

The command returned 0 in 0.544 seconds; Go reported `ok .../world 0.363s`. A preceding attempt at `13:26:10Z` used the tool's `workdir` argument, which shell startup ignored. Its printed cwd was the repository root and it failed setup with `stat .../temporal/world: directory not found`, exit 1. No tests ran in that attempt. Explicit `cd` corrected the known cause before retry. An earlier relative-path gofmt invocation failed with the same cwd issue, exit 2; absolute-path gofmt succeeded. The failed setup output remains separate in `probe-output.txt`.

## Coverage and result

| Corpus | Count | Assertion |
| --- | ---: | --- |
| Structural encoding cases | 4,860 | Four roots × 135 strings × three numeric boundaries × three collection/pointer shapes compare directly with the unchanged legacy encoder |
| Enum and signed-time cases | 22 | Every terminal, request state, event state, quiescence and cancellation constant, plus -1 and MinInt64 time encoding |
| Valid model snapshots | 49 | Seven text cases at initial, pending, deadlock, queued, delivered, canceled and idle states; exact legacy bytes and successful existing DecodeSnapshot |
| Valid replay plans | 7 | Exact bytes and successful existing DecodeReplayPlan |
| Valid transitions | 70 | All four bodies from actual model operations; exact line bytes and exact complete JSONL streams |
| Recording payload comparisons | 21 | Three payloads per frame compare directly with legacy |
| Complete recording frames | 7 | Header, big-endian lengths and payload bytes match EncodeRecording; existing DecodeRecording succeeds |
| Invalid Go UTF-8 controls | 7 | Legacy rejects; candidate accepts and normalizes; four structural roots plus targeted plan, transition and terminal |
| Decode boundary controls | 7 | Unknown, trailing, duplicate, lone surrogate, escaped Unicode, field casing and raw invalid UTF-8 |

The equality total is 5,029; frame checks, invalid-input controls and decoder controls are additional assertions. The length-delimited digest of comparison names and equal payloads is `90c7d75020484036034a94e74aabdfbd65aeff7e60c3ef83c55876019bf48730`. No performance inference follows from this test duration.

The 135 string cases include each ASCII code point independently, empty text, `<>&`, quote/backslash/control sequences, non-ASCII BMP and supplementary characters, U+2028/U+2029, and literal backslash-u text. Numeric cases cover zero, one or InitialTime, MaxUint16 priority, MaxUint32 schema/string bound, MaxUint64 limits/counters/IDs/seed and MaxInt64 time. Slices cover nil, allocated empty and populated; payloads include all 256 byte values. The first pass preserves omitempty, decimal-string custom methods and base64 encoding. Snapshot slices distinguish null from empty arrays; optional empty payloads omit in both cases. All comparisons reject a final newline.

Structural cases deliberately include invalid semantic combinations such as every transition pointer present, oversized schema values and arbitrary enum strings. They establish encoding behavior for each field shape, not validator admission. The valid lifecycle corpus supplies separate accepted-domain examples. The finite corpus does not enumerate every possible string, collection length, numeric value or cross-product of independent fields.

The probe logs every one of the 76 serialized fields. The complete inventory is retained under `FIELDS` in the raw output and derives from these declarations, with paths relative to `tools/gomad3`.

| Source | Covered types and fields |
| --- | --- |
| world/types.go:24 | Config seed/limits; all six Limits fields; ResourceID adapter/kind/key; Request kind/resource/priority/payload; Readiness request/time/kind/payload/equivalence; Cancellation IDs/status; Delivery IDs/time/kind/payload; Quiescence kind/before/after/deliveries/blocked; ReplayProgress cursor/expected |
| world/types.go:101 | Seed, RequestID, EventID and Sequence decimal-string marshalers; LogicalTime decimal-string marshaler |
| world/snapshot.go:11 | RequestSnapshot ID/request/state/optional event; EventSnapshot ID/readiness/state; all 14 Snapshot fields, including nested config, replay, transitions and both digest strings |
| world/replay.go:16 | Eight Transition fields, all four body types and every body field; four ReplayPlan fields |
| world/recording.go:40 | Terminal kind/optional detail; Recording's three payloads and its binary framing at :200 |
| world/world.go:13 | All four request-state and three event-state constants |

This closed domain contains structs, pointers, slices, bytes, strings/aliases and fixed-width integers. It contains no serialized float, map, interface, RawMessage or arbitrary user marshaler. The test population fails on an uncovered field kind. The temporary helper takes `any` for test convenience; this is not a proposal for a new shared canonicalization API.

## Required validation and decoder preservation

`world/world.go:308` validates request kind and resource key with `validateString`, validates resource adapter/kind against the ASCII pattern at :70, and enforces bounds. `validateReadiness` at :324 checks kind and equivalence class with the same UTF-8-aware helper at :337. `EncodeSnapshot` calls Restore before encoding; Restore reconstructs transitions through model operations at `snapshot.go:164`, checks digests and validates payload bounds. These checks must remain in their current order.

There are concrete gaps that the generic encoder currently closes. `validateTerminal` at `recording.go:144` accepts nonempty invalid UTF-8 detail. `validateTransitionShape` at `replay.go:111` and `validateReplayPlanIdentity` at :99 validate shape and hashes without validating nested request/readiness/delivery text. A transition built with an invalid resource key and a correctly recomputed digest passes shape validation; its plan passes identity validation. Existing EncodeTransitions, EncodeReplayPlan and EncodeRecording still reject these inputs through canonicaljson's original-string walk. The probe asserts each of these facts. A migration needs explicit typed World string validation before lossy Marshal, including all nested transition text and terminal detail, while retaining error precedence. Reusing the model's limits-dependent request validator blindly would change the existing standalone replay-plan admission contract.

The proposed helper only decodes JSON that it just marshaled. It cannot replace input validation. Keep Snapshot's config-first allocation preflight (`codec.go:70`), UTF-8 checks, duplicate-key token pass (`codec.go:276`), unknown-field rejection, EOF requirement, exact-byte reencode check, semantic validation and size limits. Keep the corresponding ReplayPlan guards at `replay.go:67`, transition final-newline framing and Recording's magic/length checks and terminal reencode check.

The seven decoder controls show that canonicaljson.StrictDecode rejects unknown fields, trailing values, duplicate keys and raw invalid UTF-8, but accepts a lone escaped surrogate, alternate Unicode escaping and case-insensitive field spelling. DecodeCanonicalJSON rejects all seven through its additional byte-equality check. Plain stdlib Unmarshal accepts every listed case except trailing input. Therefore this probe does not claim StrictDecode explicitly rejects lone surrogates, nor that normalization supplies a strict parser.

## Why the valid-domain result generalizes

The first standard encoding applies the same World field tags and custom decimal marshalers. Its HTML escapes decode back to the original valid strings. UseNumber preserves integer tokens beyond 2^53 without float conversion. All numeric tokens this closed domain emits are integral and within the legacy encoder's signed/unsigned limits. Standard map encoding sorts every decoded object's string keys, matching legacy `sort.Strings`; slices retain order. Both legacy appendJSONString and the proposed final encoder use SetEscapeHTML(false), including their common U+2028/U+2029 handling. Both remove precisely the encoder's newline. These source properties support the domain-wide planning choice alongside the finite probe; they are not a new arbitrary-JSON equivalence claim.

Pinned source references are `encoding/json/encode.go:82,185`, `stream.go:37,259`, and `decode.go:45,51` under `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/src`. Legacy behavior is `internal/canonicaljson/canonical.go:16,263,323`. Keep this reasoning tied to the pin and current type inventory; future floats, arbitrary marshalers or interface fields require a fresh decision.

## Retained inputs

The bounded companion `fn153-world-probe-evidence-20261010.json` binds baseline, source hashes, compiler hash, successful and failed raw output, exact command and corpus counts. Primary scratch source is `tools/gomad3/world/fn153_probe_test.go`, SHA-256 `0929d68d64fb049ae39ca36da48049c8f416c7f70dd119cf9fc2dc6877eb473e`. No product code, task lifecycle, Git commit, CI, PR or external message was changed by this probe.
