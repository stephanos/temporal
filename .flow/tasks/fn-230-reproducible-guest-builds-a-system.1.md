---
satisfies: [R1]
---
# fn-230-reproducible-guest-builds-a-system.1 Guest profile and guest.sh in Umpire; reproducibility proven on two clean checkouts

## Description
Add [profile.guest] (inherits release; debug false, strip symbols, lto fat, codegen-units 1, overflow-checks true, panic unwind) and model/scripts/guest.sh <crate> <output> [--check] with --remap-path-prefix for workspace, registry and sysroot, wasm32-wasip1, --locked, mise toolchain; exit codes 0/1/2 as specified. Prove byte identity by building umpire-temporal-guest and umpire-stock-guest from two clean checkouts at different absolute paths; record sizes and the proof in DESIGN.md. Confirm the stock guest's wasm report is still byte-identical to the Rust goldens under the new profile. Gate and model-time pasted.

## Acceptance
- [ ] profile.guest and guest.sh present with the specified exit codes
- [ ] Two clean checkouts produce identical bytes for both guests; sizes in DESIGN.md
- [ ] Stock guest report byte-identical to goldens; gate passes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
