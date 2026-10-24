extern crate umpire_tla as owner;
use owner::process::*;
use std::{fs, path::Path, process::Command, sync::atomic::AtomicBool, time::Duration};
fn main() -> Result<(), Refusal> {
    let root = Path::new("/Users/stephan/Workspace/skunkworks/umpire/temporal");
    let scratch = Scratch::new(root)?;
    let defaults = Limits::default();
    println!("defaults={:?},{},{},{}", defaults.timeout, defaults.output_bytes, defaults.dump_bytes, defaults.heap_mb);
    println!("refusal={}", Refusal("control".into()));
    for (label, script) in [("success", "printf out; printf err >&2"), ("failure", "printf refused; exit 7"), ("signal", "kill -TERM $$")] {
        let mut command = Command::new("/bin/sh");
        command.args(["-c", script]);
        let (status, output) = run_process(&mut command, &scratch.0, label, defaults, None)?;
        println!("{label}:status={:?},output={output:?}", status.code());
    }
    for (label, script, limits, cancelled) in [
        ("cancel", "exit 0", defaults, true),
        ("timeout", "printf started; sleep 10", Limits { timeout: Duration::from_secs(1), ..defaults }, false),
        ("utf8", "printf '\\377'", defaults, false),
    ] {
        let mut command = Command::new("/bin/sh");
        command.args(["-c", script]);
        let failure = run_process_cancelled(&mut command, &scratch.0, label, limits, None, &AtomicBool::new(cancelled))
            .err().ok_or_else(|| Refusal(format!("{label} unexpectedly passed")))?;
        println!("{label}:refusal={},output={:?}", failure.refusal, failure.output);
    }
    let text = scratch.0.join("text");
    fs::write(&text, "hello")?;
    println!("bounded: {:?}", bounded_text(&text, 5)?);
    let error = bounded_text(&text, 4).err().ok_or_else(|| Refusal("bounded read admitted".into()))?;
    assert_eq!(error.0, format!("file exceeds bounded read limit: {}", text.display()));
    println!("bounded-overflow: exact diagnostic with structurally checked scratch path");
    let error = Scratch::new_in(root, "../escape").err().ok_or_else(|| Refusal("scratch escape admitted".into()))?;
    println!("scratch: {error}");
    let command = java_command(Path::new("java"));
    println!("java-env: {:?}", command.get_envs().collect::<Vec<_>>());
    let path = scratch.0.clone();
    scratch.finish::<_, Refusal>(Ok(()))?;
    assert!(!path.exists());
    println!("cleanup: absent");
    Ok(())
}
