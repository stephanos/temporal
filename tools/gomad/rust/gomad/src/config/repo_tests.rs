use super::*;
use crate::campaign::prepare::process::Temporary;

fn module() -> Temporary {
    let root =
        Temporary::new(&Path::new(env!("CARGO_MANIFEST_DIR")).join("target/repo-tests")).unwrap();
    fs::write(root.0.join("go.mod"), "module example.test/repo\n").unwrap();
    for directory in ["runtime", "cmd/tool", "pkg/nested"] {
        fs::create_dir_all(root.0.join(directory)).unwrap();
    }
    root
}

#[test]
fn repo_discovers_quoted_comments_arrays_and_module_relative_paths() {
    let root = module();
    fs::write(
        root.0.join("gomad.toml"),
        r#"
# The hash inside a quoted path remains a literal character.
"runtime_root" = 'runtime' # runtime sources
'derive' = "tempo\u0072al"
guest = [
  './cmd/tool', # caller-relative package
  "pkg/nested/..", # lexical normalization
]
"#,
    )
    .unwrap();
    let config = Repo::discover(&root.0.join("pkg/nested")).unwrap();
    assert_eq!(config.module_root, fs::canonicalize(&root.0).unwrap());
    assert_eq!(
        config.runtime_root,
        fs::canonicalize(root.0.join("runtime")).unwrap()
    );
    assert_eq!(config.derive, Derivation::Temporal);
    assert_eq!(config.guest, ["cmd/tool", "pkg"]);
    fs::write(root.0.join("gomad.toml"), "runtime_root='runtime'\n").unwrap();
    let config = Repo::discover(&root.0).unwrap();
    assert_eq!(config.derive, Derivation::Default);
    assert_eq!(config.guest, Vec::<String>::new());
}

#[test]
fn repo_parser_respects_toml_strings_and_refuses_malformed_grammar() {
    let valid = [
        ("runtime_root = \"a#b\" # comment\n", "a#b"),
        ("runtime_root = 'a\\b'\n", "a\\b"),
        ("runtime_root = 'a\u{85}b' # \u{85}\n", "a\u{85}b"),
        ("runtime_root = \"\"\"\nrun\\\n  time\"\"\"\n", "runtime"),
        ("runtime_root = '''\nruntime'''\r\n", "runtime"),
        ("\"runtime_\\u0072oot\" = \"runtime\"", "runtime"),
    ];
    for (text, expected) in valid {
        assert_eq!(
            Parser::new(text).document().unwrap()["runtime_root"],
            Value::String(expected.into()),
            "{text}"
        );
    }
    for text in [
        "runtime_root='runtime' runtime_root='runtime'",
        "runtime_root='runtime'\n\"runtime_root\"='runtime'",
        "runtime_root='runtime'\nunknown='value'",
        "runtime_root.path='runtime'",
        "[runtime_root]\nvalue='runtime'",
        "runtime_root=1",
        "runtime_root=true",
        "guest=['a' 'b']",
        "guest=[1]",
        "guest=[['a']]",
        "guest=[,]",
        "guest=['a',,]",
        "guest=['a'",
        "runtime_root=\"bad\\q\"",
        "runtime_root=\"bad\\uD800\"",
        "runtime_root=\"bad\\U00110000\"",
        "runtime_root=\"bad\nline\"",
        "runtime_root='bad\rline'",
        "runtime_root='bad\u{7f}'",
        "runtime_root='runtime'\rderive='default'",
        "'''runtime_root'''='runtime'",
    ] {
        assert!(Parser::new(text).document().is_err(), "accepted {text:?}");
    }
}

#[test]
fn repo_refuses_types_absolute_paths_escapes_and_symlink_escape() {
    let root = module();
    let outside = module();
    std::os::unix::fs::symlink(&outside.0, root.0.join("outside")).unwrap();
    for text in [
        "runtime_root=['runtime']",
        "runtime_root='runtime'\nderive=[]",
        "runtime_root='runtime'\nderive='other'",
        "runtime_root='runtime'\nguest='cmd/tool'",
        "derive='default'",
        "runtime_root='/runtime'",
        "runtime_root='../runtime'",
        "runtime_root='runtime'\nguest=['/cmd/tool']",
        "runtime_root='runtime'\nguest=['../../cmd/tool']",
        "runtime_root='runtime'\nguest=['cmd/...']",
        "runtime_root='outside'",
        "runtime_root='outside/../runtime'",
        "runtime_root='runtime'\nguest=['outside']",
    ] {
        fs::write(root.0.join("gomad.toml"), text).unwrap();
        assert!(Repo::discover(&root.0).is_err(), "accepted {text}");
    }
    fs::write(root.0.join("gomad.toml"), "runtime_root='runtime'").unwrap();
    let saved = root.0.join("real-config");
    fs::rename(root.0.join("gomad.toml"), &saved).unwrap();
    std::os::unix::fs::symlink(saved, root.0.join("gomad.toml")).unwrap();
    assert!(Repo::optional(&root.0).is_err());
}

#[test]
fn repo_optional_absence_and_capacity_are_distinct_from_invalid_configuration() {
    let root = module();
    assert!(Repo::optional(&root.0).unwrap().is_none());
    assert!(Repo::discover(&root.0).is_err());
    fs::write(root.0.join("gomad.toml"), vec![b' '; (64 << 10) + 1]).unwrap();
    assert!(Repo::optional(&root.0).is_err());
    fs::write(root.0.join("gomad.toml"), [0xff]).unwrap();
    assert!(Repo::optional(&root.0).is_err());
    fs::remove_file(root.0.join("go.mod")).unwrap();
    assert!(Repo::optional(&root.0).unwrap().is_none());
    let text = format!("guest=[{}]", vec!["'cmd/tool'"; 1025].join(","));
    assert!(Parser::new(&text).document().is_err());
}
