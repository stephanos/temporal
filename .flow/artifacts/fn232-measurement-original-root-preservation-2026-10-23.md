# Measurement preservation continuation

This batch implements the remaining source gaps identified by the source-51 audit
(`/home/agent/evidence/fn232-test-wasm-reach/measurement-remaining-native-audit.json`).
It does not qualify all 119 original roots or the 25 named producers by suite count.
The original frozen qualification and fn185 wall/CPU runs remain separate gates.

## Behavioral changes and evidence

The continuation preserves the original hardlink publication refusal (G01), Linux
8 KiB witness and 14 malformed memory cases plus both-owner dispatch admission and
Darwin warm admission (G02–03), positive migration baseline costs and raw custody
(G04), namespace success witnesses and warm metrics (G05), terminal failure
collection (G06), admitted historical warm identities (G07), actual process death,
publication lock wait and failed reclaim recovery (G10), public request-local
cancellation (G11), ordered distinct collection failures (G12), typed native owner
projections and argument shapes (G13), published command reruns (G14), cache
configuration/OS/architecture checks (G16), guest sampling metadata and bounded
samples (G19), three-seed recommendation folding (G20), and selected suite and
Workflow subcase witnesses (G21). Original Go JSON wire encodings remain distinct
from typed native execution; the native tests do not authorize a Go transcript
importer or substitute for the original transport-only tests.

The received cluster batch owns pair availability, checked aggregates, work/wait
attribution, launch/join branches and actual internal phase guards (G08–09, G15,
G18). The received artifact batch retains canonical evidence even when execution
fails (G17). The stress owner supplies the distinct original stress catalogue
adapter. Their source and test receipts must remain independently attributable.

The new exact request path separates original compiler tuples from execution:

- Full Temporal compilation uses three original tags and no selected test args;
  derived compilation retains its original tag order. Only host-owner retains
  its explicit four-tag slim exception.
- Runtime execution preserves `guest`, `literal $arg`, the exact three ordered
  environment entries, seed 7 in the model, 64 MiB choice/transcript limits,
  256 MiB memory, 2^40 fuel, and 180 seconds. No campaign environment entries are
  appended to the direct original request.
- Temporal requests preserve `temporal.test`, the original test arguments and
  five ordered environment entries, original captured configuration, 1 GiB
  choices/transcript, digest-only transcript mode, 4 GiB memory and 2^50 fuel.
- Terminal module preparation uses the original integration configuration,
  `./tests`, three tags, no target arguments, 4 GiB memory and 2^50 fuel. It does
  not capture the later Temporal execution schema. Its explicitly separate native
  preparation normalization enables choices with a 1 GiB limit, 30-minute timeout,
  1 MiB output, one worker and the existing 64 MiB transition bound.

`--prepare-terminal-module SPEC ROOT` publishes the closed
`gomad.native-terminal-module/v1` descriptor, authenticating actual module,
provenance, installation and preparation sidecars. The terminal consumer loads
that descriptor instead of recompiling a Workflow approximation.
`--capture-host-evidence ROOT` publishes actual `host.json` and bounded raw host
witnesses through the existing collector. Nested process observations retain
actual producer PID, explicit `CLOCK_MONOTONIC`, start before spawn and reap time;
no start timestamp is derived from elapsed time.

Raw-mode canonical WASI remains a byte-identical transcript alias. Digest-only
mode publishes a null file with the actual rolling transcript length and SHA from
the finalized outcome, authenticated by request/outcome/native-catalogue hashes.
It creates no fake empty WASI payload. The artifact reader rejects ten causal
mutations, including coherent metadata rewrites and contradictory raw files.

## Actual gates

All commands use the task-owned target, `CARGO_INCREMENTAL=0`, one build job, and
serial tests. Logs are retained below `/home/agent/`.

- `measurement-exact-recipes-focused-v5.log`: 175 passed, 0 failed, 1 ignored;
  includes actual digest-only WAT mutation tests, real process/retention controls,
  actual host endpoint collection and child ancestry/monotonic enclosure.
- `measurement-exact-recipes-clippy-v6.log`: all-target native Clippy, warnings
  denied, exit 0 on that exact source.
- `measurement-exact-recipes-public-corpus.log`: 7 passed, exit 0, including actual
  raw/digest WAT comparison and public native retention. This predates the last
  request-environment and process-observation edits; rerun before final acceptance.
- Original module/recipe tests: 15 passed, exit 0 before the final added fixtures;
  the current focused gate includes their complete matrix.

The ignored actual compiler/preparation proof remains explicit. New endpoint
compiler execution, current admitted Release source receipts, a fresh original
Go/helper/native runtime tuple, full Temporal named executions and frozen final
qualification are not claimed by these unit/native fixture gates. Earlier source-51
three-leg development evidence used a supplied modified request (`gomad-target`
and campaign environment additions); it proves supplied-request transport
consistency only and is not original runtime-cost qualification.

## Independent review repairs

The fresh review of `2e8cee0c34` found two real binding gaps. The loader now checks
actual provenance kind/source/directory/tags/arguments, the authenticated original
installation tuple, and both normalized native configuration commitments against
the existing preparation configuration builder. Twelve mutations reject relabeled
slim builds, selected compile arguments, another source/checkout and changed
configuration, fuel or memory. Both producer and consumer perform this binding.
Warm and prime now load the same original terminal module descriptor and construct
the separate execution request afterwards; they no longer recompile a Workflow
approximation and compare raw provenance.

The positive fixture exposed the ordinary parser's 64 MiB cap. A private
measurement parser shares its grammar with the campaign parser while restoring the
original 1 GiB measurement cap. Ordinary campaign and strategy parsers retain their
64 MiB bound, with an explicit boundary regression for both entry points.

- `measurement-terminal-binding-review-tests-v2.log`: 18 passed, exit 0.
- `measurement-terminal-binding-focused.log`: 178 passed, 1 ignored, exit 0.
- `measurement-terminal-binding-review-clippy-v3.log`: all-target Clippy with
  warnings denied, exit 0.
- `measurement-exact-recipes-fixture-controls.log`: actual fixture role warm
  controls, 16 passed, 1 child-only ignored, exit 0 (before these binding repairs).
- `measurement-exact-recipes-public-corpus-v2.log`: 7 passed, exit 0 on `2e8cee0c34`.

The optional broad unoptimized 656-test run was interrupted during unrelated cache
fixtures repeatedly hashing the approximately 400 MB test executable. Its Cargo
exit is 101 after SIGTERM, and it receives no passing credit. The explicit record
is `/home/agent/measurement-terminal-binding-full-native-interruption.json`.
The root integrator owns the final optimized broad suite on the combined source.
