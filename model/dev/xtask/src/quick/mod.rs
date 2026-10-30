//! Temporal authored-source selection for the independent quick command.
use std::{ffi::OsString, io, path::Path};

pub(crate) fn arguments(root: &Path, supplied: Vec<OsString>) -> io::Result<Vec<OsString>> {
    let root = root.canonicalize()?;
    let mut arguments = vec![
        "quick".into(),
        "--workspace".into(),
        root.clone().into_os_string(),
        "--workspace".into(),
        root.join("umpire").into_os_string(),
    ];
    for package in [
        "umpire-temporal-tests",
        "umpire-temporal",
        "umpire-temporal-api",
        "umpire-temporal-realization-macros",
    ] {
        let mut mapping = root.join("temporal/feature").into_os_string();
        mapping.push(format!("={package}"));
        arguments.extend(["--map".into(), mapping]);
    }
    arguments.extend([
        "--compile-only".into(),
        "umpire-temporal-guest".into(),
        "--always".into(),
        "umpire-temporal-tests".into(),
        "--test-target".into(),
        "umpire-temporal-tests=fixtures".into(),
        "--test-filter".into(),
        "umpire-temporal-tests=fixtures".into(),
    ]);
    arguments.extend(supplied);
    Ok(arguments)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bridge_selects_both_workspaces_and_the_authored_model_consumer() -> io::Result<()> {
        let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
        let arguments = arguments(&root, vec!["--base".into(), "HEAD~1".into()])?;
        assert_eq!(arguments[0], "quick");
        assert_eq!(arguments[2], root.as_os_str());
        assert_eq!(arguments[4], root.join("umpire").as_os_str());
        let mappings: Vec<_> = arguments
            .windows(2)
            .filter(|pair| pair[0] == "--map")
            .map(|pair| pair[1].clone())
            .collect();
        let expected: Vec<_> = [
            "umpire-temporal-tests",
            "umpire-temporal",
            "umpire-temporal-api",
            "umpire-temporal-realization-macros",
        ]
        .into_iter()
        .map(|package| {
            let mut mapping = root.join("temporal/feature").into_os_string();
            mapping.push(format!("={package}"));
            mapping
        })
        .collect();
        assert_eq!(mappings, expected);
        assert!(arguments.windows(2).any(|pair| pair == ["--always", "umpire-temporal-tests"]));
        assert!(
            arguments.windows(2).any(|pair| pair == ["--compile-only", "umpire-temporal-guest"])
        );
        assert!(
            arguments
                .windows(2)
                .any(|pair| pair == ["--test-target", "umpire-temporal-tests=fixtures"])
        );
        assert!(
            arguments
                .windows(2)
                .any(|pair| pair == ["--test-filter", "umpire-temporal-tests=fixtures"])
        );
        assert_eq!(&arguments[arguments.len() - 2..], ["--base", "HEAD~1"]);
        Ok(())
    }
}
