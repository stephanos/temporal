use super::{Flag, Isolation, LiveMemory, Stage};

#[derive(Clone, Copy, Debug)]
pub(crate) struct Policy {
    pub(crate) cores: f64,
    pub(crate) mem_gib: f64,
    pub(crate) allow_background: bool,
    pub(crate) deadline: Option<f64>,
    pub(crate) stop_at_deadline: bool,
    pub(crate) margin_gib: f64,
}

impl Policy {
    pub(super) fn validate(&self) -> Result<(), String> {
        if !self.cores.is_finite()
            || self.cores <= 0.0
            || !self.mem_gib.is_finite()
            || self.mem_gib < 0.0
            || !self.margin_gib.is_finite()
            || self.margin_gib < 0.0
            || self.deadline.is_some_and(|d| !d.is_finite())
        {
            return Err("invalid scheduler policy".into());
        }
        Ok(())
    }
}

fn isolation_allows(s: &Stage, running: &[&Stage], p: &Policy) -> bool {
    if s.isolation_class == Isolation::Solo
        || running.iter().any(|r| r.isolation_class == Isolation::Solo)
    {
        return false;
    }
    let exclusive: Vec<_> = running
        .iter()
        .filter(|r| r.isolation_class == Isolation::Exclusive)
        .collect();
    if s.isolation_class == Isolation::Budgeted {
        if !exclusive.is_empty()
            || running
                .iter()
                .any(|r| r.contract.flags.contains(&Flag::Heavy))
        {
            return false;
        }
    } else if s.contract.flags.contains(&Flag::Heavy)
        && running
            .iter()
            .any(|r| r.isolation_class == Isolation::Budgeted)
    {
        return false;
    }
    if s.isolation_class == Isolation::Exclusive {
        if !exclusive.is_empty() {
            return false;
        }
        if !p.allow_background || !s.contract.flags.contains(&Flag::Bgok) {
            return false;
        }
        if running
            .iter()
            .any(|r| !r.contract.flags.contains(&Flag::Bg))
        {
            return false;
        }
    } else if !exclusive.is_empty() {
        if !p.allow_background || !s.contract.flags.contains(&Flag::Bg) {
            return false;
        }
        if exclusive
            .iter()
            .any(|r| !r.contract.flags.contains(&Flag::Bgok))
        {
            return false;
        }
    }
    true
}

pub(crate) fn can_start(
    s: &Stage,
    running: &[&Stage],
    p: &Policy,
    live_memory: LiveMemory<'_>,
) -> Result<bool, String> {
    // The reference's idle-host exception precedes all cap and memory checks.
    if running.is_empty() {
        return Ok(true);
    }
    if !isolation_allows(s, running, p) {
        return Ok(false);
    }
    if s.contract.flags.contains(&Flag::Cleantree)
        && running.iter().any(|r| !r.contract.scratch.is_empty())
    {
        return Ok(false);
    }
    if !s.contract.scratch.is_empty()
        && running
            .iter()
            .any(|r| r.contract.flags.contains(&Flag::Cleantree))
    {
        return Ok(false);
    }
    if running.iter().any(|r| {
        r.contract
            .locks
            .iter()
            .any(|lock| s.contract.locks.contains(lock))
    }) {
        return Ok(false);
    }
    if running.iter().map(|r| r.contract.cores).sum::<f64>() + s.contract.cores > p.cores {
        return Ok(false);
    }
    let memory = running.iter().map(|r| r.contract.mem_gib).sum::<f64>();
    if memory + s.contract.mem_gib > p.mem_gib {
        return Ok(false);
    }
    if s.contract.flags.contains(&Flag::Memguard) && memory + 8.0 > p.mem_gib {
        return Ok(false);
    }
    for guarded in running
        .iter()
        .filter(|r| r.contract.flags.contains(&Flag::Memguard))
    {
        let others = running
            .iter()
            .filter(|r| !std::ptr::eq(**r, *guarded))
            .map(|r| r.contract.mem_gib)
            .sum::<f64>();
        if others + s.contract.mem_gib + 8.0 > p.mem_gib {
            return Ok(false);
        }
    }
    if s.contract.flags.contains(&Flag::Memguard)
        && let Some(sample) = live_memory
    {
        let available = sample()?;
        if !available.is_finite() || available < 0.0 {
            return Err("invalid available-memory observation".into());
        }
        if available < 8.0 + p.margin_gib + s.contract.mem_gib {
            return Ok(false);
        }
    }
    Ok(true)
}

pub(crate) fn parse_mem_available(text: &str) -> Result<f64, String> {
    let mut available = None;
    for line in text.lines() {
        if let Some(value) = line.strip_prefix("MemAvailable:") {
            if available.is_some() {
                return Err("duplicate MemAvailable observation".into());
            }
            let fields: Vec<_> = value.split_whitespace().collect();
            if fields.len() != 2
                || fields[1] != "kB"
                || !fields[0].bytes().all(|b| b.is_ascii_digit())
            {
                return Err("invalid MemAvailable observation".into());
            }
            let kib = fields[0]
                .parse::<u64>()
                .map_err(|_| "invalid MemAvailable observation")?;
            #[allow(
                clippy::cast_precision_loss,
                reason = "GiB admission follows reference f64 peak accounting."
            )]
            let gib = kib as f64 / 1_048_576.0;
            available = Some(gib);
        }
    }
    available.ok_or_else(|| "MemAvailable missing".into())
}

pub(crate) fn mem_available_gib() -> Result<f64, String> {
    use std::io::Read;
    let file =
        std::fs::File::open("/proc/meminfo").map_err(|e| format!("sample MemAvailable: {e}"))?;
    let mut text = String::new();
    file.take((1 << 20) + 1)
        .read_to_string(&mut text)
        .map_err(|e| format!("sample MemAvailable: {e}"))?;
    if text.len() > 1 << 20 {
        return Err("memory observation exceeds byte bound".into());
    }
    parse_mem_available(&text)
}
