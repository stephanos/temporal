//! Framework test runner leaf, including Cargo's workspace doctests.
use std::process::Command;
/// Append exactly one workspace test invocation to the admitted Cargo command.
pub fn configure(command: &mut Command) {
    command.args(["test", "--workspace", "--locked", "--quiet"]);
}
