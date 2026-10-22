use super::{exploration, explore};
use std::{
    io::Write,
    path::{Path, PathBuf},
};

/// # Errors
/// Reports malformed plans, preparation failures and publication failures.
pub fn run(
    arguments: &[String],
    output: &mut dyn Write,
    installation: Option<&Path>,
) -> Result<i32, super::Error> {
    let mut json = false;
    let result: Result<i32, String> = (|| {
        let mut remaining = Vec::new();
        let mut destination = None;
        let mut strategy = None;
        let mut index = 0;
        while let Some((name, supplied)) = explore::option(arguments, &mut index) {
            let name = name?;
            if name == "json" {
                json = explore::boolean(supplied.unwrap_or("true"))?;
            }
            if matches!(
                name,
                "json" | "choices" | "diagnostics" | "soft-assertion-report-only"
            ) {
                remaining.push(format!("--{name}={}", supplied.unwrap_or("true")));
                continue;
            }
            let value = explore::value(arguments, &mut index, name, supplied)?;
            if name == "output" {
                if destination.replace(PathBuf::from(value)).is_some() {
                    return Err("--output must be supplied once".into());
                }
            } else {
                if name == "strategy" {
                    strategy = Some(value.to_owned());
                }
                remaining.push(format!("--{name}={value}"));
            }
        }
        remaining.extend_from_slice(&arguments[index..]);
        let destination = destination
            .filter(|path| !path.as_os_str().is_empty())
            .ok_or("--output is required")?;
        let prepared = if matches!(
            strategy.as_deref(),
            Some("choice-exploration" | "simulation-exploration")
        ) {
            let mut options = exploration::parse(&remaining, &mut json)?;
            options.base.installation = installation.map(Path::to_path_buf);
            if options.strategy != "choice-exploration" {
                return Err("public plan requires seed or choice-exploration strategy".into());
            }
            crate::campaign::preparation::check_output(&destination)?;
            crate::campaign::preparation::choice(&options)?
        } else {
            let mut options = explore::parse(&remaining, &mut json)?;
            options.installation = installation.map(Path::to_path_buf);
            crate::campaign::preparation::check_output(&destination)?;
            crate::campaign::preparation::seed(&options)?
        };
        let command = std::env::args().collect();
        let (receipt_path, receipt, receipt_sha256) = prepared.publish(&destination, command)?;
        crate::campaign::preparation::open_receipt(&receipt_path, &receipt_sha256)?;
        if json {
            writeln!(output, "{}", serde_json::json!({"schema":"gomad.plan-result/v1", "plan":receipt.plan,
                "plan_sha256":receipt.plan_sha256,"campaign":receipt.campaign,"guest_executions":receipt.guest_executions,
                "module_sha256":receipt.module_sha256,"envelope":prepared.envelope,"preparation_receipt":receipt_path,"preparation_receipt_sha256":receipt_sha256}))
        } else {
            writeln!(output, "plan={} preparation_receipt={} guest_executions={}", receipt.plan.display(), receipt_path.display(), receipt.guest_executions)
        }.map_err(|error| format!("runner_failure: write plan report: {error}"))?;
        Ok(0)
    })();
    match result {
        Ok(status) => Ok(status),
        Err(error) => explore::report_error(&error, json, output),
    }
}
