//! Native child programs for the process ownership controls.
#![deny(unsafe_code)]
#![allow(clippy::print_stdout, clippy::print_stderr)]
#[path = "../../src/syscall.rs"]
#[allow(dead_code)]
mod syscall;
use umpire_backend_process::DescendantPolicy;

#[forbid(unsafe_code)]
mod fixture {
    use super::syscall;
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

    fn owner(args: &[String]) -> Result<(), Box<dyn std::error::Error>> {
        let directory = Path::new(&args[2]);
        let mut target = Command::new(env::current_exe()?);
        target.args(["tree", &args[3], &args[4]]);
        if args.get(5).is_some_and(|mode| mode == "owner-eof") {
            let ready = args[3].clone();
            thread::spawn(move || {
                for _ in 0..1000 {
                    if fs::read_to_string(&ready).is_ok_and(|s| s.split_whitespace().count() == 2) {
                        process::exit(0);
                    }
                    thread::sleep(Duration::from_millis(5));
                }
                process::exit(1);
            });
        }
        let outcome = run_process_cancelled(
            &mut target,
            directory,
            "owner",
            Limits::default(),
            None,
            &AtomicBool::new(false),
        );
        let status = match outcome {
            Ok((status, _)) => status.code().unwrap_or(-1).to_string(),
            Err(error) => error.refusal.to_string(),
        };
        fs::write(directory.join("status"), status)?;
        Ok(())
    }

    #[cfg(target_os = "linux")]
    fn reap_all_groups() -> io::Result<()> {
        let child = Command::new(env::current_exe()?)
            .arg("echo")
            .process_group(0)
            .stdout(Stdio::null())
            .spawn()?;
        syscall::reap_owned_children()?;
        assert!(
            !Path::new("/proc").join(child.id().to_string()).exists(),
            "owned child in another process group was not reaped"
        );
        Ok(())
    }

    pub(super) fn main() -> Result<(), Box<dyn std::error::Error>> {
        let args: Vec<_> = env::args().collect();
        match args.get(1).map_or("", String::as_str) {
            "started" => {
                println!("started");
                io::stdout().flush()?;
                sleep();
            }
            "sleep" => sleep(),
            "tree" => {
                let mut input = String::new();
                io::stdin().read_to_string(&mut input)?;
                assert_eq!(input, "");
                let mut child = Command::new("setsid")
                    .arg(env::current_exe()?)
                    .arg("sleep")
                    .stdin(Stdio::null())
                    .spawn()?;
                fs::write(&args[3], parent_pid()?)?;
                fs::write(&args[2], format!("{} {}", process::id(), child.id()))?;
                sleep();
                child.wait()?;
            }
            "owner" => owner(&args)?,
            "lease-child" => {
                let _lease = umpire_backend_process::JvmLease::inherited_or_acquire()?;
                fs::write(&args[2], "verified")?;
            }
            #[cfg(target_os = "linux")]
            "lifecycle-harness" => lifecycle_harness(&args[2], Path::new(&args[3]))?,
            "policy" => {
                println!("{}", env::var("JAVA_TOOL_OPTIONS").unwrap_or_else(|_| "None".into()));
                println!("{}", if syscall::parent_identity().1 { "True" } else { "False" });
            }
            "symlink" => {
                fs::remove_file("capture.stdout")?;
                std::os::unix::fs::symlink(&args[2], "capture.stdout")?;
                println!("captured");
            }
            "fifo" => {
                fs::remove_file("capture.stdout")?;
                fs::rename("replacement.fifo", "capture.stdout")?;
                println!("captured");
            }
            "combined" => {
                io::stdout().write_all(b"abcd")?;
                io::stderr().write_all(b"efgh")?;
            }
            "utf8" => io::stdout().write_all(&[255])?,
            "echo" => io::stdout().write_all(b"native\n")?,
            #[cfg(target_os = "linux")]
            "poison-atfork" => {
                syscall::poison_atfork_child()?;
                let mut command = Command::new(env::current_exe()?);
                command.arg("echo");
                let (status, output) = umpire_backend_process::run_process(
                    &mut command,
                    Path::new(&args[2]),
                    "poison",
                    Limits { timeout: Duration::from_secs(5), ..Limits::default() },
                    None,
                )?;
                assert!(status.success(), "pthread_atfork child handler ran: {status}");
                assert_eq!(output, "native\n");
            }
            #[cfg(target_os = "linux")]
            "reap-all-groups" => reap_all_groups()?,
            "argv-env" => {
                println!("arg:{}", args[2]);
                println!("value:{}", env::var("UMPIRE_PROCESS_VALUE")?);
                println!("removed:{}", env::var_os("UMPIRE_PROCESS_REMOVED").is_none());
            }
            "restricted-exec" => {
                syscall::restrict_descendants()?;
                return Err(Command::new("bash").arg(&args[2]).exec().into());
            }
            "descendant-attempt" => {
                syscall::restrict_descendants()?;
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

    #[cfg(target_os = "linux")]
    struct LifecycleCleanup {
        owner: process::Child,
        pids: std::path::PathBuf,
        supervisor: std::path::PathBuf,
    }
    #[cfg(target_os = "linux")]
    fn signal(pid: &str, name: &str) -> io::Result<()> {
        let status = Command::new("kill")
            .args([name, "--", pid])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status()?;
        if status.success() { Ok(()) } else { Err(io::Error::other("signal failed")) }
    }
    #[cfg(target_os = "linux")]
    impl Drop for LifecycleCleanup {
        fn drop(&mut self) {
            if let Ok(pids) = fs::read_to_string(&self.pids) {
                for pid in pids.split_whitespace() {
                    let _ = signal(pid, "-KILL");
                }
            }
            let _ = self.owner.kill();
            let _ = self.owner.wait();
            if let Ok(pid) = fs::read_to_string(&self.supervisor) {
                let _ = signal(pid.trim(), "-KILL");
            }
            let children_path = format!("/proc/self/task/{}/children", process::id());
            for _ in 0..100 {
                let children = fs::read_to_string(&children_path).unwrap_or_default();
                let mut live = false;
                for pid in children.split_whitespace() {
                    let zombie =
                        fs::read_to_string(format!("/proc/{pid}/status")).is_ok_and(|status| {
                            status
                                .lines()
                                .any(|line| line.starts_with("State:") && line.contains('Z'))
                        });
                    if !zombie {
                        live = true;
                        let _ = signal(pid, "-KILL");
                    }
                }
                if !live {
                    let _ = syscall::reap_owned_children();
                    break;
                }
                thread::sleep(Duration::from_millis(5));
            }
        }
    }
    #[cfg(target_os = "linux")]
    fn lifecycle_harness(mode: &str, directory: &Path) -> Result<(), Box<dyn std::error::Error>> {
        use std::{os::unix::process::ExitStatusExt, time::Instant};
        syscall::set_subreaper()?;
        let pids = directory.join("pids");
        let supervisor = directory.join("supervisor");
        let owner = Command::new(env::current_exe()?)
            .arg("owner")
            .arg(directory)
            .arg(&pids)
            .arg(&supervisor)
            .arg(mode)
            .process_group(0)
            .stdin(Stdio::null())
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .spawn()?;
        let mut cleanup = LifecycleCleanup { owner, pids, supervisor };
        let ready = Instant::now();
        while ready.elapsed() < Duration::from_secs(5) {
            if fs::read_to_string(&cleanup.pids)
                .is_ok_and(|text| text.split_whitespace().count() == 2)
                && fs::read_to_string(&cleanup.supervisor)
                    .is_ok_and(|text| text.parse::<u32>().is_ok())
            {
                break;
            }
            thread::sleep(Duration::from_millis(5));
        }
        let descendants = fs::read_to_string(&cleanup.pids)?;
        assert_eq!(
            descendants.split_whitespace().count(),
            2,
            "target and escaped descendant readiness"
        );
        let supervisor = fs::read_to_string(&cleanup.supervisor)?;
        match mode {
            "owner-group" | "owner-group-kill" => {
                let signum = if mode == "owner-group" { "-TERM" } else { "-KILL" };
                signal(&format!("-{}", cleanup.owner.id()), signum)?;
            }
            "supervisor" => signal(supervisor.trim(), "-TERM")?,
            "owner-eof" => (),
            _ => return Err("unknown lifecycle control".into()),
        }
        let deadline = Instant::now();
        let owner_status = loop {
            if let Some(status) = cleanup.owner.try_wait()? {
                break status;
            }
            if deadline.elapsed() > Duration::from_secs(5) {
                return Err("owner did not terminate".into());
            }
            thread::sleep(Duration::from_millis(5));
        };
        match mode {
            "owner-group" => assert_eq!(owner_status.signal(), Some(15)),
            "owner-group-kill" => assert_eq!(owner_status.signal(), Some(9)),
            "supervisor" => {
                assert!(owner_status.success());
                assert_eq!(fs::read_to_string(directory.join("status"))?, "143");
            }
            "owner-eof" => assert!(owner_status.success()),
            _ => unreachable!(),
        }
        let deadline = Instant::now();
        while descendants.split_whitespace().any(|pid| Path::new("/proc").join(pid).exists())
            && deadline.elapsed() < Duration::from_secs(2)
        {
            thread::sleep(Duration::from_millis(5));
        }
        for pid in descendants.split_whitespace() {
            assert!(
                !Path::new("/proc").join(pid).exists(),
                "owned backend PID {pid} survived before harness fallback cleanup"
            );
        }
        drop(cleanup);
        Ok(())
    }
}
fn main() -> Result<(), Box<dyn std::error::Error>> {
    fixture::main()
}
