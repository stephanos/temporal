use std::{env, fs, io, path::PathBuf, process::Command, thread, time::Duration};
use umpire_backend_process::{JvmLease, OwnedProcess};

fn main() -> io::Result<()> {
    let proof = PathBuf::from(env::var_os("FN25412_PROOF_ROOT").ok_or_else(|| io::Error::other("missing proof root"))?);
    if env::var_os("FN25412_PROOF_CHILD").is_some() {
        let code = xtask::gate::scheduler::scheduled("tests", || {
            let mut command = Command::new("mise");
            command.args(["exec", "--", "cargo", "nextest", "run", "--locked", "-p", "umpire-tlc",
                "--run-ignored", "default", "--no-tests", "fail", "-E",
                "test(/^runner::batch_tests::/) or test(/^runner::cache_tests::/) or test(/^runner::fixture::/) or test(=controls::real_tlc_healthy_batches_use_at_most_two_model_launches)"])
                .env("UMPIRE_CACHE_DISABLE", "1").env("UMPIRE_QUICK_CACHE", "0")
                .env("NEXTEST_PROFILE", "required").env("NEXTEST_TEST_THREADS", "4");
            let output = xtask::gate::work_counts::observe_streamed(&command, None, true, true)?
                .ok_or_else(|| io::Error::other("missing actual test observer"))?;
            Ok(output.status.code().unwrap_or(1))
        })?;
        std::process::exit(code);
    }
    fs::create_dir_all(proof.join("receipts"))?;
    let lease = JvmLease::inherited_or_acquire()?;
    let mut command = Command::new(env::current_exe()?);
    command.current_dir(env::current_dir()?).env("FN25412_PROOF_CHILD", "1")
        .env("UMPIRE_GATE_RECEIPTS", proof.join("receipts"))
        .env("UMPIRE_GATE_SCOPE", "framework").env("UMPIRE_GATE_STAGE", "tests");
    let stdout = fs::File::create(proof.join("stdout.raw.log"))?;
    let stderr = fs::File::create(proof.join("stderr.raw.log"))?;
    let mut child = OwnedProcess::spawn_leased(&command, &proof, &stdout, &stderr, &lease)?;
    let status = loop {
        if let Some(status) = child.try_wait()? { break status; }
        thread::sleep(Duration::from_millis(10));
    };
    drop(child);
    if !status.success() { return Err(io::Error::other(format!("proof child failed: {status}"))); }
    Ok(())
}
