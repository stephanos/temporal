//! System-owned policy and negative controls for the generic source boundary.

#![forbid(unsafe_code)]

use std::{fs, path::Path};
use umpire_testing::boundary::{Violation, scan};

const EXTENSIONS: &[&str] = &["rs", "proto", "go", "toml"];
const FORBIDDEN: &[&str] =
    &["temporal", "Temporal", "go.temporal.io/", "temporal.api", "temporal::api", "temporal/api"];

fn organization_path(line: &str, offset: usize) -> bool {
    let path_char = |c: char| c.is_ascii_alphanumeric() || "._/-@".contains(c);
    let start = line[..offset]
        .char_indices()
        .rfind(|(_, c)| !path_char(*c))
        .map_or(0, |(i, c)| i + c.len_utf8());
    let end = line[offset..].find(|c| !path_char(c)).map_or(line.len(), |i| offset + i);
    let token = &line[start..end];
    offset == start + "github.com/".len()
        && token.strip_prefix("github.com/temporalio/").is_some_and(|rest| !rest.is_empty())
}

fn check(root: &Path) -> std::io::Result<Vec<Violation>> {
    scan(root, EXTENSIONS, FORBIDDEN, organization_path)
}

#[test]
fn generic_source_boundary() -> umpire_testing::TestResult {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../umpire");
    let violations = check(&root)?;
    assert!(
        violations.is_empty(),
        "{}",
        violations.iter().map(ToString::to_string).collect::<Vec<_>>().join("\n")
    );
    Ok(())
}

#[test]
fn boundary_controls() -> umpire_testing::TestResult {
    let root = std::env::temp_dir().join(format!("umpire-boundary-{}", std::process::id()));
    fs::create_dir_all(root.join("nested"))?;
    let result = (|| -> umpire_testing::TestResult {
        for extension in EXTENSIONS {
            fs::write(
                root.join(format!("nested/source.{extension}")),
                "ordinary\nTemporalClient temporal_client\nuse temporal::api;\nimport go.temporal.io/api;\nmodule temporal/api;\n// temporal.api.Workflow\n",
            )?;
        }
        let paths = [
            "github.com/temporalio/umpire",
            "github.com/temporalio/umpire/hosts/go",
            "temporalio",
            "github.com/temporalioish/umpire",
            "github.com/Temporalio/umpire",
            "github.com/temporalio",
            "github.com/temporalio/",
            "other/temporalio/umpire",
            "other/github.com/temporalio/umpire",
            "github.com/temporalio/Temporal",
            "github.com/temporalio/temporal",
            "github.com/temporalio/umpire Temporal",
            "github.com/temporalio/umpire_temporal",
            "github.com/temporalio/umpire temporalio",
        ];
        fs::write(root.join("paths.toml"), paths.join("\n"))?;
        fs::write(root.join("ignored.txt"), "Temporal")?;
        let mut expected = Vec::new();
        for extension in ["go", "proto", "rs", "toml"] {
            for line in 2..=6 {
                expected
                    .push(Violation { path: format!("nested/source.{extension}").into(), line });
            }
        }
        for line in 3..=paths.len() {
            expected.push(Violation { path: "paths.toml".into(), line });
        }
        let actual = check(&root)?;
        assert_eq!(actual, expected);
        assert_eq!(check(&root)?, actual);
        assert_eq!(actual[0].to_string(), "nested/source.go:2: forbidden source dependency");
        Ok(())
    })();
    fs::remove_dir_all(&root)?;
    result
}
