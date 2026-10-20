use std::{
    panic::{AssertUnwindSafe, catch_unwind},
    sync::{
        Mutex,
        atomic::{AtomicBool, Ordering},
    },
    thread,
};

enum Failure {
    Panic,
    Execution(String),
}

/// # Errors
/// Refuses zero slots and aborts the campaign on any execution error or host panic.
pub fn run<T: Send, F: Fn(u64) -> Result<T, String> + Sync>(
    seeds: &[u64],
    parallel: usize,
    execute: F,
) -> Result<Vec<T>, String> {
    if parallel == 0 {
        return Err("parallel slots must be positive".into());
    }
    let jobs = Mutex::new(seeds.iter().copied().enumerate());
    let aborted = AtomicBool::new(false);
    let completed = thread::scope(|scope| {
        let mut workers = Vec::new();
        for _ in 0..parallel.min(seeds.len()) {
            let (jobs, aborted, execute) = (&jobs, &aborted, &execute);
            workers.push(scope.spawn(move || {
                let mut completed = Vec::new();
                loop {
                    let job = {
                        let mut jobs = jobs
                            .lock()
                            .map_err(|_| Failure::Execution("slot queue poisoned".into()))?;
                        if aborted.load(Ordering::Acquire) {
                            break;
                        }
                        jobs.next()
                    };
                    let Some((ordinal, seed)) = job else {
                        break;
                    };
                    match catch_unwind(AssertUnwindSafe(|| execute(seed))) {
                        Ok(Ok(result)) => completed.push((ordinal, result)),
                        Ok(Err(error)) => {
                            aborted.store(true, Ordering::Release);
                            return Err(Failure::Execution(error));
                        }
                        Err(_) => {
                            aborted.store(true, Ordering::Release);
                            return Err(Failure::Panic);
                        }
                    }
                }
                Ok(completed)
            }));
        }
        let mut completed = Vec::new();
        let mut failure = None;
        let mut panicked = false;
        for worker in workers {
            match worker.join() {
                Ok(Ok(results)) => completed.extend(results),
                Ok(Err(Failure::Execution(error))) => {
                    if failure.is_none() {
                        failure = Some(error);
                    }
                }
                Ok(Err(Failure::Panic)) | Err(_) => panicked = true,
            }
        }
        if panicked {
            return Err("host panic in execution slot; campaign aborted".into());
        }
        if let Some(error) = failure {
            return Err(error);
        }
        Ok(completed)
    })?;
    if aborted.load(Ordering::Acquire) || completed.len() != seeds.len() {
        return Err("execution slots aborted".into());
    }
    let mut completed = completed;
    completed.sort_unstable_by_key(|(ordinal, _)| *ordinal);
    Ok(completed.into_iter().map(|(_, result)| result).collect())
}

#[cfg(test)]
mod tests {
    // A host panic must discard all slot results; a guest fixture cannot trigger a Rust host panic.
    #[test]
    fn panic_aborts_campaign() {
        let result = super::run(&[7, 8], 2, |seed| {
            assert!(seed != 7, "slot panic");
            Ok(seed)
        });
        assert_eq!(
            result,
            Err("host panic in execution slot; campaign aborted".into())
        );
        let barrier = std::sync::Barrier::new(2);
        let result: Result<Vec<()>, String> = super::run(&[7, 8], 2, |seed| {
            barrier.wait();
            assert!(seed != 8, "simultaneous host panic");
            Err("simultaneous execution error".into())
        });
        assert_eq!(
            result,
            Err("host panic in execution slot; campaign aborted".into())
        );
    }
}
