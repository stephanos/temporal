//! Actual checker defect control using the shared native fixture.

#[path = "../../../tests/support/panic.rs"]
mod control;

umpire_runtime::export_guest!(control::Model, control::model(control::support::Mode::Overflow));
