---
satisfies: [R1, R2]
---
# fn-216-mini-activity-vertical-slice-the-model.6 Temporal envelopes, realization and guest over the real API closure

## Description
Complete R1's realization and R2's system binding using the independent Model and
reusable generic core. This checkpoint proves the guest before .3 starts the SDK.

**Size:** M
**Files:** Temporal envelopes/API generation, realization, guest and extension Go module.
**Touches:** [model/temporal/**, model/Cargo.toml, model/Cargo.lock, model/scripts/**, Makefile]

### Approach
- Add exactly the three envelopes in the parent contract under the Temporal proto tree. Generate the upstream Temporal API types from the existing descriptor-generation target, with complete imports and well-known types wired to the core's shared owner. Do not copy Testpilot schemas or download a source tree.
- Make clean builds generate the descriptor before compiling Temporal crates. Compile the real generated closure natively and for wasm32-wasip1; cached or stubbed generation is insufficient evidence. Do not silently introduce runtime libraries.
- Generate the extension Go package under a separate `model/temporal/hosts/go/` module named `umpire.local/temporal`, with generated `umpire/temporal/v1/temporal.pb.go`. Keep its generator/freshness checks outside the generic host output. This preserves the existing root ownership gate's nested-module rule.
- Implement temporal.rs with strict outer/inner Any URL and RPC validation, the actual Info.Status/Info.RunState projection and the declared binary/plain Complete/Fail policy. Use the independently authored Model's transitions and callbacks; do not infer missing delivery or make new edges to explain observations.
- Validate policy and the selected enabled response before committing callback evidence for Poll. Return only Model-decided fields in partial Respond* requests; document which upstream callback/request fields the SDK owns or cannot expose.
- Native and WASM scenarios must pin Scheduled/Started/terminal repeats, terminal-before-delivery, stale Scheduled after delivery, opposite terminal after closure, unknown/malformed projection, wrong/missing URLs, malformed policy/cardinality, callback at a closed state, freeze and Reset. Refused callbacks leave Scheduled unchanged; returning Answer leaves Started.
- Link the Temporal guest against the generic library through its public hook and compile the existing stock guest too. Exactly one instance belongs to one run; no hidden global sharing.

### Investigation targets
**Required:**
- `Makefile:346` - existing descriptor generation prerequisite.
- `cmd/tools/getproto/main.go:315` - descriptor closure validation.
- `model/umpire/wasm/build.rs` - admitted prost generation pattern after .2.
- `model/scripts/proto-go.sh` - pinned generator/freshness pattern.
- `model/archived/temporal/features/activity_mini/realization/MiniActivityRun.scala.disabled:40` - delivery evidence precedent, read-only.
- `tools/umpire/ir/ownership_test.go:190` - nested Go module ownership rule.

### Quick commands
Run the actual new Temporal crate/native tests and wasm32-wasip1 guest build through mise. Run CGO_ENABLED=0 mise exec -- go test ./... in the extension module and the pinned generator freshness check. Run make model-check and make model-time from the root.

## Acceptance
- [ ] Exactly three generated envelopes and their real upstream API closure compile natively and as WASM from clean descriptor generation; generated Go freshness passes in the Temporal module.
- [ ] The realization pins parent-contract projections, input policy, callback delivery and partial replies using original Model behavior, including positive zero-step matches and every listed negative case.
- [ ] Refusal/Unexplained/reset/allocation behavior is consistent with the reusable core, with no invented Poll or partial callback mutation.
- [ ] Both Temporal and stock guests build and their focused native/WASM/Go checks and full model gate pass with recorded timing; no Temporal code enters Umpire.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
