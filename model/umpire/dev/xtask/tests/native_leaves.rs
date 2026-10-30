use std::{path::Path, process::Command};

#[test]
fn capability_controls_enforce_core_and_unsafe_policies() -> xtask::Result<()> {
    let root = umpire_guest_driver::context::ToolRoot::local()?;
    assert_eq!(xtask::capability_controls::run(root.path())?, 0);
    Ok(())
}

#[test]
fn public_timing_front_door_preserves_status_diagnostics_and_cleanup() -> std::io::Result<()> {
    assert_eq!(xtask::timing_controls::run(Path::new(env!("CARGO_BIN_EXE_xtask")))?, 0);
    Ok(())
}

#[test]
fn public_protobuf_controls_reach_the_native_implementation() -> std::io::Result<()> {
    let mut command = Command::new(env!("CARGO_BIN_EXE_xtask"));
    command.arg("proto-controls");
    let output = xtask::gate::scheduler::execute(&command, false, false)?;
    assert!(output.status.success(), "{output:?}");
    assert!(String::from_utf8_lossy(&output.stdout).contains("native protobuf controls passed"));
    Ok(())
}
