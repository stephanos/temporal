//! Rebuild the real catalog and generated metadata against an added upstream field.

use std::{fs, path::Path, process::Command};

use prost::Message;
use umpire_runtime::prost_types::{
    FieldDescriptorProto, FileDescriptorSet, field_descriptor_proto,
};

fn git(root: &Path, arguments: &[&str]) -> umpire_testing::TestResult {
    let output = Command::new("git").current_dir(root).args(arguments).output()?;
    if !output.status.success() {
        return Err(String::from_utf8_lossy(&output.stderr).into_owned().into());
    }
    Ok(())
}

fn add_field(root: &Path, workspace: &Path, original: &str) -> umpire_testing::TestResult {
    let mut descriptor =
        FileDescriptorSet::decode(fs::read(root.join("proto/api.binpb"))?.as_slice())?;
    let response = descriptor
        .file
        .iter_mut()
        .find_map(|file| {
            (file.package.as_deref() == Some("temporal.api.workflowservice.v1"))
                .then(|| {
                    file.message_type.iter_mut().find(|message| {
                        message.name.as_deref() == Some("RequestCancelActivityExecutionResponse")
                    })
                })
                .flatten()
        })
        .ok_or("missing actual cancel response descriptor")?;
    response.field.push(FieldDescriptorProto {
        name: Some("coverage_control".into()),
        number: Some(536_870_000),
        label: Some(field_descriptor_proto::Label::Optional as i32),
        r#type: Some(field_descriptor_proto::Type::String as i32),
        ..Default::default()
    });
    let empty = "pub struct RequestCancelActivityExecutionResponse {}";
    if original.matches(empty).count() != 1 {
        return Err("cancel catalog control requires its actual empty response declaration".into());
    }
    fs::write(
        workspace.join("model/temporal/feature/activity/api.rs"),
        original.replace(
            empty,
            "pub struct RequestCancelActivityExecutionResponse { coverage_control: Text }",
        ),
    )?;
    fs::write(workspace.join("proto/api.binpb"), descriptor.encode_to_vec())?;
    Ok(())
}

#[test]
#[ignore = "rebuilds the committed source in an isolated worktree; run after the coverage checkpoint"]
fn added_catalog_field_fails_the_real_coverage_golden() -> umpire_testing::TestResult {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../..").canonicalize()?;
    let lane = format!(".worktrees/fn2312-coverage-control-{}", std::process::id());
    let branch = format!("work/fn2312-coverage-control-{}", std::process::id());
    git(&root, &["worktree", "add", "--quiet", "-b", &branch, &lane, "HEAD"])?;
    let workspace = root.join(&lane);
    let catalog = workspace.join("model/temporal/feature/activity/api.rs");
    let original = fs::read_to_string(&catalog)?;
    let result = (|| -> umpire_testing::TestResult {
        let golden = "model/temporal/feature/activity/expected/realization.snap.txt";
        let before = fs::read(workspace.join(golden))?;
        if before != fs::read(root.join(golden))? {
            return Err("control requires the current committed coverage checkpoint".into());
        }
        add_field(&root, &workspace, &original)?;
        let output = Command::new("mise")
            .current_dir(workspace.join("model"))
            .args([
                "exec",
                "--",
                "cargo",
                "test",
                "-p",
                "umpire-temporal-tests",
                "--test",
                "fixtures",
                "--",
                "--exact",
                "fixtures",
                "--nocapture",
            ])
            .env("INSTA_UPDATE", "new")
            .env("INSTA_COLOR", "never")
            .env_remove("UMPIRE_BACKENDS")
            .output()?;
        let log = format!(
            "{}\n{}",
            String::from_utf8_lossy(&output.stdout),
            String::from_utf8_lossy(&output.stderr)
        );
        fs::write(root.join(".flow/tmp/realization-added-field-control.log"), &log)?;
        if output.status.success() || !log.contains("snapshot assertion for 'realization' failed") {
            return Err(format!(
                "added-field control did not reach the expected golden refusal: {}\n{log}",
                output.status
            )
            .into());
        }
        if fs::read(workspace.join(golden))? != before {
            return Err("added-field control changed the coverage baseline".into());
        }
        let pending = fs::read_to_string(
            workspace.join("model/temporal/feature/activity/expected/realization.snap.new.txt"),
        )?;
        let expected = String::from_utf8(before)?.replacen(
            "message: temporal.api.workflowservice.v1.RequestCancelActivityExecutionResponse\n  (no fields)",
            "message: temporal.api.workflowservice.v1.RequestCancelActivityExecutionResponse\n  unsupported coverage_control",
            1,
        );
        fs::write(root.join(".flow/tmp/realization-added-field-control.txt"), &pending)?;
        if pending != expected || !pending.contains("unsupported coverage_control") {
            return Err("added-field snapshot has an unexpected coverage delta".into());
        }
        Ok(())
    })();
    fs::write(&catalog, original)?;
    for file in ["realization.snap.new", "realization.snap.new.txt"] {
        let path = workspace.join("model/temporal/feature/activity/expected").join(file);
        if path.is_file() {
            fs::remove_file(path)?;
        }
    }
    git(&root, &["worktree", "remove", &lane])?;
    git(&root, &["branch", "-d", &branch])?;
    result
}
