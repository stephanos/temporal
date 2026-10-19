//! Synthetic system guest exercising the reusable ABI.

#[path = "../../../tests/support/mod.rs"]
mod support;

use umpire_wasm::Runtime;

umpire_wasm::export_guest!(Runtime<support::World, support::Realization>, support::runtime());
