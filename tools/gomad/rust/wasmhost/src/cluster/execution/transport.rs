//! Instance-tagged frames over the helper's one stdin/stdout pair (fn-178.1
//! protocol draft). An instance's frames are its single-execution frames with
//! `{"instance":"<id>",` in place of the opening `{`; stripping the prefix gives
//! back the single-execution bytes, so the per-instance `State` code is reused
//! unchanged.
use std::io::{self, BufRead, Read, Write};
use std::sync::{Arc, Mutex, MutexGuard};

#[cfg(test)]
#[path = "transport_corpus.rs"]
mod corpus;

#[cfg(test)]
#[path = "transport_input_corpus.rs"]
mod input_corpus;

#[cfg(test)]
#[path = "transport_trace_corpus.rs"]
mod trace_corpus;

#[cfg(test)]
#[path = "transport_replay.rs"]
mod replay_tests;

pub type SharedOutput = Arc<Mutex<Box<dyn Write + Send>>>;
pub type SharedInput = Arc<Mutex<Box<dyn BufRead + Send>>>;

/// Set once a Go frame cannot be attributed to the waiting instance (EOF, a
/// missing or foreign prefix). The whole run then ends as `invalid`.
#[derive(Clone, Default)]
pub struct Broken(Arc<Mutex<Option<String>>>);

impl Broken {
    pub fn set(&self, message: String) {
        let mut broken = lock(&self.0);
        if broken.is_none() {
            *broken = Some(message);
        }
    }

    pub fn get(&self) -> Option<String> {
        lock(&self.0).clone()
    }
}

/// Locks shared transport state even after a panic poisoned it; a panic is a
/// whole-run outcome, and the coordinator still writes the final frame.
pub fn lock<T: ?Sized>(mutex: &Mutex<T>) -> MutexGuard<'_, T> {
    mutex
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
}

pub fn prefix(id: &str) -> Vec<u8> {
    format!("{{\"instance\":\"{id}\",").into_bytes()
}

fn incarnation_prefix(id: &str, incarnation: u64) -> Vec<u8> {
    if incarnation == 1 {
        return prefix(id);
    }
    format!("{{\"instance\":\"{id}\",\"incarnation\":{incarnation},").into_bytes()
}

pub struct TaggedOutput {
    shared: SharedOutput,
    prefix: Vec<u8>,
    pending: Vec<u8>,
}

impl TaggedOutput {
    pub fn new(shared: SharedOutput, id: &str) -> Self {
        Self::incarnation(shared, id, 1)
    }

    pub fn incarnation(shared: SharedOutput, id: &str, incarnation: u64) -> Self {
        Self {
            shared,
            prefix: incarnation_prefix(id, incarnation),
            pending: Vec::new(),
        }
    }
}

impl Write for TaggedOutput {
    fn write(&mut self, data: &[u8]) -> io::Result<usize> {
        self.pending.extend_from_slice(data);
        while let Some(end) = self.pending.iter().position(|byte| *byte == b'\n') {
            let frame: Vec<u8> = self.pending.drain(..=end).collect();
            if frame.len() < 3 || frame[0] != b'{' || frame[1] == b'}' {
                return Err(io::Error::other("cluster frame is not a JSON object"));
            }
            let mut output = lock(&self.shared);
            output.write_all(&self.prefix)?;
            output.write_all(&frame[1..])?;
        }
        Ok(data.len())
    }

    fn flush(&mut self) -> io::Result<()> {
        lock(&self.shared).flush()
    }
}

pub struct TaggedInput {
    shared: SharedInput,
    prefix: Vec<u8>,
    frame: Vec<u8>,
    consumed: usize,
    broken: Broken,
    instance: String,
    incarnation: u64,
}

impl TaggedInput {
    pub fn new(shared: SharedInput, id: &str, broken: Broken) -> Self {
        Self::incarnation(shared, id, 1, broken)
    }

    pub fn incarnation(shared: SharedInput, id: &str, incarnation: u64, broken: Broken) -> Self {
        Self {
            shared,
            prefix: incarnation_prefix(id, incarnation),
            frame: Vec::new(),
            consumed: 0,
            broken,
            instance: id.into(),
            incarnation,
        }
    }

    pub fn replay(
        &mut self,
        request: &crate::model::replay::Request,
        mode: u8,
        limit: u64,
    ) -> wasmtime::Result<crate::model::replay_plan::ReplayPlan> {
        if self.consumed != self.frame.len() {
            return Err(wasmtime::Error::new(self.unattributable(
                "replay block overlaps a pending JSON frame".into(),
            )));
        }
        let result = {
            let mut input = lock(&self.shared);
            crate::model::replay::read(
                &mut **input,
                request,
                mode,
                limit,
                Some((&self.instance, self.incarnation)),
            )
        };
        if let Err(error) = &result
            && error
                .downcast_ref::<crate::model::replay::Preflight>()
                .is_some_and(|failure| matches!(failure.check, "framing" | "schema"))
        {
            self.broken.set(error.to_string());
        }
        result
    }

    fn unattributable(&self, message: String) -> io::Error {
        self.broken.set(message.clone());
        io::Error::other(message)
    }

    /// Reads one Go frame. Its prefix must name this instance before anything
    /// else is judged; a correctly tagged frame over the single-execution limit
    /// is drained from the pipe and handed on unterminated, so the instance's
    /// own `read_frame` reports `capacity` as a single execution would.
    fn next_frame(&mut self) -> io::Result<()> {
        let limit = crate::protocol::FRAME_LIMIT + self.prefix.len();
        let mut raw = Vec::new();
        let mut oversized = false;
        {
            let mut input = lock(&self.shared);
            loop {
                let available = input.fill_buf()?;
                if available.is_empty() {
                    drop(input);
                    return Err(self.unattributable("unexpected cluster protocol EOF".into()));
                }
                let count = available
                    .iter()
                    .position(|byte| *byte == b'\n')
                    .map_or(available.len(), |end| end + 1);
                let done = available[count - 1] == b'\n';
                if !oversized {
                    let keep = count.min(limit + 1 - raw.len());
                    raw.extend_from_slice(&available[..keep]);
                }
                input.consume(count);
                let checked = raw.len().min(self.prefix.len());
                if raw[..checked] != self.prefix[..checked] {
                    drop(input);
                    return Err(self.unattributable(
                        "cluster frame is not addressed to the waiting instance".into(),
                    ));
                }
                oversized = oversized || raw.len() > limit;
                if done {
                    break;
                }
            }
        }
        if raw.len() <= self.prefix.len() + 1 {
            return Err(self
                .unattributable("cluster frame is not addressed to the waiting instance".into()));
        }
        self.frame.clear();
        self.frame.push(b'{');
        self.frame.extend_from_slice(&raw[self.prefix.len()..]);
        if oversized {
            self.frame.pop_if(|byte| *byte == b'\n');
        }
        self.consumed = 0;
        Ok(())
    }
}

impl Read for TaggedInput {
    fn read(&mut self, buffer: &mut [u8]) -> io::Result<usize> {
        let available = self.fill_buf()?;
        let count = available.len().min(buffer.len());
        buffer[..count].copy_from_slice(&available[..count]);
        self.consume(count);
        Ok(count)
    }
}

impl BufRead for TaggedInput {
    fn fill_buf(&mut self) -> io::Result<&[u8]> {
        if self.consumed == self.frame.len() {
            self.next_frame()?;
        }
        Ok(&self.frame[self.consumed..])
    }

    fn consume(&mut self, amount: usize) {
        self.consumed = (self.consumed + amount).min(self.frame.len());
    }
}
