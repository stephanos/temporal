use std::{path::Path, process::Command};
use umpire_backend_process::{Limits, Refusal, Scratch, run_process};
fn main() -> Result<(), Refusal> {
    let root = std::env::args_os().nth(1).ok_or_else(|| Refusal("workspace required".into()))?;
    let scratch = Scratch::new(Path::new(&root))?;
    let mut command = Command::new("/bin/sh");
    command.args(["-c", "printf cold-consumer"]).env_clear();
    let (status, text) = run_process(&mut command, &scratch.0, "cold", Limits::default(), None)?;
    assert!(status.success());
    assert_eq!(text, "cold-consumer");
    scratch.finish::<_, Refusal>(Ok(()))?;
    println!("cold direct dependency consumer passed without Python, Cargo or sibling executable at runtime");
    Ok(())
}
