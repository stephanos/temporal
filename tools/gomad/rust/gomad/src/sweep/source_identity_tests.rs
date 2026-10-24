use super::*;
use std::sync::atomic::{AtomicU64, Ordering};
struct GitFixture {
    root: PathBuf,
    options: Options,
}
impl GitFixture {
    fn new() -> Self {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        let root = std::env::temp_dir().join(format!(
            "gomad-sweep-source-{}-{}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ));
        let checkout = root.join("checkout");
        directory(&checkout).unwrap();
        directory(&root.join("work")).unwrap();
        directory(&checkout.join("tests/alpha")).unwrap();
        fs::write(
            checkout.join("tests/alpha/alpha_test.go"),
            "package alpha\n",
        )
        .unwrap();
        let options = Options {
            checkout: checkout.clone(),
            gomad: root.join("gomad"),
            compiler: root.join("compiler"),
            helper: root.join("helper"),
            runtime_root: root.join("runtime"),
            config_source: "throwaway".into(),
            packages: vec!["./tests/...".into()],
            max_tests: 0,
            seed: 0,
            per_test_timeout: Duration::from_secs(1),
            prepare_timeout: Duration::from_secs(1),
            overall_budget: Duration::from_secs(60),
            retain_bytes: 0,
            out: checkout.join("space [abc]*?"),
            work: root.join("work"),
            artifacts: root.join("artifacts"),
            environ: BTreeMap::from([
                ("PATH".into(), "/usr/bin:/bin".into()),
                ("GIT_CONFIG_NOSYSTEM".into(), "1".into()),
                ("GIT_CONFIG_GLOBAL".into(), "/dev/null".into()),
            ]),
        };
        let fixture = Self { root, options };
        fixture.git(&["init", "-q"]);
        fixture.git(&["add", "--", "."]);
        fixture.git(&[
            "-c",
            "user.name=Fixture",
            "-c",
            "user.email=fixture@example.invalid",
            "commit",
            "-q",
            "-m",
            "fixture",
        ]);
        fixture
    }
    fn git(&self, args: &[&str]) {
        let status = crate::fork_order::status(
            std::process::Command::new("git")
                .args(args)
                .current_dir(&self.options.checkout)
                .env_clear()
                .envs(&self.options.environ),
        )
        .unwrap();
        assert!(status.success());
    }
    fn identity(&self) -> Result<(String, String), String> {
        source_identity(
            &mut super::super::process::Host,
            &Context::new(Instant::now() + Duration::from_secs(60)),
            &self.options,
        )
    }
}
impl Drop for GitFixture {
    fn drop(&mut self) {
        fs::remove_dir_all(&self.root).unwrap();
    }
}
#[test]
fn real_git_literal_outputs_do_not_hide_source_or_similarly_named_directories() {
    let mut fixture = GitFixture::new();
    let identity = fixture.identity().unwrap();
    directory(&fixture.options.out).unwrap();
    for name in ["sweep.jsonl", "sweep.json", "sweep.json.tmp"] {
        fs::write(fixture.options.out.join(name), "output").unwrap();
    }
    assert_eq!(fixture.identity().unwrap(), identity);
    fs::write(
        fixture.options.checkout.join("tests/alpha/alpha_test.go"),
        "package alpha\n// modified\n",
    )
    .unwrap();
    assert!(
        fixture
            .identity()
            .unwrap_err()
            .contains("clean frozen checkout")
    );
    fs::write(
        fixture.options.checkout.join("tests/alpha/alpha_test.go"),
        "package alpha\n",
    )
    .unwrap();
    let neighbor = fixture.options.checkout.join("space a");
    directory(&neighbor).unwrap();
    fs::write(neighbor.join("sweep.json"), "untracked").unwrap();
    assert!(fixture.identity().is_err());
    fs::remove_dir_all(neighbor).unwrap();
    fs::remove_dir_all(&fixture.options.out).unwrap();
    fixture.options.out = fixture.options.checkout.join("tests/alpha");
    assert_eq!(fixture.identity().unwrap(), identity);
    fs::write(fixture.options.out.join("new_test.go"), "package alpha\n").unwrap();
    assert!(fixture.identity().is_err());
}
#[test]
fn status_failure_stops_identity_and_changed_commits_change_candidate() {
    let fixture = GitFixture::new();
    let before = fixture.identity().unwrap();
    fs::write(
        fixture.options.checkout.join("tests/alpha/alpha_test.go"),
        "package alpha\n// committed\n",
    )
    .unwrap();
    fixture.git(&["add", "."]);
    fixture.git(&[
        "-c",
        "user.name=Fixture",
        "-c",
        "user.email=fixture@example.invalid",
        "commit",
        "-q",
        "-m",
        "changed",
    ]);
    assert_ne!(fixture.identity().unwrap(), before);
    fs::rename(
        fixture.options.checkout.join(".git"),
        fixture.root.join("git-hidden"),
    )
    .unwrap();
    assert!(fixture.identity().unwrap_err().contains("git: exit"));
}
