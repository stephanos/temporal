#[cfg(test)]
mod tests {
    use super::*;
    fn options() -> Options {
        Options::parse(
            &[
                "checkout",
                "gomad",
                "compiler",
                "helper",
                "runtime-root",
                "out",
                "work",
                "artifacts",
            ]
            .into_iter()
            .flat_map(|name| [format!("--{name}"), format!("/fixture/{name}")])
            .chain(
                [
                    "--config-source=throwaway",
                    "--packages=./tests/...",
                    "--per-test-timeout=10m",
                    "--prepare-timeout=30m",
                    "--overall-budget=24h",
                    "--retain-bytes=0",
                ]
                .map(str::to_owned),
            )
            .collect::<Vec<_>>(),
            &[],
        )
        .unwrap()
    }
    #[test]
    fn explicit_zero_retention_and_default_seed_are_accepted() {
        let value = options();
        assert_eq!((value.retain_bytes, value.seed, value.max_tests), (0, 0, 0));
        assert_eq!(value.per_test_timeout, Duration::from_secs(600));
    }
    #[test]
    fn inherited_inputs_overlaps_and_unbounded_budgets_refuse() {
        let mut value = options();
        value.environ.insert("GOMAD_SEED".into(), "0".into());
        assert!(value.validate().unwrap_err().contains("unset GOMAD_SEED"));
        value.environ.clear();
        value.work = value.checkout.join("scratch");
        assert!(value.validate().unwrap_err().contains("outside"));
        value.work = PathBuf::from("/fixture/work");
        value.prepare_timeout = Duration::ZERO;
        assert!(value.validate().unwrap_err().contains("positive"));
        value.prepare_timeout = Duration::from_secs(7201);
        assert!(value.validate().unwrap_err().contains("exceeds"));
    }
    #[test]
    fn original_explicit_options_refusal_matrix() {
        for case in [
            "gomad-variable",
            "gomad-empty-variable",
            "zero-per-test",
            "zero-prepare",
            "zero-overall",
            "huge-per-test",
            "huge-prepare",
            "huge-overall",
            "budget-fits-nothing",
            "relative-checkout",
            "missing-gomad",
            "work-in-checkout",
            "artifacts-in-checkout",
            "config-source",
            "no-packages",
            "package-outside-tests",
            "package-escapes-tests",
            "work-overlaps-out",
            "artifacts-in-work",
        ] {
            let mut value = options();
            match case {
                "gomad-variable" => {
                    value.environ.insert("GOMAD_SEED".into(), "1".into());
                }
                "gomad-empty-variable" => {
                    value.environ.insert("GOMAD_SEED".into(), String::new());
                }
                "zero-per-test" => value.per_test_timeout = Duration::ZERO,
                "zero-prepare" => value.prepare_timeout = Duration::ZERO,
                "zero-overall" => value.overall_budget = Duration::ZERO,
                "huge-per-test" => value.per_test_timeout = Duration::from_secs(21601),
                "huge-prepare" => value.prepare_timeout = Duration::from_secs(7201),
                "huge-overall" => value.overall_budget = Duration::from_secs(14 * 24 * 3600 + 1),
                "budget-fits-nothing" => value.overall_budget = Duration::from_secs(1),
                "relative-checkout" => value.checkout = "relative".into(),
                "missing-gomad" => value.gomad = PathBuf::new(),
                "work-in-checkout" => value.work = value.checkout.join("work"),
                "artifacts-in-checkout" => value.artifacts = value.checkout.join("artifacts"),
                "config-source" => value.config_source = "unknown".into(),
                "no-packages" => value.packages.clear(),
                "package-outside-tests" => value.packages = vec!["./common/...".into()],
                "package-escapes-tests" => value.packages = vec!["./tests/../common".into()],
                "work-overlaps-out" => value.work = value.out.join("work"),
                "artifacts-in-work" => value.artifacts = value.work.join("artifacts"),
                _ => unreachable!(),
            }
            assert!(value.validate().is_err(), "{case}");
        }
        for (flag, invalid) in [("--max-tests", "-1"), ("--overall-budget", "-1s")] {
            let arguments = ["--retain-bytes=0".to_owned(), format!("{flag}={invalid}")];
            assert!(Options::parse(&arguments, &[]).is_err(), "{flag}");
        }
    }
    #[test]
    fn byte_counts_and_compound_duration_preserve_bounds() {
        for (input, expected) in [
            ("0", 0),
            ("1024", 1024),
            ("5GiB", 5 << 30),
            ("16MiB", 16 << 20),
            ("3KiB", 3 << 10),
        ] {
            assert_eq!(parse_bytes(input).unwrap(), expected, "{input}");
        }
        for input in ["", "-1", "1.5GiB", "5GB", "18446744073709551615GiB"] {
            assert!(parse_bytes(input).is_err(), "{input}");
        }
        assert_eq!(parse_bytes("5GiB").unwrap(), 5 << 30);
        assert!(parse_bytes("18446744073709551615GiB").is_err());
        assert!(parse_bytes("1.5MiB").is_err());
        assert_eq!(
            parse_duration("1h2m3.5s").unwrap(),
            Duration::from_millis(3_723_500)
        );
        assert_eq!(parse_duration("0").unwrap(), Duration::ZERO);
    }
    #[test]
    fn dirty_patterns_and_missing_explicit_retention_refuse() {
        let mut value = options();
        for pattern in [
            "./tests/../common",
            "./tests//a",
            "tests/...",
            "./tests2/...",
            "./tests/./a",
        ] {
            value.packages = vec![pattern.into()];
            assert!(value.validate().is_err(), "{pattern}");
        }
        assert!(Options::parse(&[], &[]).is_err());
    }
}

use std::{
    collections::BTreeMap,
    path::{Component, Path, PathBuf},
    time::Duration,
};

#[derive(Clone, Debug)]
pub struct Options {
    pub checkout: PathBuf,
    pub gomad: PathBuf,
    pub compiler: PathBuf,
    pub helper: PathBuf,
    pub runtime_root: PathBuf,
    pub config_source: String,
    pub packages: Vec<String>,
    pub max_tests: usize,
    pub seed: u64,
    pub per_test_timeout: Duration,
    pub prepare_timeout: Duration,
    pub overall_budget: Duration,
    pub retain_bytes: u64,
    pub out: PathBuf,
    pub work: PathBuf,
    pub artifacts: PathBuf,
    pub environ: BTreeMap<String, String>,
}

pub fn flags(args: &[String]) -> Result<BTreeMap<String, String>, String> {
    let allowed = [
        "verify",
        "checkout",
        "gomad",
        "compiler",
        "helper",
        "runtime-root",
        "config-source",
        "packages",
        "max-tests",
        "seed",
        "per-test-timeout",
        "prepare-timeout",
        "overall-budget",
        "retain-bytes",
        "out",
        "work",
        "artifacts",
    ];
    let mut values = BTreeMap::new();
    let mut index = 0;
    while index < args.len() {
        let arg = &args[index];
        let name = arg
            .strip_prefix("--")
            .or_else(|| arg.strip_prefix('-'))
            .ok_or_else(|| format!("unexpected argument {arg}"))?;
        let (name, inline) = name
            .split_once('=')
            .map_or((name, None), |(n, v)| (n, Some(v)));
        if !allowed.contains(&name) {
            return Err(format!("flag provided but not defined: -{name}"));
        }
        let value = if let Some(value) = inline {
            value
        } else {
            index += 1;
            args.get(index)
                .ok_or_else(|| format!("flag needs an argument: -{name}"))?
        };
        values.insert(name.to_owned(), value.to_owned());
        index += 1;
    }
    Ok(values)
}
impl Options {
    pub fn parse(args: &[String], environ: &[(String, String)]) -> Result<Self, String> {
        let values = flags(args)?;
        let required = |name: &str| {
            values
                .get(name)
                .filter(|v| !v.is_empty())
                .cloned()
                .ok_or_else(|| format!("--{name} is required"))
        };
        let retain_bytes = parse_bytes(&required("retain-bytes")?)?;
        let duration = |name: &str| {
            values
                .get(name)
                .map_or(Ok(Duration::ZERO), |value| parse_duration(value))
        };
        let value = Self {
            checkout: values.get("checkout").cloned().unwrap_or_default().into(),
            gomad: values.get("gomad").cloned().unwrap_or_default().into(),
            compiler: values.get("compiler").cloned().unwrap_or_default().into(),
            helper: values.get("helper").cloned().unwrap_or_default().into(),
            runtime_root: values
                .get("runtime-root")
                .cloned()
                .unwrap_or_default()
                .into(),
            config_source: values.get("config-source").cloned().unwrap_or_default(),
            packages: values
                .get("packages")
                .cloned()
                .unwrap_or_default()
                .split(',')
                .map(str::trim)
                .filter(|p| !p.is_empty())
                .map(str::to_owned)
                .collect(),
            max_tests: values.get("max-tests").map_or(Ok(0), |v| {
                v.parse::<usize>()
                    .map_err(|_| "--max-tests must be nonnegative".to_owned())
            })?,
            seed: values.get("seed").map_or(Ok(0), |v| {
                v.parse::<u64>().map_err(|_| "invalid seed".to_owned())
            })?,
            per_test_timeout: duration("per-test-timeout")?,
            prepare_timeout: duration("prepare-timeout")?,
            overall_budget: duration("overall-budget")?,
            retain_bytes,
            out: values.get("out").cloned().unwrap_or_default().into(),
            work: values.get("work").cloned().unwrap_or_default().into(),
            artifacts: values.get("artifacts").cloned().unwrap_or_default().into(),
            environ: environ.iter().cloned().collect(),
        };
        value.validate()?;
        Ok(value)
    }
    pub fn validate(&self) -> Result<(), String> {
        for key in self.environ.keys() {
            if key.starts_with("GOMAD_") {
                return Err(format!("unset {key}; pass inputs as flags"));
            }
        }
        for (name, value) in [
            ("checkout", &self.checkout),
            ("gomad", &self.gomad),
            ("compiler", &self.compiler),
            ("helper", &self.helper),
            ("runtime-root", &self.runtime_root),
            ("out", &self.out),
            ("work", &self.work),
            ("artifacts", &self.artifacts),
        ] {
            if value.as_os_str().is_empty() {
                return Err(format!("--{name} is required"));
            }
            if !clean_absolute(value) {
                return Err(format!("--{name} must be an absolute clean path"));
            }
        }
        for (name, value) in [("work", &self.work), ("artifacts", &self.artifacts)] {
            if value.starts_with(&self.checkout) {
                return Err(format!("--{name} must be outside --checkout"));
            }
        }
        if self.out == self.checkout {
            return Err("--out must not be checkout itself".into());
        }
        if !["throwaway", "temporal-install"].contains(&self.config_source.as_str()) {
            return Err("--config-source must be throwaway or temporal-install".into());
        }
        if self.packages.is_empty() {
            return Err("--packages is required".into());
        }
        for pattern in &self.packages {
            let base = pattern.strip_suffix("/...").unwrap_or(pattern);
            if !(base == "./tests" || base.starts_with("./tests/"))
                || base[2..]
                    .split('/')
                    .any(|p| p.is_empty() || matches!(p, "." | ".."))
                || base.contains(['\\', '\0'])
            {
                return Err(format!(
                    "package pattern {pattern:?} is not a clean pattern under ./tests"
                ));
            }
        }
        for value in [&self.out, &self.artifacts] {
            if self.work.starts_with(value) || value.starts_with(&self.work) {
                return Err("--work must not overlap --out or --artifacts".into());
            }
        }
        for (name, budget, max) in [
            ("per-test-timeout", self.per_test_timeout, 21600),
            ("prepare-timeout", self.prepare_timeout, 7200),
            ("overall-budget", self.overall_budget, 14 * 24 * 3600),
        ] {
            if budget.is_zero() {
                return Err(format!("--{name} must be a positive bound"));
            }
            if budget > Duration::from_secs(max) {
                return Err(format!("--{name} exceeds maximum"));
            }
        }
        if self.overall_budget < self.per_test_timeout + self.prepare_timeout {
            return Err("--overall-budget cannot fit one test (per-test + prepare)".into());
        }
        Ok(())
    }
    pub fn require_files(&self) -> Result<(), String> {
        use std::os::unix::fs::PermissionsExt as _;
        for (name, path) in [
            ("gomad", &self.gomad),
            ("compiler", &self.compiler),
            ("helper", &self.helper),
        ] {
            let meta = std::fs::symlink_metadata(path).map_err(|e| format!("--{name}: {e}"))?;
            if !meta.is_file() || meta.permissions().mode() & 0o111 == 0 {
                return Err(format!("--{name} is not an executable regular file"));
            }
        }
        for path in [
            &self.checkout,
            &self.runtime_root,
            &self.checkout.join("tests/testcore"),
        ] {
            if !path.is_dir() {
                return Err(format!("{} is not a directory", path.display()));
            }
        }
        Ok(())
    }
}
fn clean_absolute(path: &Path) -> bool {
    let value = path.to_string_lossy();
    path.is_absolute()
        && (value == "/" || !value.ends_with('/'))
        && !value.contains(['\0', '\\'])
        && !value
            .split('/')
            .skip(1)
            .any(|p| p.is_empty() || matches!(p, "." | ".."))
        && path
            .components()
            .all(|p| matches!(p, Component::RootDir | Component::Normal(_)))
}
pub fn parse_bytes(value: &str) -> Result<u64, String> {
    let (digits, scale) = [("GiB", 1_u64 << 30), ("MiB", 1 << 20), ("KiB", 1 << 10)]
        .into_iter()
        .find_map(|(suffix, scale)| value.strip_suffix(suffix).map(|v| (v, scale)))
        .unwrap_or((value, 1));
    if digits.is_empty() || !digits.bytes().all(|b| b.is_ascii_digit()) {
        return Err("want a whole byte count with optional KiB, MiB or GiB".into());
    }
    digits
        .parse::<u64>()
        .ok()
        .and_then(|n| n.checked_mul(scale))
        .ok_or_else(|| "byte count overflows uint64".into())
}
pub fn parse_duration(value: &str) -> Result<Duration, String> {
    if value == "0" {
        return Ok(Duration::ZERO);
    }
    let mut remaining = value;
    let mut total = 0_u128;
    while !remaining.is_empty() {
        let end = remaining
            .find(|c: char| !c.is_ascii_digit() && c != '.')
            .ok_or("duration requires a unit")?;
        let digits = &remaining[..end];
        remaining = &remaining[end..];
        let (whole, fraction) = digits.split_once('.').unwrap_or((digits, ""));
        if whole.is_empty() && fraction.is_empty()
            || !whole
                .bytes()
                .chain(fraction.bytes())
                .all(|c| c.is_ascii_digit())
            || fraction.len() > 9
        {
            return Err("invalid duration".into());
        }
        let (unit, scale) = [
            ("ns", 1_u128),
            ("us", 1_000),
            ("µs", 1_000),
            ("ms", 1_000_000),
            ("s", 1_000_000_000),
            ("m", 60_000_000_000),
            ("h", 3_600_000_000_000),
        ]
        .into_iter()
        .find(|(unit, _)| remaining.starts_with(unit))
        .ok_or("invalid duration unit")?;
        remaining = &remaining[unit.len()..];
        let whole = if whole.is_empty() {
            0
        } else {
            whole.parse::<u128>().map_err(|_| "duration overflow")?
        };
        let part = whole
            .checked_mul(scale)
            .and_then(|n| {
                fraction
                    .parse::<u128>()
                    .unwrap_or(0)
                    .checked_mul(scale)
                    .and_then(|f| {
                        n.checked_add(f / 10_u128.pow(u32::try_from(fraction.len()).ok()?))
                    })
            })
            .ok_or("duration overflow")?;
        total = total.checked_add(part).ok_or("duration overflow")?;
    }
    if total > u128::from(u64::MAX) {
        return Err("duration must be a positive bounded interval".into());
    }
    Ok(Duration::from_nanos(
        u64::try_from(total).map_err(|_| "duration overflow")?,
    ))
}
