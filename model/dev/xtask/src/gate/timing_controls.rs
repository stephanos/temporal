//! Exact soft-budget boundaries retained from the Model timing controls.
use std::io;

/// Exercise the warning renderer used by the native timing entry point.
pub(super) fn run() -> io::Result<i32> {
    const INNER: &str = "note: inner loop over soft target (fn-254 R6)\n";
    const GATE: &str = "note: gate over soft target (fn-254 R6)\n";
    // The committed table's check-tier and quick targets are the defaults.
    let defaults = xtask::gate::budgets::timing_targets(
        include_str!("budgets.toml"),
        xtask::gate::budgets::Tier::Check,
    )?;
    if defaults != (180.0, 60.0) {
        return Err(io::Error::other(format!("timing targets changed: {defaults:?}")));
    }
    for (total, inner, gate_budget, inner_budget, expected) in [
        (180.0, 60.0, Some(""), Some(""), String::new()),
        (180.0, 60.001, Some(""), Some(""), INNER.into()),
        (180.001, 60.0, Some(""), Some(""), GATE.into()),
        (4.0, 2.0, Some("4"), Some("2"), String::new()),
        (5.0, 3.0, Some("4"), Some("2"), format!("{GATE}{INNER}")),
        (180.0, 60.0, None, None, String::new()),
        (180.001, 60.001, None, None, format!("{GATE}{INNER}")),
        (0.0, 0.0, Some("-1"), Some("-1"), format!("{GATE}{INNER}")),
    ] {
        let mut output = Vec::new();
        super::timing::warnings(
            &mut output,
            (total, inner),
            (gate_budget, inner_budget),
            defaults,
        )?;
        if output != expected.as_bytes() {
            return Err(io::Error::other(format!(
                "timing warnings: expected {expected:?}, got {:?}",
                String::from_utf8_lossy(&output),
            )));
        }
    }
    Ok(0)
}

#[cfg(test)]
mod tests {
    #[test]
    fn soft_budget_defaults_equality_overrides_and_warning_order() -> std::io::Result<()> {
        assert_eq!(super::run()?, 0);
        Ok(())
    }
}
