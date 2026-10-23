use super::{CHUNK, Decision, Divergence, MAX_TIME, Model, RemoteCapacity, inspection};
use crate::protocol::{self, Reply};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use sha2::Digest;
use std::io::Write;
use wasmtime::{Error, Result};

impl Model {
    pub(crate) fn interrupted(&mut self) -> Result<()> {
        if self.finished {
            return Err(Error::msg("completed runtime on storage interruption"));
        }
        if self.config.mode != 1
            && let Some(tape) = &self.config.tape
            && let Some(expected) = tape.get(self.cursor).cloned()
        {
            return Err(self.diverge(7, Some(expected), None));
        }
        Ok(())
    }
    pub fn inspection_cut(&self) -> inspection::Cut {
        inspection::Cut {
            handle: self.inspection.handle.clone(),
            witness: self.inspection.witness.clone(),
            clock_run: self
                .transcript_encoder
                .pending()
                .map(|bytes| bytes.to_vec())
                .unwrap_or_default(),
            ..Default::default()
        }
    }
    pub fn request_state(&mut self) -> protocol::RequestState {
        self.close_clock_run();
        protocol::RequestState {
            advances: self.advances,
            now: self.now,
            transcript_bytes: self.transcript_bytes,
            offered_output_bytes: None,
        }
    }
    pub fn metadata(&self, calls: u64) -> Value {
        let mut hash = self.transcript_hash.clone();
        if let Some(bytes) = self.transcript_encoder.pending() {
            hash.update(bytes);
        }
        json!({"now":self.now,"advances":self.advances,"calls":calls,"records":self.records,
            "trace_sha256":format!("{:x}",self.choice_hash.clone().finalize()),"branching_records":self.branching_records,
            "tape_records":self.config.tape.as_ref().map_or(0,Vec::len),
            "tape_sha256":self.config.replay.as_ref().map_or("",|replay|replay.tape_sha256.as_str()),
            "next_expected":self.config.tape.as_ref().and_then(|tape|tape.get(self.cursor)),
            "cursor":self.cursor,"configured":self.configured,"finished":self.finished,"inspection_sealed":self.inspection.sealed,
            "peak":self.peak,"elapsed":self.elapsed,"terminal_base64":STANDARD.encode(&self.terminal),
            "transcript_bytes":self.transcript_bytes,
            "transcript_sha256":format!("{:x}",hash.finalize())})
    }
    pub(super) fn queue(&mut self, stream: &'static str, bytes: &[u8]) {
        if matches!(stream, "choice" | "stdout" | "stderr") {
            self.inspection
                .lock()
                .stream(&self.inspection.handle, stream, bytes);
        }
        self.pending
            .entry(stream)
            .or_default()
            .extend_from_slice(bytes);
    }
    pub fn flush(&mut self, output: &mut dyn Write, all: bool) -> Result<()> {
        #[cfg(feature = "diagnostic-profile")]
        let _flush = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Flush);
        if !all
            && self
                .linked_evidence
                .as_ref()
                .is_some_and(crate::linked::Sink::streaming)
        {
            self.flush_output(output)?;
        }
        if all && !self.inspection.sealed {
            self.close_clock_run();
        }
        self.flush_streams(
            output,
            &[
                "choice",
                "diagnostic",
                "transcript",
                "stdout",
                "stderr",
                "storage",
                "storage_state",
                "storage_image",
                "storage_precrash_image",
                "network",
                "inspection",
                "debug",
            ],
            all,
        )
    }
    /// Sends full chunks of the named evidence streams, and with `all` every
    /// pending byte.
    fn flush_streams(
        &mut self,
        output: &mut dyn Write,
        streams: &[&'static str],
        all: bool,
    ) -> Result<()> {
        for &stream in streams {
            if let Some(bytes) = self.pending.get_mut(stream) {
                while bytes.len() >= CHUNK || all && !bytes.is_empty() {
                    let count = bytes.len().min(CHUNK);
                    if let Some(sink) = &self.linked_evidence {
                        let result = sink.append(stream, &bytes[..count]);
                        bytes.drain(..count);
                        result?;
                        continue;
                    }
                    protocol::send_frame(
                        output,
                        &json!({"type":"evidence","stream":stream,
                        "data_base64":STANDARD.encode(&bytes[..count])}),
                    )?;
                    bytes.drain(..count);
                }
            }
        }
        Ok(())
    }
    /// Queues terminal storage evidence; it flushes with the final streams.
    pub fn storage_evidence(&mut self, evidence: crate::storage_transport::Evidence) {
        self.queue("storage", &evidence.log);
        self.queue("storage_state", &evidence.state);
        if let Some(image) = evidence.precrash_image {
            self.queue("storage_precrash_image", &image);
        }
        if let Some(image) = evidence.image {
            self.queue("storage_image", &image);
        }
    }
    pub fn inspection_evidence(&mut self, metadata: Option<Vec<u8>>) {
        if let Some(bytes) = metadata {
            self.queue("inspection", &bytes);
        }
        self.inspection.finish_output();
        if !self.inspection.output.is_empty() {
            let bytes = std::mem::take(&mut self.inspection.output);
            self.queue("debug", &bytes);
        }
    }
    pub(crate) fn network_evidence(&mut self, bytes: &[u8]) {
        self.queue("network", bytes);
    }
    pub fn namespace_output(&mut self, stdout: &[u8], stderr: &[u8]) {
        self.queue("stdout", stdout);
        self.queue("stderr", stderr);
    }
    pub fn flush_output(&mut self, output: &mut dyn Write) -> Result<()> {
        self.flush_streams(output, &["stdout", "stderr"], true)
    }
    pub fn transcript(&mut self, bytes: &[u8]) {
        self.inspection
            .lock()
            .stream(&self.inspection.handle, "transcript", bytes);
        #[cfg(feature = "diagnostic-profile")]
        let hash = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::TranscriptHash);
        self.transcript_hash.update(bytes);
        #[cfg(feature = "diagnostic-profile")]
        drop(hash);
        self.transcript_bytes += bytes.len() as u64;
        if !self.config.digest_only {
            self.queue("transcript", bytes);
        }
    }
    pub fn remote(&mut self, reply: &Reply, chunks: &[u8]) -> Result<()> {
        let state = reply
            .local_state
            .as_ref()
            .ok_or_else(|| Error::msg("missing remote model state"))?;
        if state.now < self.now
            || state.now > MAX_TIME - self.config.epoch_nanos
            || state.advances < self.advances
        {
            return Err(Error::msg("invalid remote clock state"));
        }
        let encoded = reply
            .transcript_base64
            .as_ref()
            .ok_or_else(|| Error::msg("missing remote transcript"))?;
        if !chunks.is_empty() && !encoded.is_empty() {
            return Err(Error::msg(
                "remote transcript cannot combine chunks and inline bytes",
            ));
        }
        let bytes = {
            #[cfg(feature = "diagnostic-profile")]
            let _encoding = crate::diagnostic::WallSpan::new(crate::diagnostic::Nested::Encoding);
            if chunks.is_empty() {
                STANDARD.decode(encoded).map_err(Error::new)?
            } else {
                chunks.to_vec()
            }
        };
        if bytes.len() > protocol::FRAME_LIMIT {
            return Err(Error::msg("remote transcript capacity"));
        }
        if !self.config.digest_only
            && self.transcript_bytes + bytes.len() as u64 > self.config.limits.transcript_bytes
        {
            return Err(Error::new(RemoteCapacity(
                "WASI capacity: transcript-bytes",
            )));
        }
        self.close_clock_run();
        self.now = state.now;
        self.advances = state.advances;
        self.transcript(&bytes);
        Ok(())
    }
    /// Split ownership: adopts the broker-raised clock from a `continue`.
    pub fn adopt_broker_clock(&mut self, now: u64) -> bool {
        if now < self.now || now > MAX_TIME - self.config.epoch_nanos {
            return false;
        }
        self.close_clock_run();
        self.now = now;
        true
    }
    /// Split ownership: adopts a remote call that Go completed before the
    /// broker revoked the incarnation: its clock state and, for WASI calls,
    /// its canonical record, as the Go-owned path counts and records it.
    pub fn interrupted_call(&mut self, now: u64, advances: u64, record: &[u8]) -> Result<()> {
        if now < self.now
            || now > MAX_TIME - self.config.epoch_nanos
            || advances < self.advances
            || record.len() > protocol::FRAME_LIMIT
            || !self.config.digest_only
                && self.transcript_bytes + record.len() as u64 > self.config.limits.transcript_bytes
        {
            return Err(Error::msg("invalid interrupted call state"));
        }
        self.close_clock_run();
        self.now = now;
        self.advances = advances;
        if !record.is_empty() {
            self.transcript(record);
        }
        Ok(())
    }
    pub(super) fn reserve_transcript(&self, reservation: u64) -> Result<()> {
        let header = if self.transcript_bytes == 0 {
            crate::transcript::HEADER.len() as u64
        } else {
            0
        };
        if !self.config.digest_only
            && self.transcript_bytes + header + reservation > self.config.limits.transcript_bytes
        {
            return Err(Error::new(crate::namespace::Boundary {
                kind: "capacity",
                dimension: "transcript-bytes",
            }));
        }
        Ok(())
    }
    pub(crate) fn close_clock_run(&mut self) {
        if let Some(bytes) = self.transcript_encoder.close() {
            self.transcript_bytes -= crate::transcript::CLOCK_RUN_BYTES;
            self.transcript(&bytes);
        }
    }
    pub(super) fn append_record(&mut self, bytes: &[u8]) {
        self.close_clock_run();
        if self.transcript_bytes == 0 {
            self.transcript(crate::transcript::HEADER);
        }
        self.transcript(bytes);
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        self.event(
            "wasi-op",
            self.transcript_bytes,
            crate::event_order::bytes_value(bytes),
        );
    }
    pub(super) fn append(&mut self, record: &[u8], diagnostic: &[u8]) -> Result<()> {
        if 64 + (self.records + 1) * 96 > self.config.limit {
            return Err(Error::msg("WASI capacity: runtime-choice-bytes"));
        }
        self.queue("choice", record);
        #[cfg(feature = "diagnostic-profile")]
        let hash = crate::diagnostic::OwnerSpan::new(crate::diagnostic::Owner::Transcript);
        self.choice_hash.update(record);
        #[cfg(feature = "diagnostic-profile")]
        drop(hash);
        if record[9] & 1 != 0 && super::get32(record, 12) > 1 {
            self.branching_records += 1;
        }
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        self.event(
            "choice",
            self.records,
            crate::event_order::bytes_value(record),
        );
        if self.config.diagnostics {
            self.queue("diagnostic", diagnostic);
        }
        self.records += 1;
        #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
        if record[9] & 1 != 0 {
            self.event_decisions += 1;
        }
        Ok(())
    }
    pub(super) fn diverge(
        &mut self,
        reason: u8,
        expected: Option<Decision>,
        observed: Option<Decision>,
    ) -> Error {
        let names = [
            "",
            "kind",
            "site",
            "alternatives",
            "selected",
            "alternative_set",
            "tape_exhausted",
            "tape_unconsumed",
        ];
        self.divergence = Some(Divergence {
            ordinal: self.cursor as u64,
            reason,
            expected,
            observed,
            tape_records: self.config.tape.as_ref().unwrap().len() as u64,
        });
        Error::msg(format!(
            "WASM choice {}: {}",
            self.cursor, names[reason as usize]
        ))
    }
}
