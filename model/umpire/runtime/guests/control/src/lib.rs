//! Synthetic system guest exercising the reusable ABI.

#[path = "../../../tests/support/mod.rs"]
mod support;

use umpire_runtime::Runtime;

umpire_runtime::export_guest!(Runtime<support::World, support::Realization>, support::runtime());
