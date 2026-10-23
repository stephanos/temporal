use super::*;
use crate::sweep::process::Completed;
use std::{
    collections::VecDeque,
    time::{Instant, SystemTime, UNIX_EPOCH},
};

struct Fixture {
    root: PathBuf,
    options: Options,
}
impl Fixture {
    fn new() -> Self {
        let unique = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        let root =
            std::env::temp_dir().join(format!("gomad-enumerate-{}-{unique}", std::process::id()));
        std::fs::create_dir(&root).unwrap();
        let checkout = root.join("checkout");
        std::fs::create_dir_all(checkout.join("tests")).unwrap();
        let options = Options {
            checkout,
            gomad: root.join("gomad"),
            compiler: root.join("go"),
            helper: root.join("helper"),
            runtime_root: root.join("runtime"),
            config_source: "throwaway".into(),
            packages: vec!["./tests/...".into()],
            max_tests: 0,
            seed: 0,
            per_test_timeout: Duration::from_secs(60),
            prepare_timeout: Duration::from_secs(120),
            overall_budget: Duration::from_secs(3600),
            retain_bytes: 0,
            out: root.join("out"),
            work: root.join("work"),
            artifacts: root.join("artifacts"),
            environ: BTreeMap::from([
                ("HOME".into(), "/operator-home".into()),
                ("GOFLAGS".into(), "-mod=mod".into()),
                ("GOWORK".into(), "/ambient/work".into()),
                ("GOTOOLCHAIN".into(), "auto".into()),
                ("GOOS".into(), "plan9".into()),
                ("GOARCH".into(), "arm".into()),
                ("CGO_ENABLED".into(), "1".into()),
                ("GOENV".into(), "/ambient/env".into()),
                ("GOEXPERIMENT".into(), "ambient".into()),
            ]),
        };
        Self { root, options }
    }
    fn write(&self, relative: &str, source: &str) -> PathBuf {
        let path = self.options.checkout.join(relative);
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(&path, source).unwrap();
        path
    }
    fn context() -> Context {
        Context::new(Instant::now() + Duration::from_secs(60))
    }
}
impl Drop for Fixture {
    fn drop(&mut self) {
        std::fs::remove_dir_all(&self.root).unwrap();
    }
}

struct Expected {
    directory: PathBuf,
    args: Vec<String>,
    guest: bool,
    timeout: Duration,
    completed: Completed,
}
struct MetadataExecutor {
    compiler: PathBuf,
    expected: VecDeque<Expected>,
}
impl MetadataExecutor {
    fn new(f: &Fixture) -> Self {
        Self {
            compiler: f.options.compiler.clone(),
            expected: VecDeque::new(),
        }
    }
    fn list(
        &mut self,
        directory: &Path,
        patterns: &[&str],
        guest: bool,
        metadata: &serde_json::Value,
    ) {
        self.expected.push_back(Expected {
            directory: directory.into(),
            guest,
            timeout: Duration::from_secs(600),
            args: [
                "list",
                "-mod=readonly",
                "-e",
                "-json=ImportPath,Dir,GoFiles,TestGoFiles,XTestGoFiles,Error",
                "-tags",
                "disable_grpc_modules,gomad_wasm_slim,sqlite3_dotlk,test_dep",
            ]
            .into_iter()
            .chain(patterns.iter().copied())
            .map(str::to_owned)
            .collect(),
            completed: Completed {
                stdout: serde_json::to_vec(metadata).unwrap(),
                ..Completed::default()
            },
        });
    }
    fn tests(&mut self, directory: &Path, target: &str, output: &str) {
        self.expected.push_back(Expected {
            directory: directory.into(),
            guest: false,
            timeout: Duration::from_mins(30),
            args: [
                "test",
                "-mod=readonly",
                "-count=1",
                "-tags",
                "disable_grpc_modules,gomad_wasm_slim,sqlite3_dotlk,test_dep",
                "-list",
                "^Test",
                target,
            ]
            .map(str::to_owned)
            .to_vec(),
            completed: Completed {
                stdout: output.as_bytes().to_vec(),
                ..Completed::default()
            },
        });
    }
    fn finished(&self) {
        assert!(
            self.expected.is_empty(),
            "expected metadata command was not issued"
        );
    }
}
impl Executor for MetadataExecutor {
    fn exec(&mut self, _: &Context, command: &Command) -> Completed {
        let next = self
            .expected
            .pop_front()
            .expect("unexpected metadata command");
        assert_eq!(command.path, self.compiler);
        assert_eq!(command.dir, next.directory);
        assert_eq!(command.args, next.args);
        assert_eq!(command.timeout, next.timeout);
        let mut environment = BTreeMap::from([
            ("HOME".into(), "/operator-home".into()),
            ("GOFLAGS".into(), String::new()),
            ("GOWORK".into(), "off".into()),
            ("GOTOOLCHAIN".into(), "local".into()),
            ("GOENV".into(), "off".into()),
            ("CGO_ENABLED".into(), "0".into()),
        ]);
        if next.guest {
            environment.extend([
                ("GOOS".into(), "wasip1".into()),
                ("GOARCH".into(), "wasm".into()),
                ("GOEXPERIMENT".into(), "nogreenteagc".into()),
            ]);
        }
        assert_eq!(command.env, environment);
        next.completed
    }
}
fn metadata(
    import: &str,
    directory: &Path,
    go: &[&str],
    tests: &[&str],
    external: &[&str],
) -> serde_json::Value {
    serde_json::json!({"ImportPath":import,"Dir":directory,"GoFiles":go,"TestGoFiles":tests,"XTestGoFiles":external})
}

// Original roots: parsed inventory/list reconciliation and compiler environment.
#[test]
fn native_enumeration_excludes_comment_and_raw_literal_phantom_tests() {
    let f = Fixture::new();
    f.write("tests/alpha/alpha_test.go", "package alpha\nimport \"testing\"\n// func TestComment(t *testing.T) {}\nconst text = `func TestRaw(t *testing.T) {}`\nfunc TestAlpha(t *testing.T) {}\nfunc Testlower(t *testing.T) {}\nfunc TestMain(m *testing.M) {}\n");
    f.write(
        "tests/alpha/external_test.go",
        "package alpha_test\nfunc TestExternal(t *testing.T) {}\n",
    );
    let dir = f.options.checkout.join("tests/alpha");
    let listed = metadata(
        "example.com/alpha",
        &dir,
        &[],
        &["alpha_test.go"],
        &["external_test.go"],
    );
    let mut executor = MetadataExecutor::new(&f);
    executor.list(&f.options.checkout, &["./tests/..."], false, &listed);
    executor.list(&f.options.checkout, &["./tests/..."], true, &listed);
    executor.tests(
        &f.options.checkout,
        "./tests/alpha",
        "TestExternal\nTestAlpha\nok example.com/alpha 0.1s\n",
    );
    assert_eq!(
        enumerate(&mut executor, &Fixture::context(), &f.options).unwrap(),
        vec![Package {
            import_path: "example.com/alpha".into(),
            pattern: "./tests/alpha".into(),
            module_directory: f.options.checkout.display().to_string(),
            target: "./tests/alpha".into(),
            tests: vec!["TestAlpha".into(), "TestExternal".into()],
            units: BTreeMap::from([
                ("TestAlpha".into(), vec!["TestAlpha".into()]),
                ("TestExternal".into(), vec!["TestExternal".into()])
            ]),
            top_level_tests: vec!["TestAlpha".into(), "TestExternal".into()],
            guest_excluded: BTreeMap::new(),
            list_count: 2,
            parsed_count: 2,
        }]
    );
    executor.finished();
}

// Original root: enumeration mismatch refuses work rather than accepting extra names.
#[test]
fn native_enumeration_names_generated_elsewhere_in_mismatch_error() {
    let f = Fixture::new();
    f.write(
        "tests/alpha/alpha_test.go",
        "package alpha; func TestAlpha(t *testing.T) {}",
    );
    let listed = metadata(
        "example.com/alpha",
        &f.options.checkout.join("tests/alpha"),
        &[],
        &["alpha_test.go"],
        &[],
    );
    let mut executor = MetadataExecutor::new(&f);
    executor.list(&f.options.checkout, &["./tests/..."], false, &listed);
    executor.list(&f.options.checkout, &["./tests/..."], true, &listed);
    executor.tests(
        &f.options.checkout,
        "./tests/alpha",
        "TestAlpha\nTestGeneratedElsewhere\n",
    );
    let error = enumerate(&mut executor, &Fixture::context(), &f.options).unwrap_err();
    assert_eq!(
        error,
        "enumeration mismatch: ./tests/alpha: parsed [\"TestAlpha\"], go test -list [\"TestAlpha\", \"TestGeneratedElsewhere\"]"
    );
    executor.finished();
}

// Original root: TestSuiteInventoryGuestExclusionsIncludeWrapperAndMethod.
#[test]
fn native_enumeration_checks_wrapper_method_and_non_test_go_file_guest_membership() {
    for (wrapper_excluded, method_file) in [
        (true, "method_test.go"),
        (false, "method_test.go"),
        (false, "method.go"),
    ] {
        let f = Fixture::new();
        f.write("tests/alpha/wrapper_test.go", "package alpha; import ps \"go.temporal.io/server/common/testing/parallelsuite\"; type S struct{}; func TestWrapper(t *testing.T) { ps.Run(t, &S{}) }");
        f.write(
            &format!("tests/alpha/{method_file}"),
            "package alpha; func (*S) TestMethod() {}",
        );
        let dir = f.options.checkout.join("tests/alpha");
        let (host_go, host_test) = if method_file.ends_with("_test.go") {
            (vec![], vec!["wrapper_test.go", method_file])
        } else {
            (vec![method_file], vec!["wrapper_test.go"])
        };
        let excluded = if wrapper_excluded {
            "wrapper_test.go"
        } else {
            method_file
        };
        let guest_go: Vec<_> = host_go
            .iter()
            .copied()
            .filter(|name| *name != excluded)
            .collect();
        let guest_test: Vec<_> = host_test
            .iter()
            .copied()
            .filter(|name| *name != excluded)
            .collect();
        let mut executor = MetadataExecutor::new(&f);
        executor.list(
            &f.options.checkout,
            &["./tests/..."],
            false,
            &metadata("example.com/alpha", &dir, &host_go, &host_test, &[]),
        );
        executor.list(
            &f.options.checkout,
            &["./tests/..."],
            true,
            &metadata("example.com/alpha", &dir, &guest_go, &guest_test, &[]),
        );
        executor.tests(&f.options.checkout, "./tests/alpha", "TestWrapper\n");
        let packages = enumerate(&mut executor, &Fixture::context(), &f.options).unwrap();
        assert_eq!(packages[0].top_level_tests, ["TestWrapper"]);
        assert_eq!(packages[0].tests, ["TestWrapper/TestMethod"]);
        assert_eq!(
            packages[0].guest_excluded,
            BTreeMap::from([("TestWrapper/TestMethod".into(), excluded.into())])
        );
        executor.finished();
    }
}

// Original root: TestModuleSelectionsHonorPackageFilters.
#[test]
fn discovered_modules_honor_exact_recursive_and_mixed_filters() {
    let f = Fixture::new();
    for path in [
        "tests/mixedbrain/go.mod",
        "tests/mixedbrain/deeper/go.mod",
        "tests/vendor/ignored/go.mod",
        "tests/.hidden/go.mod",
    ] {
        f.write(path, "module example.com/fixture\n");
    }
    let root = &f.options.checkout;
    let nested = root.join("tests/mixedbrain");
    let deeper = nested.join("deeper");
    let modules = module_roots(root).unwrap();
    assert_eq!(modules, vec![root.clone(), nested.clone(), deeper.clone()]);
    for (patterns, expected) in [
        (
            vec!["./tests"],
            BTreeMap::from([(root.clone(), vec!["./tests".into()])]),
        ),
        (
            vec!["./tests/..."],
            BTreeMap::from([
                (root.clone(), vec!["./tests/...".into()]),
                (nested.clone(), vec!["./...".into()]),
                (deeper.clone(), vec!["./...".into()]),
            ]),
        ),
        (
            vec!["./tests/mixedbrain"],
            BTreeMap::from([(nested.clone(), vec!["./.".into()])]),
        ),
        (
            vec!["./tests/mixedbrain/..."],
            BTreeMap::from([
                (nested.clone(), vec!["./...".into()]),
                (deeper, vec!["./...".into()]),
            ]),
        ),
        (
            vec![
                "./tests/mixedbrain/sub",
                "./tests",
                "./tests/mixedbrain/sub",
            ],
            BTreeMap::from([
                (root.clone(), vec!["./tests".into()]),
                (nested, vec!["./sub".into()]),
            ]),
        ),
    ] {
        let patterns = patterns.into_iter().map(str::to_owned).collect::<Vec<_>>();
        assert_eq!(
            select_modules(root, &patterns, &modules).unwrap(),
            expected,
            "{patterns:?}"
        );
    }
}

// Original roots: TestEnumerationIncludesNestedModules and nested exact targets.
#[test]
fn native_enumeration_uses_nested_module_cwd_and_dot_target() {
    for recursive in [false, true] {
        let mut f = Fixture::new();
        f.write(
            "tests/parent_test.go",
            "package parent; func TestParent(t *testing.T) {}",
        );
        f.write("tests/mixedbrain/go.mod", "module example.com/nested\n");
        f.write(
            "tests/mixedbrain/nested_test.go",
            "package nested; func TestNested(t *testing.T) {}",
        );
        f.options.packages = vec![
            if recursive {
                "./tests/..."
            } else {
                "./tests/mixedbrain"
            }
            .into(),
        ];
        let root = &f.options.checkout;
        let nested = root.join("tests/mixedbrain");
        let mut executor = MetadataExecutor::new(&f);
        if recursive {
            let listed = metadata(
                "example.com/parent",
                &root.join("tests"),
                &[],
                &["parent_test.go"],
                &[],
            );
            executor.list(root, &["./tests/..."], false, &listed);
            executor.list(root, &["./tests/..."], true, &listed);
            executor.tests(root, "./tests", "TestParent\n");
        }
        let listed = metadata("example.com/nested", &nested, &[], &["nested_test.go"], &[]);
        let selected_pattern = if recursive { "./..." } else { "./." };
        executor.list(&nested, &[selected_pattern], false, &listed);
        executor.list(&nested, &[selected_pattern], true, &listed);
        executor.tests(&nested, "./.", "TestNested\n");
        let packages = enumerate(&mut executor, &Fixture::context(), &f.options).unwrap();
        assert_eq!(
            packages
                .iter()
                .map(|p| p.pattern.as_str())
                .collect::<Vec<_>>(),
            if recursive {
                vec!["./tests", "./tests/mixedbrain"]
            } else {
                vec!["./tests/mixedbrain"]
            }
        );
        let package = packages
            .iter()
            .find(|p| p.import_path == "example.com/nested")
            .unwrap();
        assert_eq!(package.module_directory, nested.display().to_string());
        assert_eq!(package.target, "./.");
        assert_eq!(package.top_level_tests, ["TestNested"]);
        assert_eq!(package.tests, ["TestNested"]);
        executor.finished();
    }
}
