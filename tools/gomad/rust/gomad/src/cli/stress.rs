/// # Errors
/// Requires explicit stress admission and a complete original workload matrix.
pub fn run(arguments: &[String], output: &mut dyn std::io::Write) -> Result<i32, super::Error> {
    crate::stress::run(arguments, output).map_err(Into::into)
}
