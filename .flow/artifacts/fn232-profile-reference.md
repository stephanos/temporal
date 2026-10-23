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
