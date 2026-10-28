use super::*;
use serde_json::{Value, json};
use std::os::unix::fs::PermissionsExt;

fn stream(events: &[Value]) -> io::Result<Vec<u8>> {
    let mut bytes = Vec::new();
    for event in events {
        serde_json::to_writer(&mut bytes, event)?;
        bytes.push(b'\n');
    }
    Ok(bytes)
}

#[test]
fn observer_child() -> io::Result<()> {
    let Ok(mode) = std::env::var("GO_OBSERVER_CONTROL") else { return Ok(()) };
    if mode.starts_with("real") {
        return real_child(&mode);
    }
    let fixture = tempfile::tempdir()?;
    let package = "control";
    let empty = mode == "zero";
    let listing = if empty {
        vec![
            json!({"Action":"start","Package":package}),
            json!({"Action":"skip","Package":package}),
        ]
    } else {
        vec![
            json!({"Action":"start","Package":package}),
            json!({"Action":"output","Package":package,"Output":"TestRuns\nTestSkips\n"}),
            json!({"Action":"pass","Package":package}),
        ]
    };
    let failed = mode == "failure" || mode == "contradiction";
    let execution = if empty {
        listing.clone()
    } else {
        vec![
            json!({"Action":"start","Package":package}),
            json!({"Action":"run","Package":package,"Test":"TestRuns"}),
            json!({"Action":"output","Package":package,"Test":"TestRuns","Output":"real Go diagnostic\n"}),
            json!({"Action":if failed {"fail"} else {"pass"},"Package":package,"Test":"TestRuns"}),
            json!({"Action":"run","Package":package,"Test":"TestSkips"}),
            json!({"Action":"skip","Package":package,"Test":"TestSkips"}),
            json!({"Action":if failed {"fail"} else {"pass"},"Package":package}),
        ]
    };
    fs::write(fixture.path().join("listing"), stream(&listing)?)?;
    fs::write(
        fixture.path().join("execution"),
        if mode == "malformed" { b"truncated Go JSON".to_vec() } else { stream(&execution)? },
    )?;
    let mise = fixture.path().join("mise");
    fs::write(
        &mise,
        br#"#!/bin/sh
case " $* " in
 *' env -json GOFLAGS ') printf '{"GOFLAGS":"-p=1"}\n'; exit 0 ;;
 *' -list . '*) /bin/cat "$GO_CONTROL_ROOT/listing"; exit "$GO_CONTROL_LIST_EXIT" ;;
esac
/bin/cat "$GO_CONTROL_ROOT/execution"
printf 'real stderr diagnostic' >&2
exit "$GO_CONTROL_EXIT"
"#,
    )?;
    fs::set_permissions(&mise, fs::Permissions::from_mode(0o700))?;
    let actual_exit = if ["failure", "malformed"].contains(&mode.as_str()) { 37 } else { 0 };
    let list_exit = if mode == "list-failure" { 43 } else { 0 };
    let expected_exit = if list_exit != 0 { list_exit } else { actual_exit };
    let result = super::super::scheduler::scheduled("go-host", || {
        let mut command = Command::new(&mise);
        command
            .args(["exec", "--", "go", "-C"])
            .arg(fixture.path())
            .args(["test", "-tags", "test_dep", "-count=1", "./..."])
            .current_dir(fixture.path())
            .env("GOFLAGS", "-p=1")
            .env("CGO_ENABLED", "0")
            .env("GO_CONTROL_ROOT", fixture.path())
            .env("GO_CONTROL_EXIT", actual_exit.to_string())
            .env("GO_CONTROL_LIST_EXIT", list_exit.to_string());
        let output =
            observe(&command)?.ok_or_else(|| io::Error::other("missing Go test observer"))?;
        assert_eq!(output.status.code(), Some(expected_exit));
        if list_exit == 0 {
            assert_eq!(output.stderr, b"real stderr diagnostic");
            if mode == "malformed" {
                assert_eq!(output.stdout, b"truncated Go JSON");
            } else if empty {
                assert_eq!(output.stdout, b"");
            } else {
                assert_eq!(output.stdout, b"real Go diagnostic\n");
            }
        }
        Ok(output.status.code().unwrap_or(1))
    });
    if mode == "contradiction" {
        assert!(result.is_err(), "successful process accepted failed Go metadata");
    } else {
        assert_eq!(result?, expected_exit);
    }
    Ok(())
}

fn real_child(mode: &str) -> io::Result<()> {
    let fixture = tempfile::tempdir()?;
    fs::write(fixture.path().join("go.mod"), "module example.com/count-control\n\ngo 1.27.0\n")?;
    fs::write(
        fixture.path().join("control_test.go"),
        r#"package control
import ("fmt"; "testing")
func TestRuns(t *testing.T) { t.Run("nested", func(t *testing.T) { t.Log("nested diagnostic") }) }
func TestSkips(t *testing.T) { t.Skip("explicit skip") }
func FuzzDecode(f *testing.F) { f.Add("seed"); f.Fuzz(func(t *testing.T, value string) {}) }
func Example() { fmt.Println("example"); // Output: example
}
func BenchmarkUnused(b *testing.B) {}
"#,
    )?;
    fs::create_dir(fixture.path().join("other"))?;
    fs::write(
        fixture.path().join("other/control_test.go"),
        "package other\nimport \"testing\"\nfunc TestRuns(t *testing.T) {}\n",
    )?;
    let goenv = fixture.path().join("go.env");
    fs::write(
        &goenv,
        if mode == "real-hidden-selector" {
            "GOFLAGS=-p=1 -run=TestRuns\n"
        } else {
            "GOFLAGS=-p=1\n"
        },
    )?;
    let result = super::super::scheduler::scheduled("go-host", || {
        let mut command = Command::new("mise");
        command
            .args(["exec", "--", "go", "-C"])
            .arg(fixture.path())
            .args(["test", "-tags", "test_dep", "-count=1", "./..."])
            .current_dir(Path::new(env!("CARGO_MANIFEST_DIR")).join("../.."))
            .env("GOENV", &goenv)
            .env_remove("GOFLAGS")
            .env("CGO_ENABLED", "0")
            .env("GOMAXPROCS", "1")
            .env("MISE_AUTO_INSTALL", "false")
            .env("MISE_EXEC_AUTO_INSTALL", "false");
        let output =
            observe(&command)?.ok_or_else(|| io::Error::other("missing real Go observer"))?;
        assert!(
            output.status.success(),
            "{}{}",
            String::from_utf8_lossy(&output.stdout),
            String::from_utf8_lossy(&output.stderr)
        );
        assert!(!output.stdout.starts_with(b"{\"Time\""));
        Ok(0)
    });
    if mode == "real-hidden-selector" {
        assert!(result.is_err(), "Go config selector bypassed count admission");
    } else {
        assert_eq!(result?, 0);
    }
    Ok(())
}

fn child(mode: &str) -> io::Result<(tempfile::TempDir, Value)> {
    let directory = tempfile::tempdir()?;
    let output = Command::new(std::env::current_exe()?)
        .args(["--exact", "gate::work_counts::go_tests::observer_child", "--nocapture"])
        .env("GO_OBSERVER_CONTROL", mode)
        .env("UMPIRE_GATE_RECEIPTS", directory.path())
        .env("UMPIRE_GATE_SCOPE", "framework")
        .env("UMPIRE_GATE_STAGE", "go-host")
        .env_remove("UMPIRE_GATE_TEST_CARRIER")
        .output()?;
    assert!(
        output.status.success(),
        "{}{}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr)
    );
    let receipt =
        serde_json::from_slice(&fs::read(directory.path().join("framework/go-host.json"))?)?;
    Ok((directory, receipt))
}

#[test]
fn go_observer_publishes_only_complete_actual_counts() -> io::Result<()> {
    for mode in ["success", "failure", "zero"] {
        let (_directory, receipt) = child(mode)?;
        assert_eq!(receipt["state"], if mode == "failure" { "failed" } else { "success" });
        assert_eq!(receipt["exit"], if mode == "failure" { 37 } else { 0 });
        assert_eq!(receipt["counts"]["discovered"], if mode == "zero" { 0 } else { 2 });
        assert_eq!(receipt["counts"]["executed"], u64::from(mode != "zero"));
        assert_eq!(receipt["counts"]["ignored"], 0);
        assert_eq!(receipt["counts"]["skipped"], u64::from(mode != "zero"));
        assert!(receipt["counts"]["tlc_launches"].is_null());
        assert!(receipt["count_error"].is_null());
    }
    Ok(())
}

#[test]
fn go_observer_preserves_failed_processes_and_refuses_incomplete_or_contradictory_counts()
-> io::Result<()> {
    for (mode, exit) in [("malformed", 37), ("list-failure", 43), ("contradiction", 1)] {
        let (_directory, receipt) = child(mode)?;
        assert_eq!(receipt["state"], "failed");
        assert_eq!(receipt["exit"], exit);
        assert!(receipt["counts"].is_null());
        assert!(receipt["count_error"].is_string());
    }
    Ok(())
}

#[test]
fn real_go_producer_counts_roots_and_resolves_go_environment_configuration() -> io::Result<()> {
    let (_directory, receipt) = child("real")?;
    assert_eq!(receipt["state"], "success");
    assert_eq!(receipt["counts"]["discovered"], 5);
    assert_eq!(receipt["counts"]["executed"], 4);
    assert_eq!(receipt["counts"]["skipped"], 1);
    assert_eq!(receipt["counts"]["ignored"], 0);
    let (_directory, refused) = child("real-hidden-selector")?;
    assert_eq!(refused["state"], "failed");
    assert!(refused["counts"].is_null());
    assert!(refused["count_error"].is_string());
    Ok(())
}
