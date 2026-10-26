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
        "--map".into(),
    ];
    let mut mapping = root.join("temporal/feature").into_os_string();
    mapping.push("=umpire-temporal-tests");
    arguments.extend([
        mapping,
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
        let mut mapping = root.join("temporal/feature").into_os_string();
        mapping.push("=umpire-temporal-tests");
        assert_eq!(arguments[6], mapping);
        assert_eq!(arguments[8], "umpire-temporal-tests");
        assert_eq!(arguments[10], "umpire-temporal-tests=fixtures");
        assert_eq!(arguments[12], "umpire-temporal-tests=fixtures");
        assert_eq!(&arguments[13..], ["--base", "HEAD~1"]);
        Ok(())
    }
}
