//! Diagnostic feature entry into the shared linked measurement producer.
pub use crate::measurement::execution::run;

/// # Errors
/// Retains a diagnostic run with the profile command's exact rerun grammar.
pub fn run_retained(
    input: &std::path::Path,
    root: &std::path::Path,
    keep: bool,
) -> Result<std::path::PathBuf, String> {
    crate::measurement::execution::run_retained_command(input, root, keep, "retained")
}
