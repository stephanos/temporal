use super::explore::{boolean, option, report_error};
use std::{io::Write, path::Path};

/// # Errors
/// Reports invalid recovery commands and campaign journal validation failures.
pub fn run(arguments: &[String], output: &mut dyn Write) -> Result<i32, super::Error> {
    let mut json = false;
    let result = (|| {
        let mut index = 0;
        while let Some((name, supplied)) = option(arguments, &mut index) {
            let name = name?;
            match name {
                "json" => json = boolean(supplied.unwrap_or("true"))?,
                _ => return Err(format!("flag provided but not defined: -{name}")),
            }
        }
        if arguments.len() - index != 1 || arguments[index].is_empty() {
            return Err("recover requires one campaign path".into());
        }
        recover(Path::new(&arguments[index]), json, output)
    })();
    match result {
        Ok(status) => Ok(status),
        Err(error) if error.starts_with("campaign writer lock unavailable: ") => {
            report_error(&format!("runner_failure: {error}"), json, output)
        }
        Err(error) => report_error(&error, json, output),
    }
}

fn recover(path: &Path, json: bool, output: &mut dyn Write) -> Result<i32, String> {
    let path = std::path::absolute(path)
        .map_err(|error| format!("runner_failure: resolve recovery path: {error}"))?;
    let changed = if crate::campaign::exploration::plan::is_round_campaign(&path)? {
        crate::campaign::exploration::round::RoundJournal::open(&path, true)?.recovery_changed()
    } else {
        crate::campaign::ledger::Journal::open(&path, true)?.recovery_changed()
    };
    let action = if changed { "repair" } else { "none" };
    if json {
        writeln!(
            output,
            "{}",
            serde_json::json!({
                "schema":"gomad.recovery/v1", "path":path, "action":action, "changed":changed
            })
        )
    } else {
        writeln!(
            output,
            "gomad recover: path={} action={action} changed={changed}",
            path.display()
        )
    }
    .map_err(|error| format!("runner_failure: write recovery report: {error}"))?;
    Ok(0)
}
