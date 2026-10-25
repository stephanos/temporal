---
satisfies: [R1]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.34 Bump wasmtime off 47.0.3 to clear the 20 cargo-deny advisories

## Description
`cargo deny --workspace check -D warnings` (policy: `tools/gomad/rust/deny.toml`) fails with 20 RustSec advisories against the pinned `wasmtime = "=47.0.3"` in `wasmhost/Cargo.toml`: RUSTSEC-2026-0268, -0269, -0315, -0316, -0325, -0326, -0327, -0353..-0359, -0361..-0364, -0378, -0379. There is no fixed 47.x release; fixed lines are >=48.0.7 <49 or >=49.0.4.

Several of these touch what R1 depends on, not only features we leave off. Examples: `call_ref`/exception `catch` dropping fuel accounting, dynamic record lifting beyond the hostcall fuel limit, aarch64 Cranelift `br_table` mis-targeting (the host is linux/arm64), and mis-typed tag imports corrupting the GC heap. Others are in the component model, WASIp3 or the filesystem and are probably out of reach with `default-features = false, features = [cranelift, runtime, std, wat]`. Nobody has triaged them one by one.

Do: bump the wasmtime pin (and the cranelift/wasm-encoder/wast versions it drags in) to a fixed line; prefer the newest 49.x. Update the version-pinned `[bans] skip` entries in `deny.toml` that the bump invalidates; `-D warnings` reports a stale skip as `unmatched-skip`. Do not add `[advisories] ignore` entries unless one advisory is shown to be unreachable for our feature set, with a per-ID reason.

Ordering: this is a host-runtime change, so it lands behind whichever fn-242 lane owns the gate at the time and goes through the full qualification cycle, not just unit tests. Engine upgrades can change fuel consumption, and that moves fuel-bounded outcomes.

## Acceptance
- `cargo deny --workspace check -D warnings` in `tools/gomad/rust` exits 0 (advisories, bans, licenses, sources).
- wasmtime is pinned to a release that fixes every listed advisory; `Cargo.lock` has no vulnerable wasmtime/cranelift crate.
- The `[bans] skip` list in `deny.toml` matches the post-bump duplicate set exactly, each entry with a reason.
- A green qualification cycle on head after the bump; any change in fuel-bounded outcomes is explained in the task evidence.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
