//! Synthetic system guest exercising the reusable ABI.

#[path = "../../../tests/support/control.rs"]
mod control;
#[path = "../../../tests/support/mod.rs"]
#[allow(dead_code, reason = "Shared native fixture contains helpers outside this guest entrypoint")]
mod support;

umpire_runtime::export_guest!(control::Control, control::control());
