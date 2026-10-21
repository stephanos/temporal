use super::{
    Error,
    explore::{self, ExploreOptions},
};
use crate::campaign::exploration::simulation::DimensionLimits;
use std::{collections::BTreeMap, io::Write, path::Path};

pub(crate) struct ExplorationOptions {
    pub base: ExploreOptions,
    pub strategy: String,
    pub max_executions: u64,
    pub max_choice_depth: u64,
    pub choice_start_ordinal: u64,
    pub max_exploration_bytes: u64,
    pub max_forced_decisions: u64,
    pub max_exploration_result_bytes: u64,
    pub dimension_limits: DimensionLimits,
}

pub(crate) fn try_run(
    arguments: &[String],
    output: &mut dyn Write,
    installation: Option<&Path>,
) -> Option<Result<i32, Error>> {
    let mut index = 0;
    let mut strategy = None;
    while let Some((name, supplied)) = explore::option(arguments, &mut index) {
        let Ok(name) = name else { break };
        if matches!(
            name,
            "json" | "choices" | "diagnostics" | "soft-assertion-report-only"
        ) {
            continue;
        }
        let Ok(value) = explore::value(arguments, &mut index, name, supplied) else {
            break;
        };
        if name == "strategy" {
            strategy = Some(value);
        }
    }
    if !matches!(
        strategy,
        Some("choice-exploration" | "simulation-exploration")
    ) {
        return None;
    }
    let mut json = false;
    let result = match parse(arguments, &mut json) {
        Ok(mut options) => {
            options.base.installation = installation.map(Path::to_path_buf);
            if options.strategy == "choice-exploration" {
                crate::campaign::choice::explore(&options, output)
            } else {
                crate::campaign::simulation::explore(&options, output)
            }
        }
        Err(error) => Err(error),
    };
    Some(match result {
        Ok(status) => Ok(status),
        Err(error) => explore::report_error(&error, json, output),
    })
}

#[expect(
    clippy::too_many_lines,
    reason = "Exploration bounds share one ordered flag and target grammar"
)]
pub(crate) fn parse(arguments: &[String], json: &mut bool) -> Result<ExplorationOptions, String> {
    let mut remaining = Vec::new();
    let mut bounds = BTreeMap::new();
    let mut strategy = String::new();
    let mut count = false;
    let mut index = 0;
    while let Some((name, supplied)) = explore::option(arguments, &mut index) {
        let name = name?;
        let boolean = matches!(
            name,
            "json" | "choices" | "diagnostics" | "soft-assertion-report-only"
        );
        if boolean {
            if name == "json" {
                *json = explore::boolean(supplied.unwrap_or("true"))?;
            }
            remaining.push(format!("--{name}={}", supplied.unwrap_or("true")));
            continue;
        }
        let value = explore::value(arguments, &mut index, name, supplied)?;
        match name {
            "strategy" => strategy = value.into(),
            "max-executions"
            | "max-choice-depth"
            | "choice-start-ordinal"
            | "max-exploration-bytes"
            | "max-forced-decisions"
            | "max-exploration-result-bytes"
            | "max-runtime-decisions"
            | "max-scenario-decisions"
            | "max-network-decisions"
            | "max-storage-decisions"
            | "max-fault-decisions"
            | "max-crash-decisions" => {
                bounds.insert(
                    name.to_owned(),
                    if name.ends_with("bytes") {
                        byte_size(value)?
                    } else {
                        unsigned(value)?
                    },
                );
            }
            "count" => {
                count = true;
                remaining.push(format!("--{name}={value}"));
            }
            _ => remaining.push(format!("--{name}={value}")),
        }
    }
    if count {
        return Err(format!("--strategy={strategy} does not accept --count"));
    }
    if !matches!(
        strategy.as_str(),
        "choice-exploration" | "simulation-exploration"
    ) {
        return Err(format!("unknown exploration strategy {strategy:?}"));
    }
    remaining.push("--choices=true".into());
    remaining.extend_from_slice(&arguments[index..]);
    let base = explore::parse_for_strategy(&remaining, json, strategy == "simulation-exploration")?;
    if crate::campaign::seeds::SeedSelection::parse(&base.seeds)?.count() != 1 {
        return Err(format!(
            "--strategy={strategy} requires exactly one base seed"
        ));
    }
    let positive = |name: &str| {
        bounds
            .get(name)
            .copied()
            .filter(|value| *value > 0)
            .ok_or_else(|| format!("--strategy={strategy} requires an explicit positive --{name}"))
    };
    let max_executions = positive("max-executions")?;
    let max_exploration_bytes = positive("max-exploration-bytes")?;
    let combined = [
        "max-forced-decisions",
        "max-exploration-result-bytes",
        "max-runtime-decisions",
        "max-scenario-decisions",
        "max-network-decisions",
        "max-storage-decisions",
        "max-fault-decisions",
        "max-crash-decisions",
    ];
    let (max_choice_depth, max_forced_decisions, max_exploration_result_bytes, dimension_limits) =
        if strategy == "choice-exploration" {
            if combined.iter().any(|name| bounds.contains_key(*name)) {
                return Err(
                    "simulation exploration bounds require --strategy=simulation-exploration"
                        .into(),
                );
            }
            (
                positive("max-choice-depth")?,
                0,
                0,
                DimensionLimits {
                    runtime: 0,
                    scenario: 0,
                    network: 0,
                    storage: 0,
                    fault: 0,
                    crash: 0,
                },
            )
        } else {
            for name in ["max-choice-depth", "choice-start-ordinal"] {
                if bounds.contains_key(name) {
                    return Err(format!(
                        "--strategy=simulation-exploration does not accept --{name}"
                    ));
                }
            }
            (
                0,
                positive("max-forced-decisions")?,
                positive("max-exploration-result-bytes")?,
                DimensionLimits {
                    runtime: positive("max-runtime-decisions")?,
                    scenario: positive("max-scenario-decisions")?,
                    network: positive("max-network-decisions")?,
                    storage: positive("max-storage-decisions")?,
                    fault: positive("max-fault-decisions")?,
                    crash: positive("max-crash-decisions")?,
                },
            )
        };
    Ok(ExplorationOptions {
        base,
        strategy,
        max_executions,
        max_choice_depth,
        choice_start_ordinal: bounds.get("choice-start-ordinal").copied().unwrap_or(0),
        max_exploration_bytes,
        max_forced_decisions,
        max_exploration_result_bytes,
        dimension_limits,
    })
}

fn unsigned(value: &str) -> Result<u64, String> {
    if value.is_empty() || !value.bytes().all(|byte| byte.is_ascii_digit()) {
        return Err(format!("invalid unsigned integer {value:?}"));
    }
    value
        .parse()
        .map_err(|_| format!("invalid unsigned integer {value:?}"))
}

fn byte_size(value: &str) -> Result<u64, String> {
    let (digits, factor) = [("KiB", 1_u64 << 10), ("MiB", 1 << 20), ("GiB", 1 << 30)]
        .into_iter()
        .find_map(|(suffix, factor)| value.strip_suffix(suffix).map(|digits| (digits, factor)))
        .unwrap_or((value, 1));
    if digits.len() > 1 && digits.starts_with('0') {
        return Err(format!("invalid byte size {value:?}"));
    }
    unsigned(digits)?
        .checked_mul(factor)
        .filter(|bytes| *bytes > 0)
        .ok_or_else(|| format!("invalid byte size {value:?}"))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn arguments(values: &[&str]) -> Vec<String> {
        values.iter().map(|value| (*value).into()).collect()
    }

    #[test]
    fn choice_bounds_preserve_target_arguments() {
        let values = arguments(&[
            "--strategy=choice-exploration",
            "--max-executions=4",
            "--max-choice-depth=2",
            "--max-exploration-bytes=1MiB",
            "go-test",
            "./sample",
            "--",
            "--strategy=target-option",
        ]);
        let options = parse(&values, &mut false).unwrap();
        assert_eq!(options.max_executions, 4);
        assert_eq!(options.max_choice_depth, 2);
        assert_eq!(options.max_exploration_bytes, 1 << 20);
        assert!(options.base.choices);
        assert_eq!(options.base.target_args, ["--strategy=target-option"]);
    }

    #[test]
    fn simulation_requires_every_dimension_bound() {
        let values = arguments(&[
            "--strategy=simulation-exploration",
            "--max-executions=4",
            "--max-exploration-bytes=1MiB",
            "--max-forced-decisions=4",
            "--max-exploration-result-bytes=1MiB",
            "--max-runtime-decisions=4",
            "--max-scenario-decisions=4",
            "--max-network-decisions=4",
            "--max-storage-decisions=4",
            "--max-fault-decisions=4",
            "go-test",
            "./sample",
        ]);
        assert!(
            matches!(parse(&values, &mut false), Err(error) if error.contains("max-crash-decisions"))
        );
    }

    #[test]
    fn exploration_dispatch_reports_bound_errors_as_json() {
        let values = arguments(&[
            "--json",
            "--strategy=choice-exploration",
            "go-test",
            "./sample",
        ]);
        let mut output = Vec::new();
        let result = explore::run(&values, &mut output);
        assert!(matches!(result, Ok(2)));
        let event: serde_json::Value = serde_json::from_slice(&output).unwrap();
        assert_eq!(event["classification"], "invalid_input");
        assert!(
            event["message"]
                .as_str()
                .unwrap()
                .contains("max-executions")
        );
    }
}
