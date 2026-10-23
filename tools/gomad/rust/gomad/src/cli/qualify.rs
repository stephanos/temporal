//! `qualify --spec FILE` and installation-free `report --result FILE`.

use crate::{
    evidence::canonical,
    qualify::{cells, report, result, spec, usage},
};
use std::{io::Write, path::Path};

fn status(status: i32, message: impl Into<String>) -> super::Error {
    super::Error {
        status,
        message: message.into(),
    }
}

/// Parses the existing single-flag grammar, `-name VALUE`, `--name VALUE` or
/// `--name=VALUE`, with no positional arguments.
fn single_flag<'a>(
    arguments: &'a [String],
    name: &str,
    usage: &str,
) -> Result<&'a str, super::Error> {
    let mut value = None;
    let mut index = 0;
    while index < arguments.len() {
        let argument = &arguments[index];
        if argument == "--" && index + 1 == arguments.len() {
            break;
        }
        let Some(flag) = argument
            .strip_prefix("--")
            .or_else(|| argument.strip_prefix('-'))
            .filter(|flag| !flag.is_empty() && argument != "-" && argument != "--")
        else {
            return Err(status(2, usage));
        };
        let (flag, supplied) = flag
            .split_once('=')
            .map_or((flag, None), |(flag, value)| (flag, Some(value)));
        if flag != name {
            return Err(status(2, format!("flag provided but not defined: -{flag}")));
        }
        let supplied = if let Some(supplied) = supplied {
            supplied
        } else {
            index += 1;
            arguments
                .get(index)
                .map(String::as_str)
                .ok_or_else(|| status(2, format!("flag needs an argument: -{name}")))?
        };
        value = Some(supplied);
        index += 1;
    }
    value
        .filter(|value| !value.is_empty())
        .ok_or_else(|| status(2, usage))
}

/// # Errors
/// Grammar and spec errors return 2; freeze, run and publication failures 3.
pub fn run(
    arguments: &[String],
    output: &mut dyn Write,
    installation: Option<&Path>,
) -> Result<i32, super::Error> {
    crate::qualify::require_release_role().map_err(|error| status(2, error))?;
    usage::enable_phases();
    let start = usage::snapshot();
    let path = single_flag(
        arguments,
        "spec",
        "qualify requires --spec FILE and no positional arguments",
    )?;
    let specification = spec::read(Path::new(path)).map_err(|error| status(2, error))?;
    let mut command = vec!["gomad".to_owned()];
    if let Some(installation) = installation {
        command.extend([
            "--installation".into(),
            installation.to_string_lossy().into_owned(),
        ]);
    }
    command.push("qualify".into());
    command.extend(arguments.iter().cloned());
    let frozen = cells::freeze(&specification, installation, command, start)
        .map_err(|error| status(3, error))?;
    let qualified = cells::run(&frozen).map_err(|error| status(3, error))?;
    let published = cells::publish(&frozen, qualified).map_err(|error| status(3, error))?;
    let result_path = published.path.to_string_lossy().into_owned();
    if !published.path.is_absolute() || result_path.len() > 1024 {
        return Err(status(
            3,
            "qualification result_path must be absolute and at most 1024 bytes",
        ));
    }
    let mut encoded = canonical::encode(&serde_json::json!({
        "result_path": result_path,
        "operator_receipt": published.receipt,
    }))?;
    encoded.push(b'\n');
    if encoded.len() > 8 << 10 {
        return Err(status(3, "qualification response exceeds 8KiB limit"));
    }
    output
        .write_all(&encoded)
        .map_err(|error| status(3, format!("qualification response output failed: {error}")))?;
    if let Some(error) = published.publication_error {
        return Err(publication_error(&error));
    }
    let retained = &published.result;
    if !retained.manifest.provisional
        && (retained.cost_failure.is_some()
            || retained.budget_failure.is_some()
            || !retained
                .budget
                .as_ref()
                .is_some_and(|budget| budget.within_budget))
    {
        let pending = retained
            .budget
            .as_ref()
            .map(|budget| budget.pending.join(", "))
            .filter(|pending| !pending.is_empty())
            .map(|pending| format!(" (unattributed: {pending})"))
            .unwrap_or_default();
        return Err(status(
            3,
            format!(
                "qualification cost/budget gate failed{pending}; inspect retained result {result_path}"
            ),
        ));
    }
    Ok(report::exit_status(report::classification(
        &retained.report,
    )))
}

/// # Errors
/// Refuses malformed grammar with 2 and unauthenticated or inconsistent
/// Results or receipts with 3; no installation, source or compiler is read.
pub fn report(arguments: &[String], output: &mut dyn Write) -> Result<i32, super::Error> {
    let path = Path::new(single_flag(
        arguments,
        "result",
        "report requires --result FILE and no positional arguments",
    )?);
    let opened = result::open(path).map_err(|error| status(3, error))?;
    let (receipt, terminal) =
        result::open_receipt(path, &opened).map_err(|error| status(3, error))?;
    let mut encoded = canonical::encode(&serde_json::json!({
        "classification": report::classification(&opened.report),
        "publication": terminal,
        "receipt": receipt,
        "result": opened,
    }))?;
    encoded.push(b'\n');
    output
        .write_all(&encoded)
        .map_err(|error| status(3, format!("write qualification report: {error}")))?;
    Ok(0)
}

fn publication_error(error: &str) -> super::Error {
    let mut end = error.len().min(1024);
    while !error.is_char_boundary(end) {
        end -= 1;
    }
    let suffix = if end < error.len() {
        " [truncated]"
    } else {
        ""
    };
    status(
        3,
        format!(
            "qualification publication observation failed: {}{suffix}",
            &error[..end]
        ),
    )
}

#[cfg(test)]
mod receiving_response_tests {
    use super::publication_error;

    #[test]
    fn publication_error_output_is_bounded_and_marks_truncation() {
        for detail in [
            "publication path diagnostic ".repeat(1024),
            "界".repeat(2048),
        ] {
            let error = publication_error(&detail);
            assert_eq!(error.status, 3);
            assert!(error.message.len() < 2 << 10);
            assert!(error.message.contains("publication"));
            assert!(error.message.contains("truncated"));
        }
        assert_eq!(
            publication_error("short").message,
            "qualification publication observation failed: short"
        );
    }
}
