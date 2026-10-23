# Immutable Go reference invocation glue (2026-10-23)

Scope: fn-185 R4/fn-232 original-reference preservation. This is an invocation
fixture and development ABI evidence, not final runtime-cost/workload qualification.
Source worktree `/home/agent/wt/fn232-profile-reference`, base `4fcf9fa06b`;
accepted terminal-control worktree is untouched.

`driver/reference_profile.py derive` exports immutable commit
`1d695061b6382009d0659238c652a5a765b1d5f5` to an external private copy. Its only
source addition is `tools/gomad/cmd/gomad-reference/main.go`, copied from the
reviewed `driver/reference/main.go.txt`. Every original regular file and symlink,
the Git archive, and the sole addition are committed by `source.json`; verify
refuses mutation or undeclared files. Frozen source is never edited.

The added command uses existing original `backend.InspectModule`,
`wasi.Request.SetRemoteModel` and `wasi.Run`, then emits the full exported original
Result plus original terminal/helper-stderr accessors. It adds no host/model
algorithm or replacement observer. Source anchors in the immutable tree:
`wasi/process.go:274`, `wasi/measurement.go:19`,
`backend/simulation_inventory.go:5`. Original runtime predicate is
`internal/measure/producers.go:310`; runtime-cost owner loop is at 2420–2508.

The wrapper accepts only an uncached standalone record request with writable
empty directories, no captured files/storage/network/replay/inspection and a
frozen clock. It binds the native input exactly to actual prepared provenance,
allowing only the existing seed/environment derivation, and passes the actual
prepared choice identity verbatim. It preserves fuel, memory, timeout, output,
WASI/choice/call and filesystem bounds. Old Go has no aggregate evidence cap;
therefore input is refused unless that cap is provably nonbinding: output +
transcript + choice + optional diagnostic choice + 64 MiB inspection metadata.
The last term covers automatic inspection metadata even without an inspection
request (`model/inspection.rs MAX_METADATA`, `engine.rs` terminal serialization).
No accepted bound is increased. Unsupported shapes are refused, not projected.

Commands (absolute paths; each output must be fresh):

```sh
python3 reference_profile.py derive --repo ROOT --output DERIVED
python3 reference_profile.py build --derived DERIVED --stock-go PINNED_GO \
  --files BUILD_FILES --output BUILD
python3 reference_profile.py run --derived DERIVED \
  --native-request ACTUAL_REQUEST --request-sha256 REQUEST_SHA \
  --provenance ACTUAL_PROVENANCE --provenance-sha256 PROVENANCE_SHA \
  --files RUN_FILES --build-receipt BUILD/build.json \
  --reference-binary BUILD/gomad-reference --helper FRAMED_HELPER \
  --helper-receipt HELPER_BUILD_RECEIPT --output PAIR
```

`BUILD_FILES` maps absolute source-manifest/stock-Go paths to hashes. `RUN_FILES`
also binds build receipt, harness binary, helper binary and helper build receipt.
Dynamic request/module/provenance are authenticated by the enclosing actual
producer and explicit hashes, not falsely listed as static pre-frozen outputs.
The pair manifest (`gomad.runtime-reference-pair/v1`) binds all of these inputs,
Go/helper run JSON paths/hashes and actual command/PID/resource/session receipts.
Each owner retains exact `raw-result.json`, six byte streams and normalized
`run.json`; original WASI comparison uses the retained digest even when raw
transcript is omitted. Final reader independently reads the raw result.

The standalone helper must have **only** `reference-helper`. `conformance`
selects the model driver (`wasmhost/src/lib.rs:84`), not the engine's framed
transport (`lib.rs:91`). Both-feature development attempt and its protocol-type
refusal are retained; the corrected build has an actual Cargo artifact receipt
listing exactly `['reference-helper']`. No stress/diagnostic helper is relabeled.

Evidence root `/home/agent/evidence/fn232-profile-reference/` retains original
source copies/manifests, exact added-Go diff, two real Go build receipts, helper
source/compiler/feature/process receipts, all unit and failed probe exits.
Seven source/custody tests pass, including original evidence-predicate mutations,
CLI build dispatch, exact request binding and accepted-control refusals.
Fresh independent same-family gpt-6.1-sol/high reviewer resolved four blockers
(CLI shadowing, clock mismatch, fabricated choice identity, ignored aggregate
cap) and returned no remaining source blockers; no runtime qualification follows
from this review.

Actual `development-pair-v1` refused before execution: both helper features
selected the wrong protocol. Actual Go-owned `development-pair-v2` reached the
retained draw-v6 fixture: exit 1, reaped, 146 WASI calls, nonempty Go filesystem
service counts and 27 select polls. This fixture intentionally exits 1; the
runtime wrapper correctly refuses it as a successful runtime-cost execution.
Its subsequent helper-owner development comparison is separately retained,
without changing that refusal or claiming the original positive runtime gate.
The original runtime-cost module, successful zero exit, 100000-loop witness,
three-way Go/helper/native ordinary comparison and final frozen gates remain
root-owned pending actual execution.

Development helper-owner comparison completed, exit 0:
`development-pair-v2/development-comparison.json`. Both runs intentionally exit
1 and reap successfully; all six raw stream hashes and the original WASI digest,
outcome and exit fields match. Go owner PID 521076 serves 146 calls with Go
filesystem counts (including fd_write); helper owner PID 533861 serves the same
146 calls with an empty Go filesystem count map. Both report 27 select polls;
actual authorization-to-terminal intervals are 12,174,417 and 8,078,334 ns.
Actual Started reports Wasmtime 47.0.3 and pinned original configuration;
ChoiceTrace.Version is 4 and terminal length 16. These establish the development
invocation/ownership bridge, not a zero-exit runtime-cost pass or a native
ordinary comparison. The earlier wrapper refusal and both raw results remain.

Source checkpoint `dd66afa117` and exact hashes/receipts/exits are recorded in
`final-summary.json`. Correct framed helper SHA-256 is
`4dc2eea66907d81654dd45ba1eeee3b0c661826aa48ab4cdb557f46ed9264e4d`, built from
helper source `4fcf9fa06b` in dev profile with only reference-helper; ordinary
final qualification must rebuild/rebind after source freeze. New evidence uses
1.4 GiB; retained assigned target uses 3.3 GiB. All worker compiler/guest commands
are terminal, Go lease was explicitly released back to core, and no full
Activity/profile qualification ran. At handover shared free disk is 18 GiB;
other workers' targets/evidence were not cleaned.

## Additive helper build/copy custody

`build-helper --repo ROOT --cargo ABS_CARGO --rustc ABS_RUSTC --target ASSIGNED
--files FROZEN_FILES --output FRESH` executes the exact framed-helper Cargo
command with an absolute Cargo executable and explicit RUSTC. All helper
implementation/build inputs must match their immutable Git blobs and caller's
frozen file hashes before and after the command. It retains actual Cargo JSON,
command/PID/process receipts, compiler hashes, original artifact executable hash
and an observed byte-identical copy binding. The actual artifact executable must
remain available/frozen, alongside the independent copy.

Reference `run` now independently admits those raw receipts and immutable source
files: artifact metadata must equal the sealed Cargo JSON, process/command PID,
argv, interval and completion must match, exact reference-helper-only Cargo argv
and compiler must be present, and both artifact and copied executable must match
the same hash. Synthetic receipt-mutation tests exercise this admission; they
supply no additional runtime evidence. Earlier development receipts are retained
historically, and are not silently upgraded to this stronger contract.

The additive custody check now also requires the complete Git-tracked helper
source membership, refusing missing inactive source files as well as additions.
Actual `helper-build-v3/helper-build.json` has source commit `fa175a02a9`, exact
Cargo/RUSTC paths and hashes, all source-file Git blobs, raw artifact JSON,
command/process receipts, artifact executable identity and copy binding. Helper
source files are unchanged from `4fcf9fa06b`; the observed copied bytes retain
SHA-256 `4dc2eea66907d81654dd45ba1eeee3b0c661826aa48ab4cdb557f46ed9264e4d`.
`helper-admission-inputs-v3.json` lists every required frozen input hash.

First admission exposed a real receipt timing error: driver process registration
ends before its enclosing command receipt is written. Exact endpoint equality
was corrected to require equal start and process end within command end; the
failed admission exit 1 remains in `helper-build-v3.exit`, while actual Cargo
exit is 0. Re-admission is `helper-admission-v3.exit` 0. Nine focused tests pass
in `unit-helper-custody-final2.log`; fresh independent reviewer found no remaining
source blockers. The final reader must retain/seal the actual artifact path as
well as the independent copy and all declared receipt/source/compiler files.
