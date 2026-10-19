//! One mini activity run per guest instance through the reusable ABI hooks.

umpire_wasm::export_guest!(
    umpire_wasm::Runtime<umpire_temporal::World, umpire_temporal::Realization>,
    umpire_temporal::runtime()
);
