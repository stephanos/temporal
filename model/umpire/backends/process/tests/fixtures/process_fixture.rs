//! Native child programs for the process ownership controls.
#![forbid(unsafe_code)]
#![allow(clippy::print_stdout, clippy::print_stderr)]
use std::{
    env, fs,
    io::{self, Read, Write},
    os::unix::process::CommandExt,
    path::Path,
    process::{self, Command, Stdio},
    sync::atomic::AtomicBool,
    thread,
    time::Duration,
};
use umpire_backend_process::{Limits, run_process_cancelled};

fn sleep() {
    thread::sleep(Duration::from_secs(60));
}

fn parent_pid() -> io::Result<String> {
    let status = fs::read_to_string("/proc/self/status")?;
    Ok(status.lines().find_map(|line| line.strip_prefix("PPid:")).unwrap_or("").trim().into())
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args: Vec<_> = env::args().collect();
    match args.get(1).map(String::as_str).unwrap_or("") {
        "started" => {
            println!("started");
            io::stdout().flush()?;
            sleep();
        }
        "sleep" => sleep(),
        "tree" => {
            let mut input = String::new();
            io::stdin().read_to_string(&mut input)?;
            assert!(input.is_empty());
            let child = Command::new("setsid")
                .arg(env::current_exe()?)
                .arg("sleep")
                .stdin(Stdio::null())
                .spawn()?;
            fs::write(&args[2], format!("{} {}", process::id(), child.id()))?;
            fs::write(&args[3], parent_pid()?)?;
            sleep();
        }
        "owner" => {
            let directory = Path::new(&args[2]);
            let mut target = Command::new(env::current_exe()?);
            target.args(["tree", &args[3], &args[4]]);
            let _ = run_process_cancelled(
                &mut target,
                directory,
                "owner",
                Limits::default(),
                None,
                &AtomicBool::new(false),
            );
        }
        "policy" => {
            println!("{}", env::var("JAVA_TOOL_OPTIONS").unwrap_or_else(|_| "None".into()));
            let parent = parent_pid()?;
            let output = Command::new("ps").args(["-o", "pgid=", "-p", &parent]).output()?;
            println!(
                "{}",
                if String::from_utf8(output.stdout)?.trim() == parent { "True" } else { "False" }
            );
        }
        "symlink" => {
            fs::remove_file("capture.stdout")?;
            std::os::unix::fs::symlink(&args[2], "capture.stdout")?;
            println!("captured");
        }
        "fifo" => {
            fs::remove_file("capture.stdout")?;
            assert!(Command::new("mkfifo").arg("capture.stdout").status()?.success());
            println!("captured");
        }
        "combined" => {
            io::stdout().write_all(b"abcd")?;
            io::stderr().write_all(b"efgh")?;
        }
        "utf8" => io::stdout().write_all(&[255])?,
        "echo" => io::stdout().write_all(b"native\n")?,
        "descendant-attempt" => {
            match Command::new(env::current_exe()?).arg("sleep").process_group(0).spawn() {
                Err(error) if error.kind() == io::ErrorKind::WouldBlock => {
                    println!("descendant creation refused");
                    process::exit(75);
                }
                Err(error) => return Err(error.into()),
                Ok(mut child) => {
                    println!("escaped {}", child.id());
                    child.kill()?;
                    child.wait()?;
                    process::exit(1);
                }
            }
        }
        mode => return Err(format!("unknown fixture mode: {mode}").into()),
    }
    Ok(())
}
