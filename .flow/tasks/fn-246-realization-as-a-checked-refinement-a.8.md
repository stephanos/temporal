---
satisfies: [R1, R2]
---
# fn-246-realization-as-a-checked-refinement-a.8 Opt-in guest proof metadata and immutable semantic-version admission

## Description
Implement the conductor-selected semantic-version discovery amendment for R1/R2 using opt-in immutable guest metadata and generic Host admission. Preserve protobuf/export layout and all unmarked legacy artifacts. Establish production retention before ambiguity integration relies on version2.

**Size:** M
**Files:** generic guest export/proof-mode support, Go Host module inspection and focused producer/admission fixtures, protocol/guest-build contract documentation; fn-247 xtask check only where needed.
**Touches:** [model/umpire/protocol/runtime/src/{abi,lib}.rs, model/umpire/protocol/runtime/tests/**, model/umpire/protocol/runtime/guests/**, model/umpire/hosts/go/host/**, model/dev/xtask/**, model/umpire/protocol/proto/README.md, model/GUEST_BUILDS.md]

### Approach

All source coordinates are provisional planning anchors. Reanchor final fn-237.3 fields, emitted graph, accessors, golden hashes and fn-247 xtask/build owners after their committed closures before implementation. The current fn-237.3 source checkpoint and fn-247 plan do not satisfy those gates.

1. Reanchor fn-247's Rust guest builder and pinned wasm32-wasip1 fat-LTO/strip flags. Begin with a production-profile opt-in custom-section fixture plus unchanged default-v1 artifact guard. Rust #[used] does not alone guarantee final-link retention. If the closed metadata cannot survive the actual builder, report the exact blocker; no version export, probe RPC, Python helper, metadata postprocessor or new dependency is assumed.
2. Define one closed immutable `umpire.proof-semantics` custom section containing exactly four little-endian bytes encoding one unsigned semantic version, derived from the actual framework-owned proof-mode type/configuration. Legacy macro path emits no new section. Prefix2 opt-in path couples its metadata to whole-prefix semantics; forwarding annotation/facade wrappers retain that mode. Compile controls reject or make unrepresentable mismatched proof-mode advertisement/legacy export. No caller numeric version flag. A narrowly scoped unsafe link-section attribute allowance, if necessary, must be explicitly documented/reviewed; no blanket unsafe policy change.
3. At existing Host.New/validateModule, enable pinned wazero WithCustomSections(true), inspect CustomSections Name/Data before application calls, and expose immutable discovered semantics. Missing section means legacy1; explicit1 means delta and explicit2 means whole-prefix mode. Zero/unknown values, duplicate sections, any length other than four bytes or unsupported combinations refuse before Reset/Driver. Preserve exact function export whitelist/signatures and protobuf layouts. Add no capability-probe calls outside or inside Run history.
4. Bind the complete bytes/metadata to the existing model_hash/freshness contract. A stripped prefix advertisement fails producer/freshness/hash controls; missing metadata alone still denotes legacy1, so do not claim Host can distinguish a deliberately stripped prefix binary from genuine legacy code. Likewise unmodified old hosts ignore sections and are not supported readers for new prefix artifacts. Updated Host must retain old unmarked-v1 loading/replay support; no reverse-compatibility claim.

### Investigation targets

**Required** (source facts are in fn246-prefix-version-carrier-audit.md):

- `model/umpire/protocol/runtime/src/abi.rs:127-158` and `src/lib.rs` for three exports, transactional boundary and proof-mode composition.
- `model/umpire/hosts/go/host/host.go::New,validateModule` for existing compile/admission/strict export controls.
- Pinned wazero v1.12.0 `config.go:139-147,351`, `api/wasm.go:692-696` for section inspection.
- `model/Cargo.toml:134-140`, runtime lint policy and pinned Rust Reference ABI used/link_section constraints.
- Post-fn-247 `model/dev/xtask`, retained guest shell launcher and `model/GUEST_BUILDS.md` for actual production/freshness ownership.
- Existing generic guest export/import/signature negative controls for preservation.

### Verification

At implementation only, use the actual production guest builder/xtask, strict native/wasm checks and existing generic Go-host tests with test_dep. Inspect exact metadata and preserve v1 bytes/export sets. The planning audit ran no build/probe and does not claim linker retention proved.

## Acceptance
- [ ] Production fat-LTO/strip builder retains exactly the opt-in proof-mode-derived prefix2 metadata; default unmarked v1 guest bytes/export whitelist stay pinned. Mode mismatch and removed-prefix metadata fail compile/producer/freshness/hash controls without fabricated caller stamping.
- [ ] Updated Host discovers immutable semantics through pinned wazero metadata inspection before application/Driver I/O. One immutable `umpire.proof-semantics` section contains exactly four little-endian bytes for an unsigned version. Absence is legacy1, explicit1 is delta and explicit2 is prefix; zero/unknown values, duplicate sections, other lengths and unsupported advertisements refuse explicitly with no probe RPCs or added function export.
- [ ] Any metadata link-section attribute exception is narrow and reviewed; no new unsafe execution, dependency, Python, Wasm rewriter or schema field. Concrete production-retention failure remains blocking, not papered over by a version export.
- [ ] Legacy host limitations and stripped-metadata ambiguity are stated truthfully; updated reader preserves unmodified v1 guests and exact old fixtures while version2 artifacts require the updated admitted reader.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
