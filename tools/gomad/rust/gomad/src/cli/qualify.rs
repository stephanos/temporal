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
    emit(output, &published)
}

/// Writes the bounded response before any publication or cost/budget
/// failure, so the retained pointer and receipt always survive; a final
/// cell's cost/budget gate then precedes the application classification.
fn emit(output: &mut dyn Write, published: &cells::Qualified) -> Result<i32, super::Error> {
    let result_path = published.path.to_string_lossy().into_owned();
    let encoded = response(&published.path, &published.receipt)?;
    output
        .write_all(&encoded)
        .map_err(|error| status(3, format!("qualification response output failed: {error}")))?;
    if let Some(error) = &published.publication_error {
        return Err(publication_error(error));
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

/// The bounded single-line response: an absolute Result path of at most 1024
/// bytes and the operator receipt, at most 8 KiB encoded.
fn response(path: &Path, receipt: &impl serde::Serialize) -> Result<Vec<u8>, super::Error> {
    let result_path = path.to_string_lossy();
    if !path.is_absolute() || result_path.len() > 1024 {
        return Err(status(
            3,
            "qualification result_path must be absolute and at most 1024 bytes",
        ));
    }
    let mut encoded = canonical::encode(&serde_json::json!({
        "result_path": result_path,
        "operator_receipt": receipt,
    }))?;
    encoded.push(b'\n');
    if encoded.len() > 8 << 10 {
        return Err(status(3, "qualification response exceeds 8KiB limit"));
    }
    Ok(encoded)
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
    use super::{publication_error, response};

    /// Original `TestReportRejectsInvalidResultWithoutResolvingInstallation`:
    /// report takes no installation, refuses grammar with 2 and unreadable or
    /// invented Results with 3, and writes nothing on refusal.
    #[test]
    fn report_rejects_invalid_results_without_an_installation() {
        let root =
            crate::campaign::prepare::process::Temporary::new(&std::env::temp_dir()).unwrap();
        let invalid = root.0.join("invalid-result.json");
        std::fs::write(&invalid, br#"{"schema":"invented"}"#).unwrap();
        let invalid = invalid.to_string_lossy().into_owned();
        for (arguments, expected) in [
            (vec![], 2),
            (vec!["--result", "/missing/qualification-result"], 3),
            (vec!["--result", invalid.as_str()], 3),
            (
                vec!["--result", "/missing/qualification-result", "extra"],
                2,
            ),
        ] {
            let arguments = arguments.into_iter().map(str::to_owned).collect::<Vec<_>>();
            let mut output = Vec::new();
            let Err(error) = super::report(&arguments, &mut output) else {
                panic!("{arguments:?} reported an invalid Result");
            };
            assert_eq!(error.status, expected, "{arguments:?}: {}", error.message);
            assert!(output.is_empty() && !error.message.contains("installation"));
        }
    }

    /// Original `TestQualificationResponseBoundsAbsolutePathAndEncodedBytes`.
    #[test]
    fn response_bounds_absolute_path_and_encoded_bytes() {
        let receipt = serde_json::json!({});
        for path in [
            "relative/result.json".to_owned(),
            format!("/{}", "x".repeat(1024)),
        ] {
            let Err(error) = response(std::path::Path::new(&path), &receipt) else {
                panic!("unbounded or relative response path accepted");
            };
            assert_eq!(error.status, 3);
        }
        let Ok(encoded) = response(
            std::path::Path::new(&format!("/{}", "<".repeat(1023))),
            &receipt,
        ) else {
            panic!("bounded escaped path refused");
        };
        assert!(encoded.len() <= 8 << 10);
    }

    fn published(
        provisional: bool,
        target_success: bool,
        cost_failure: Option<&str>,
        budget_failure: Option<&str>,
        within_budget: Option<bool>,
    ) -> crate::qualify::cells::Qualified {
        let mut manifest = serde_json::json!({});
        for field in [
            "schema",
            "id",
            "workload",
            "profile",
            "integrated_source_sha256",
            "workload_source_sha256",
            "runtime_sha256",
            "compiler_sha256",
            "helper_sha256",
            "host_sha256",
            "model_sha256",
            "engine_sha256",
            "host_config_sha256",
            "module_sha256",
            "toolchain_sha256",
            "host_os",
            "host_arch",
        ] {
            manifest[field] = "fixture".into();
        }
        manifest["provisional"] = provisional.into();
        manifest["cell"] = "success".into();
        for (field, value) in [("seed", 7), ("repeat", 2), ("fuel", 1), ("memory_bytes", 1)] {
            manifest[field] = value.into();
        }
        manifest["unsupported"] = serde_json::json!([]);
        let budget = within_budget.map(|within| serde_json::json!({"resident_stores":1,"checks":[],"violations":[],"pending":[],"within_budget":within}));
        let result = serde_json::from_value(serde_json::json!({"schema":"fixture","manifest":manifest,
            "report":{"schema":"fixture","qualified":target_success,"deterministic":true,"target_success":target_success,"seed":7,"repeat":2,"command":["gomad"],"record_sha256":null,"first_divergence":null,"executions":[],"failure":null},
            "observations":{"operator_process_id":1,"resident_stores":1,"start":null,"prepared":null,"executed":null,"preparation_wall_nanos":1,"samples":[]},
            "cost":null,"cost_failure":cost_failure,"budget":budget,"budget_failure":budget_failure,"integrated_source_manifest":null,"provenance":{}}))
        .unwrap();
        crate::qualify::cells::Qualified {
            path: "/retained/result.json".into(),
            receipt: crate::qualify::result::Receipt {
                schema: "fixture".into(),
                result_sha256: crate::evidence::canonical::digest(b"published result"),
                operator_process_id: u64::from(std::process::id()),
                linked_after: None,
                operator_peak_rss_bytes: None,
            },
            result,
            publication_error: None,
        }
    }

    fn retained_response(output: &[u8], published: &crate::qualify::cells::Qualified) {
        let response: serde_json::Value = serde_json::from_slice(output).unwrap();
        assert_eq!(response["result_path"], "/retained/result.json");
        assert_eq!(
            response["operator_receipt"],
            serde_json::to_value(&published.receipt).unwrap()
        );
    }

    /// Original `TestFinalQualificationCostGatePreservesPublishedSupportOutcome`:
    /// a final cell's missing or failed cost/budget evidence exits 3 apart from
    /// the application outcome, a provisional cell keeps its application status,
    /// and the retained pointer and receipt are written before any status.
    #[test]
    fn final_cost_gate_preserves_published_support_outcome() {
        for (name, provisional, success, cost, budget_failure, within, expected) in [
            ("final success", false, true, None, None, Some(true), 0),
            (
                "final application failure",
                false,
                false,
                None,
                None,
                Some(true),
                1,
            ),
            (
                "final missing cost measurement",
                false,
                true,
                Some("missing CPU"),
                None,
                Some(true),
                3,
            ),
            (
                "final missing budget measurement",
                false,
                true,
                None,
                Some("missing operator RSS"),
                None,
                3,
            ),
            ("final absent budget", false, true, None, None, None, 3),
            (
                "final exceeded budget",
                false,
                true,
                None,
                None,
                Some(false),
                3,
            ),
            (
                "application failure with invalid cost",
                false,
                false,
                Some("missing CPU"),
                None,
                Some(true),
                3,
            ),
            (
                "provisional success diagnostic",
                true,
                true,
                Some("missing CPU"),
                Some("missing operator RSS"),
                None,
                0,
            ),
            (
                "provisional application diagnostic",
                true,
                false,
                Some("missing CPU"),
                Some("missing operator RSS"),
                None,
                1,
            ),
        ] {
            let published = published(provisional, success, cost, budget_failure, within);
            let before = published.result.clone();
            let mut output = Vec::new();
            let status = match super::emit(&mut output, &published) {
                Ok(status) => status,
                Err(error) => {
                    assert!(
                        error.message.contains("cost/budget")
                            && !error.message.contains("target_failure"),
                        "{name}: {}",
                        error.message
                    );
                    error.status
                }
            };
            assert_eq!(status, expected, "{name}");
            retained_response(&output, &published);
            assert_eq!(
                published.result, before,
                "{name}: raw support evidence changed"
            );
        }
    }

    /// Original `TestQualificationResponseRetainsReceiptBeforePublicationFailure`:
    /// a publication failure exits 3 as a publication error, never as an
    /// application failure or cost acceptance, after the response is written.
    #[test]
    fn response_retains_receipt_before_publication_failure() {
        for empty in [true, false] {
            let mut published = published(false, true, None, None, Some(true));
            if empty {
                published.receipt.result_sha256 = String::new();
                published.receipt.operator_process_id = 0;
            }
            published.publication_error = Some("missing publication self observation".into());
            let mut output = Vec::new();
            let error = super::emit(&mut output, &published).unwrap_err();
            assert_eq!(error.status, 3);
            assert!(
                error.message.contains("publication") && !error.message.contains("target_failure")
            );
            retained_response(&output, &published);
        }
    }

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
