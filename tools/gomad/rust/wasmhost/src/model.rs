use crate::transcript::Event;
use serde::{Deserialize, Serialize};
use sha2::Sha256;
use std::collections::HashMap;

mod choice_wire_generated;
mod guest_state;
mod imports;
pub mod inspection;
pub mod replay;
pub mod replay_plan;
mod reply;

const CHUNK: usize = 64 * 1024;
const MAX_TIME: u64 = i64::MAX as u64;
pub const CPU_CLOCK_UNSUPPORTED: &str = "CPU clocks outside stock WASI profile";

#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Config {
    pub entropy_key: [u8; 32],
    pub working_directory_sha256: String,
    pub arguments: StringDeclaration,
    pub environment: StringDeclaration,
    pub seed: u64,
    pub mode: u8,
    pub limit: u64,
    pub diagnostics: bool,
    #[serde(skip)]
    pub tape: Option<Vec<Decision>>,
    #[serde(default)]
    pub replay: Option<replay::Request>,
    pub epoch_nanos: u64,
    pub read_step_nanos: u64,
    pub now: u64,
    pub limits: ModelLimits,
    #[serde(default)]
    pub digest_only: bool,
    /// Broker split ownership: runtime controls and guest fd 1/2 writes stay
    /// remote to the Go broker; the helper owns namespace, volume, clock reads
    /// and polls on the broker-adopted clock, and the WASI transcript.
    #[serde(default)]
    pub split: bool,
    #[serde(default)]
    pub inspection: Option<inspection::Request>,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct StringDeclaration {
    bytes: usize,
    count: usize,
    chunks: usize,
    #[serde(skip)]
    packed: Vec<u8>,
    #[serde(skip)]
    offsets: Vec<u32>,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ModelLimits {
    pub calls: u64,
    pub transcript_bytes: u64,
    pub pending_events: u64,
}
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields, rename_all = "PascalCase")]
pub struct Decision {
    ordinal: u64,
    kind: u8,
    site_offset: u64,
    site_missing: bool,
    rank_override: bool,
    alternatives: u32,
    selected: u32,
    data: u32,
    selected_identity: [u8; 32],
    alternative_set_digest: [u8; 32],
    #[serde(skip)]
    creation_sites: Vec<CreationSite>,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct CreationSite {
    pub identity: [u8; 32],
    pub site_offset: u64,
    pub site_missing: bool,
}

impl CreationSite {
    #[must_use]
    pub fn fingerprint(&self, target: [u8; 32]) -> Option<String> {
        replay_plan::site_fingerprint(target, 4, self.site_offset, self.site_missing)
    }
}

impl Decision {
    #[must_use]
    pub fn creation_sites(&self) -> &[CreationSite] {
        &self.creation_sites
    }

    /// This decision as a replay tape records it. Tape records carry no
    /// creation sites, so a divergence's expected decision, which comes from
    /// the tape, compares against this view of a projected plan decision.
    #[must_use]
    pub fn tape_view(&self) -> Self {
        Self {
            creation_sites: Vec::new(),
            ..self.clone()
        }
    }
    #[must_use]
    pub fn site_fingerprint(&self, target: [u8; 32]) -> Option<String> {
        replay_plan::site_fingerprint(target, self.kind, self.site_offset, self.site_missing)
    }
    #[must_use]
    pub fn ordinal(&self) -> u64 {
        self.ordinal
    }
    #[must_use]
    pub fn kind(&self) -> u8 {
        self.kind
    }
    #[must_use]
    pub fn alternatives(&self) -> u32 {
        self.alternatives
    }
    #[must_use]
    pub fn selected(&self) -> u32 {
        self.selected
    }
    #[must_use]
    pub fn data(&self) -> u32 {
        self.data
    }
    #[must_use]
    pub fn alternative_set_digest(&self) -> [u8; 32] {
        self.alternative_set_digest
    }
    #[must_use]
    pub fn selected_identity(&self) -> [u8; 32] {
        self.selected_identity
    }
    #[must_use]
    pub fn rank_override(&self) -> bool {
        self.rank_override
    }
    #[must_use]
    pub fn site(&self) -> (u64, bool) {
        (self.site_offset, self.site_missing)
    }
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct Divergence {
    pub ordinal: u64,
    pub reason: u8,
    pub expected: Option<Decision>,
    pub observed: Option<Decision>,
    pub tape_records: u64,
}

impl Divergence {
    #[must_use]
    pub fn reason_name(&self) -> &'static str {
        match self.reason {
            1 => "kind",
            2 => "site",
            3 => "alternatives",
            4 => "selected",
            5 => "alternative_set",
            6 => "tape_exhausted",
            7 => "tape_unconsumed",
            8 => "identity_missing",
            9 => "identity_duplicate",
            10 => "alternative_capacity",
            11 => "observation",
            _ => "unknown",
        }
    }
}
/// A remote reply that would exceed a retained-evidence limit; classified as
/// the deterministic `capacity` outcome rather than a protocol violation.
#[derive(Debug)]
pub struct RemoteCapacity(pub &'static str);
impl std::fmt::Display for RemoteCapacity {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str(self.0)
    }
}
impl std::error::Error for RemoteCapacity {}

pub struct PollResult {
    pub events: Vec<Event>,
    pub errno: u16,
    pub unsupported: &'static str,
}

pub struct Model {
    pub(crate) linked_evidence: Option<crate::linked::Sink>,
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    event_decisions: u64,
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    pub event_owner: Option<inspection::Handle>,
    #[cfg(feature = "determinism-fixture")]
    fixture_fault: crate::determinism_fixture::Fault,
    config: Config,
    pub now: u64,
    pub advances: u64,
    pub transcript_bytes: u64,
    transcript_hash: Sha256,
    transcript_encoder: crate::transcript::Encoder,
    records: u64,
    branching_records: u64,
    choice_hash: Sha256,
    cursor: usize,
    configured: bool,
    finished: bool,
    peak: u32,
    elapsed: u64,
    terminal: Vec<u8>,
    pending: HashMap<&'static str, Vec<u8>>,
    entropy_offset: u64,
    pub divergence: Option<Divergence>,
    /// A revocable remote reply awaits confirmation before a local effect.
    pub sync_pending: bool,
    pub inspection: inspection::Transport,
}

#[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
impl Model {
    #[cfg(feature = "determinism-fixture")]
    pub(crate) fn fixture_choice_mode(&self) -> u8 {
        self.config.mode
    }
    pub fn event_context(&self) -> crate::event_order::Context {
        let handle = self.event_owner.as_ref().unwrap_or(&self.inspection.handle);
        crate::event_order::Context {
            instance: handle.node.clone(),
            incarnation: handle.incarnation,
            trace: self.records,
            decision: self.event_decisions,
            time_nanos: self.now.cast_signed(),
        }
    }
    fn event(&self, kind: &str, ordinal: u64, value: serde_json::Value) {
        crate::event_order::emit(&self.event_context(), kind, ordinal, value);
    }
    fn witness_event(&self, witness: &inspection::Witness) {
        let Ok(state) = <&[u8; 128]>::try_from(witness.runtime_state.as_slice()) else {
            crate::event_order::incomplete("invalid runtime-witness diagnostic length");
            return;
        };
        let virtual_time = i64::from_be_bytes(state[8..16].try_into().unwrap());
        self.event("draw-counter",self.records,serde_json::json!({"runq_draws":get64(state,40),"scheduler_draws":get64(state,48),"select_draws":get64(state,56),"runtime_rand_draws":get64(state,64),"runtime_cheaprand_draws":get64(state,72),"timer_draws":get64(state,80),"clock_tick_draws":get64(state,88)}));
        self.event("model-transition",self.records,serde_json::json!({"model":"runtime-witness","virtual_time":virtual_time,"state":crate::event_order::bytes_value(state)}));
    }
}
fn get32(data: &[u8], at: usize) -> u32 {
    u32::from_be_bytes(data[at..at + 4].try_into().unwrap())
}
fn get64(data: &[u8], at: usize) -> u64 {
    u64::from_be_bytes(data[at..at + 8].try_into().unwrap())
}
fn set32(data: &mut [u8], at: usize, value: u32) {
    data[at..at + 4].copy_from_slice(&value.to_be_bytes());
}
fn set64(data: &mut [u8], at: usize, value: u64) {
    data[at..at + 8].copy_from_slice(&value.to_be_bytes());
}

#[cfg(test)]
mod tests {
    mod default_cut_corpus;

    use super::*;
    use crate::{abi::Operation, protocol};
    use base64::{Engine as _, engine::general_purpose::STANDARD};
    use serde_json::{Value, json};
    use sha2::Digest;
    use std::hint::black_box;
    use std::io::Write;
    fn model() -> Model {
        let mut config: Config = serde_json::from_value(json!({"seed":7,"mode":1,"limit":4096,
            "diagnostics":false,"entropy_key":vec![0u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),"arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":6,"count":1,"chunks":1},"epoch_nanos":1000,"read_step_nanos":0,"now":1,
            "limits":{"calls":100,"transcript_bytes":65536,"pending_events":16}}))
        .unwrap();
        config.validate().unwrap();
        config
            .read_tape(&mut std::io::Cursor::new(Vec::new()))
            .unwrap();
        Model::new(config)
    }
    fn namespace() -> crate::namespace::Namespace {
        use crate::namespace::{Image, Limits, Namespace};
        Namespace::new(Image {
            entries: Vec::new(),
            writable_directories: Vec::new(),
            preopens: Vec::new(),
            working_directory: b"/".to_vec(),
            stdin: b"q".to_vec(),
            limits: Limits {
                files: 10,
                descriptors: 10,
                filesystem_bytes: 1024,
                output_bytes: 100_000,
            },
            epoch_nanos: 1000,
        })
        .unwrap()
    }
    #[test]
    fn namespace_capacity_refuses_before_mutation_and_preserves_completed_evidence() {
        use crate::namespace::Operation as N;
        let mut m = model();
        let mut n = namespace();
        let call = |data: &[u8]| {
            crate::namespace::call::Call::new(N::Write {
                fd: 1,
                data: data.to_vec(),
                offset: None,
            })
        };
        m.namespace_call(1, &mut n, call(b"ab")).unwrap();
        let _ = n.take_output();
        m.config.limits.transcript_bytes = m.transcript_bytes;
        let before = n.canonical_state();
        m.namespace_call(2, &mut n, call(b"x")).unwrap_err();
        assert_eq!(n.canonical_state(), before);
    }
    #[test]
    fn namespace_unsupported_exact_bounds_preserve_message_bytes_and_ids() {
        use crate::namespace::{Image, Limits, Namespace, Open, Operation as N};
        for (operation, limit) in [
            (N::SetFlags { fd: 3, flags: 2 }, 75),
            (
                N::Open(Open {
                    fd: 3,
                    path: b"x".to_vec(),
                    dirflags: 0,
                    oflags: 0,
                    base: 2,
                    inheriting: 0,
                    flags: 2,
                }),
                96,
            ),
            (
                N::Readlink {
                    fd: 3,
                    path: b"x".to_vec(),
                    length: 0,
                },
                92,
            ),
        ] {
            let mut n = Namespace::new(Image {
                entries: Vec::new(),
                writable_directories: Vec::new(),
                preopens: vec![b"/".to_vec()],
                working_directory: b"/".to_vec(),
                stdin: Vec::new(),
                limits: Limits {
                    files: 10,
                    descriptors: 10,
                    filesystem_bytes: 1024,
                    output_bytes: 1024,
                },
                epoch_nanos: 1000,
            })
            .unwrap();
            let before = n.canonical_state();
            let mut m = model();
            m.config.limits.transcript_bytes = limit - 1;
            m.namespace_call(
                1,
                &mut n,
                crate::namespace::call::Call::new(operation.clone()),
            )
            .unwrap_err();
            assert_eq!(n.canonical_state(), before);
            m.config.limits.transcript_bytes = limit;
            m.namespace_call(1, &mut n, crate::namespace::call::Call::new(operation))
                .unwrap();
            assert_eq!(n.canonical_state(), before);
        }
    }
    #[test]
    fn entropy_boundaries_and_literal_transcript_preserve_offsets() {
        let mut m = model();
        m.entropy_offset = 30;
        m.random(1, 6).unwrap();
        assert_eq!(
            m.pending["transcript"],
            b"GMWT\x01\0\0\0\x07\0\0\0\x01\0\0\0\0\0\0\0\x0c\0\0\0\x06\0\0\0\x1e\0\0\0\0\0\0\0"
        );
        assert!(m.random(2, 4 * 1024 * 1024 + 1).is_err());
        assert_eq!(m.entropy_offset, 36);
        m.entropy_offset = u64::MAX;
        assert!(m.random(2, 1).is_err());
        assert_eq!(m.entropy_offset, u64::MAX);
        assert_eq!(m.random(2, 0).unwrap(), [] as [u8; 0]);
    }
    /// Pins the streaming transcript digest to a value computed by an
    /// independent SHA-256 (`sha256sum`, `openssl dgst`), so the runtime-selected
    /// sha2 backend (aarch64 SHA-2 instructions or soft) cannot change transcript
    /// identities.
    #[test]
    fn transcript_digest_matches_pinned_independent_sha256() {
        let mut m = model();
        for id in 1..=1200_u64 {
            match id % 4 {
                0 => drop(m.random(id, (id % 61) as usize).unwrap()),
                1 => drop(m.clock_read(id, 1, id % 3).unwrap()),
                2 => m.yield_call(id).unwrap(),
                _ => drop(m.strings(id, Operation::ArgsSizesGet).unwrap()),
            }
        }
        let remote: Vec<u8> = (0..5000_u32).map(|i| (i * 31 % 251) as u8).collect();
        m.transcript(&remote);
        m.clock_read(1201, 0, 5).unwrap();
        let digest = m.metadata(1201)["transcript_sha256"].clone();
        m.request_state();
        let bytes = &m.pending["transcript"];
        assert_eq!(m.transcript_bytes, bytes.len() as u64);
        assert_eq!(digest, format!("{:x}", Sha256::digest(bytes)));
        for split in [1, 3, 63, 64, 65, 127, 4096] {
            let mut hash = Sha256::new();
            bytes.chunks(split).for_each(|chunk| hash.update(chunk));
            assert_eq!(digest, format!("{:x}", hash.finalize()));
        }
        assert_eq!(bytes.len(), 36252);
        assert_eq!(
            digest,
            "8a857b55e2835d563120e045c8ce32d5f0a2ad7488ef404276bda5aa21b13a32"
        );
    }
    /// Inspecting a pending frozen clock run keeps the run open and its digest
    /// stable across a flush, for every key variant. That each key (count,
    /// clock ID, precision, start) changes the digest is asserted by the
    /// `helper_transcript/model_frozen_clock_{base,count,id,precision,start}`
    /// corpus cases, whose `replaces` lists this test's former name
    /// `pending_clock_inspection_preserves_run_and_digest_binds_every_key`.
    #[test]
    fn pending_clock_inspection_preserves_run_and_digest() {
        for (count, clock_id, precision, now) in [
            (3, 1, 7, 1),
            (2, 1, 7, 1),
            (3, 0, 7, 1),
            (3, 1, 8, 1),
            (3, 1, 7, 2),
        ] {
            let mut m = model();
            m.now = now;
            for id in 1..=count {
                m.clock_read(id, clock_id, precision).unwrap();
                let _snapshot = m.metadata(id);
            }
            assert_eq!(m.transcript_bytes, 52);
            let before = m.metadata(count)["transcript_sha256"].clone();
            m.flush(&mut Vec::new(), true).unwrap();
            let digest = m.metadata(count)["transcript_sha256"].clone();
            assert_eq!(digest, before);
        }
    }
    fn decision() -> Vec<u8> {
        let mut bytes = vec![0; 96];
        set32(&mut bytes, 0, 1);
        set32(&mut bytes, 4, 1);
        set64(&mut bytes, 8, 42);
        set32(&mut bytes, 20, 2);
        set32(&mut bytes, 24, 1);
        bytes[32..64].fill(2);
        bytes[64..96].fill(1);
        bytes
    }
    fn bootstrap(model: &mut Model) {
        model.runtime("runtime_config", vec![0; 16]).unwrap();
    }
    #[test]
    fn runtime_creation_divergence_keeps_an_inspectable_prefix() {
        let mut recorded = model();
        bootstrap(&mut recorded);
        recorded.runtime("runtime_decision", decision()).unwrap();
        let raw = &recorded.pending["choice"];
        let mut expected = Decision {
            ordinal: 0,
            kind: 2,
            site_offset: 42,
            site_missing: false,
            rank_override: false,
            alternatives: 2,
            selected: 0,
            data: 0,
            selected_identity: [1; 32],
            alternative_set_digest: raw[64..].try_into().unwrap(),
            creation_sites: Vec::new(),
        };
        for reason in [1, 2, 3, 5, 4] {
            expected.kind = if reason == 1 { 2 } else { 1 };
            expected.site_offset = if reason == 2 { 43 } else { 42 };
            expected.alternatives = if reason == 3 { 3 } else { 2 };
            expected.alternative_set_digest = if reason == 5 {
                [9; 32]
            } else {
                raw[64..].try_into().unwrap()
            };
            expected.selected_identity = if reason == 4 { [9; 32] } else { [1; 32] };
            let mut replayed = model();
            replayed.config.mode = 2;
            replayed.config.tape = Some(vec![expected.clone()]);
            bootstrap(&mut replayed);
            let mut input = decision();
            set32(&mut input, 0, 2);
            input.resize(128, 0);
            replayed.runtime("runtime_decision", input).unwrap_err();
            assert_eq!(replayed.divergence.as_ref().unwrap().reason, reason);
            assert_eq!(replayed.divergence.as_ref().unwrap().ordinal, 0);
            assert_eq!(replayed.records, 0);
            assert!(replayed.pending.get("choice").is_none_or(Vec::is_empty));
        }
    }
    #[test]
    fn runtime_creation_sites_preserve_alternative_identities_and_exact_replay() {
        let identity = replay_plan::ExecutionIdentity {
            target_sha256: [1; 32],
            toolchain_build_key: "02".repeat(32),
            goos: "wasip1".into(),
            goarch: "wasm".into(),
            implementation_sha256: [3; 32],
        };
        let mut previous = model();
        bootstrap(&mut previous);
        previous.runtime("runtime_decision", decision()).unwrap();
        let previous_trace = previous.pending["choice"].clone();
        let previous_plan = replay_plan::project(&previous_trace, &identity).unwrap();
        assert_eq!(previous_plan.decisions[0].creation_sites(), []);
        let mut input = decision();
        set32(&mut input, 0, 2);
        input.resize(128, 0);
        set64(&mut input, 96, 123);
        input[120] = 4;
        let mut recorded = model();
        bootstrap(&mut recorded);
        recorded.runtime("runtime_decision", input.clone()).unwrap();
        let trace = recorded.pending["choice"].clone();
        assert_eq!(&trace[192 + 8..], &previous_trace[8..]);
        let plan = replay_plan::project(&trace, &identity).unwrap();
        let sites = plan.decisions[0].creation_sites();
        assert_eq!(
            sites,
            &[
                CreationSite {
                    identity: [1; 32],
                    site_offset: 0,
                    site_missing: true
                },
                CreationSite {
                    identity: [2; 32],
                    site_offset: 123,
                    site_missing: false
                },
            ]
        );
        assert!(sites[0].fingerprint(identity.target_sha256).is_none());
        let inspected = replay_plan::inspect(&trace, &identity).unwrap();
        assert!(
            inspected
                .sites
                .iter()
                .any(|site| site.kind == "goroutine-creation"
                    && site.site_offset == 123
                    && site.fingerprint == sites[1].fingerprint(identity.target_sha256).unwrap())
        );
        for (tape, expected, request) in [
            (plan.decisions, trace, input),
            (previous_plan.decisions, previous_trace, decision()),
        ] {
            let mut replayed = model();
            replayed.config.mode = 2;
            replayed.config.tape = Some(tape);
            bootstrap(&mut replayed);
            replayed.runtime("runtime_decision", request).unwrap();
            assert_eq!(replayed.pending["choice"], expected);
        }
    }
    #[test]
    fn runtime_creation_diagnostics_keep_trace_ordinals() {
        let mut recorded = model();
        recorded.config.diagnostics = true;
        bootstrap(&mut recorded);
        let mut input = decision();
        set32(&mut input, 0, 2);
        input.resize(224, 0);
        recorded.runtime("runtime_decision", input).unwrap();
        let diagnostics = &recorded.pending["diagnostic"];
        assert_eq!(diagnostics.len(), 288);
        for ordinal in 0..3 {
            assert_eq!(get64(diagnostics, ordinal * 96), ordinal as u64);
        }
    }
    #[test]
    fn debug_exchange_preserves_open_clock_run_and_canonical_state() {
        let mut model = model();
        bootstrap(&mut model);
        model.clock_read(1, 1, 0).unwrap();
        let before = model.metadata(1);
        let mut frame = [0u8; 20];
        frame[..4].copy_from_slice(&1u32.to_le_bytes());
        frame[4..8].copy_from_slice(&3u32.to_le_bytes());
        frame[12..16].copy_from_slice(&4u32.to_le_bytes());
        frame[16..20].copy_from_slice(&1u32.to_le_bytes());
        model.debug_exchange(&mut frame).unwrap();
        assert_eq!(model.metadata(1), before);
        assert!(model.transcript_encoder.is_open());
        model.clock_read(2, 1, 0).unwrap();
        let mut reference = super::tests::model();
        bootstrap(&mut reference);
        reference.clock_read(1, 1, 0).unwrap();
        reference.clock_read(2, 1, 0).unwrap();
        assert_eq!(model.metadata(2), reference.metadata(2));
    }
    #[expect(
        clippy::cast_possible_truncation,
        reason = "This fixture contains a 96-byte decision and a 128-byte witness."
    )]
    fn debug_predecision() -> Vec<u8> {
        let pending = decision();
        let mut frame = vec![0; 16 + pending.len() + 128];
        frame[..4].copy_from_slice(&1u32.to_le_bytes());
        frame[4..8].copy_from_slice(&1u32.to_le_bytes());
        frame[12..16].copy_from_slice(&((pending.len() + 128) as u32).to_le_bytes());
        frame[16..16 + pending.len()].copy_from_slice(&pending);
        set64(&mut frame, 16 + pending.len() + 112, 1);
        frame
    }
    #[test]
    fn debug_seals_validated_pending_decision_without_consuming_tape_suffix() {
        let mut recorded = model();
        bootstrap(&mut recorded);
        recorded
            .inspection
            .lock()
            .snapshot(recorded.inspection_cut(), &[], &[]);
        recorded.debug_exchange(&mut debug_predecision()).unwrap();
        let boundary = recorded.inspection.lock().evidence.boundaries[0].clone();
        let request = {
            let capture = recorded.inspection.lock();
            inspection::Request {
                target: recorded.inspection.handle.clone(),
                decision: 0,
                expected: boundary,
                world_sha256: inspection::digest(&capture.evidence.world),
                lifecycle_sha256: inspection::digest(&capture.evidence.lifecycle),
                stream_sha256: vec![
                    ["choice", "wasi", "network", "storage", "stdout", "stderr"]
                        .into_iter()
                        .map(|name| (name.into(), inspection::digest(&[])))
                        .collect(),
                ],
                probes: vec![],
                commands: vec![],
                registries: vec![inspection::Registry::default()],
            }
        };
        recorded.runtime("runtime_decision", decision()).unwrap();
        let bytes = &recorded.pending["choice"];
        let expected = Decision {
            ordinal: 0,
            kind: 1,
            site_offset: 42,
            site_missing: false,
            rank_override: false,
            alternatives: 2,
            selected: 0,
            data: 0,
            selected_identity: [1; 32],
            alternative_set_digest: bytes[64..].try_into().unwrap(),
            creation_sites: Vec::new(),
        };
        let mut replay = model();
        replay.config.mode = 2;
        replay.config.tape = Some(vec![expected.clone(), expected]);
        replay.inspection.request = Some(request.clone());
        bootstrap(&mut replay);
        replay
            .inspection
            .lock()
            .snapshot(replay.inspection_cut(), &[], &[]);
        let mut frame = debug_predecision();
        replay.debug_exchange(&mut frame).unwrap();

        let mut changed = model();
        bootstrap(&mut changed);
        changed
            .inspection
            .lock()
            .snapshot(changed.inspection_cut(), &[], &[]);
        changed.inspection.request = Some(request);
        changed.inspection.lock().stream(
            &changed.inspection.handle,
            "stdout",
            b"changed neighbor output",
        );
        assert!(
            changed
                .debug_exchange(&mut debug_predecision())
                .unwrap_err()
                .to_string()
                .contains("instrumentation divergence")
        );
        assert!(!changed.inspection.sealed);
    }
    #[cfg(feature = "diagnostic-profile")]
    #[test]
    fn diagnostic_model_hooks_measure_actual_hash_encoding_and_flush() {
        let _session = crate::diagnostic::Session::start();
        let mut m = model();
        bootstrap(&mut m);
        m.runtime("runtime_decision", decision()).unwrap();
        m.handle(3, "clock_time_get", &json!({"clock_id":1,"precision":0}))
            .unwrap();
        m.flush(&mut Vec::new(), true).unwrap();
        let report = crate::diagnostic::snapshot();
        assert_eq!(report["nested"]["alternative_hash"]["count"], 1);
        assert_eq!(report["nested"]["transcript_hash"]["count"], 2);
        assert_eq!(report["nested"]["flush"]["count"], 1);
        assert_eq!(report["nested"]["encoding"]["count"], 3);
    }
    #[cfg(feature = "diagnostic-profile")]
    #[test]
    #[ignore = "instrumentation overhead calibration only; not a guest or performance gate"]
    fn diagnostic_calibration_preserves_model_bytes_without_timing_feedback() {
        use std::time::Instant;
        let run = |active: bool| {
            let mut m = model();
            m.config.limit = 16 * 1024 * 1024;
            m.config.limits.transcript_bytes = 16 * 1024 * 1024;
            let session = active.then(crate::diagnostic::Session::start);
            bootstrap(&mut m);
            let before = crate::metrics::diagnostic_process_usage();
            let start = Instant::now();
            let mut output = Vec::new();
            let mut id = 1;
            for _ in 0..10_000 {
                m.runtime("runtime_decision", decision()).unwrap();
                id += 1;
                for _ in 0..5 {
                    id += 1;
                    m.handle(id, "clock_time_get", &json!({"clock_id":1,"precision":0}))
                        .unwrap();
                    m.flush(&mut output, false).unwrap();
                }
            }
            let wall = crate::metrics::elapsed(start);
            let after = crate::metrics::diagnostic_process_usage();
            m.flush(&mut output, true).unwrap();
            let metadata = m.metadata(id);
            let profile = active.then(crate::diagnostic::snapshot);
            let cpu = match (before, after) {
                (Some((u0, s0, _)), Some((u1, s1, rss))) => {
                    json!({"user_nanos":u1-u0,"system_nanos":s1-s0,"peak_rss_bytes_high_water":rss})
                }
                _ => Value::Null,
            };
            drop(session);
            (
                output,
                metadata,
                json!({"wall_nanos":wall,"cpu":cpu,"profile":profile}),
            )
        };
        for replicate in 1..=3 {
            let (control_bytes, control_metadata, control) = run(false);
            let (sampled_bytes, sampled_metadata, sampled) = run(true);
            assert_eq!(control_bytes, sampled_bytes);
            assert_eq!(control_metadata, sampled_metadata);
            assert_eq!(
                sampled["profile"]["nested"]["alternative_hash"]["count"],
                10_000
            );
            assert_eq!(
                sampled["profile"]["nested"]["transcript_hash"]["count"],
                10_001
            );
            println!(
                "{}",
                json!({"diagnostic_only":true,"replicate":replicate,"decisions":10_000,
                "clock_reads":50_000,"canonical_equal":true,"control":control,"sampled":sampled,
                "control_scope":"same diagnostic build with no active session; cfg hooks still perform inactive checks",
                "excludes":"Wasmtime/JIT guest instructions, import trampolines, IPC callbacks, final flush and byte comparison",
                "qualification":false})
            );
        }
    }
    #[test]
    fn typed_clock_preserves_json_reference_and_literal_canonical_bytes() {
        for (input, clock_id, timestamp, literal_input) in [
            (
                json!({"clock_id":0,"precision":0}),
                Some(0),
                Some(1001),
                "{\"clock_id\":0,\"precision\":0}",
            ),
            (
                json!({"clock_id":1,"precision":u64::MAX}),
                Some(1),
                Some(1),
                "{\"clock_id\":1,\"precision\":18446744073709551615}",
            ),
            (
                json!({"clock_id":2,"precision":0}),
                Some(2),
                None,
                "{\"clock_id\":2,\"precision\":0}",
            ),
            (
                json!({"clock_id":u32::MAX,"precision":0}),
                Some(u64::from(u32::MAX)),
                None,
                "{\"clock_id\":4294967295,\"precision\":0}",
            ),
            (
                json!({"clock_id":"1","precision":0}),
                None,
                None,
                "{\"clock_id\":\"1\",\"precision\":0}",
            ),
            (json!({"precision":0}), None, None, "{\"precision\":0}"),
            (
                json!({"clock_id":-1,"precision":0}),
                None,
                None,
                "{\"clock_id\":-1,\"precision\":0}",
            ),
        ] {
            let mut typed = model();
            let mut reference = model();
            typed.config.read_step_nanos = 3;
            reference.config.read_step_nanos = 3;
            let call = format!(
                "{{\"type\":\"call\",\"id\":7,\"op\":\"clock_time_get\",\"input\":{literal_input}}}\n"
            );
            let actual = typed.clock_time_get(7, clock_id, call.as_bytes()).unwrap();
            if clock_id.is_none() {
                assert_eq!(actual, None);
            }
            let reply = reference
                .handle(7, "clock_time_get", &input)
                .unwrap()
                .unwrap();
            assert_eq!(reply.unsupported, actual.is_none());
            assert_eq!(reply.output["timestamp"].as_u64(), actual);
            typed.close_clock_run();
            reference.close_clock_run();
            if clock_id.is_none() {
                let mut literal = b"GMWT\x01\0\0\0".to_vec();
                literal.extend_from_slice(&[
                    if timestamp.is_some() { 33 } else { 1 },
                    u8::from(timestamp.is_none()),
                    0,
                    0,
                    7,
                    0,
                    0,
                    0,
                    0,
                    0,
                    0,
                    0,
                ]);
                literal.extend_from_slice(
                    &(if timestamp.is_some() { 28_u32 } else { 53_u32 }).to_le_bytes(),
                );
                literal.extend_from_slice(
                    &u32::try_from(clock_id.unwrap_or(u64::from(u32::MAX)))
                        .unwrap()
                        .to_le_bytes(),
                );
                literal.extend_from_slice(&input["precision"].as_u64().unwrap().to_le_bytes());
                if timestamp.is_some() {
                    literal.extend_from_slice(b"\x01\0\0\0\0\0\0\0\x01\0\0\0\0\0\0\0");
                } else {
                    literal.extend_from_slice(b"\x25\0\0\0CPU clocks outside stock WASI profile");
                }
                assert_eq!(typed.pending["transcript"], literal);
                assert_eq!(typed.now, if timestamp.is_some() { 4 } else { 1 });
            }
            assert_eq!(typed.pending, reference.pending);
            assert_eq!(typed.metadata(7), reference.metadata(7));
        }
    }
    #[test]
    fn typed_clock_reservation_digest_and_read_step_failures_preserve_state() {
        let call = b"{\"type\":\"call\",\"id\":1,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":1,\"precision\":0}}\n";
        for (digest, capacity) in [(false, 52), (true, 1)] {
            let mut typed = model();
            typed.config.digest_only = digest;
            typed.config.limits.transcript_bytes = capacity;
            typed.clock_time_get(1, Some(1), call).unwrap();
            assert_eq!(typed.pending.contains_key("transcript"), !digest);
        }
        let mut typed = model();
        typed.now = MAX_TIME - typed.config.epoch_nanos;
        typed.config.read_step_nanos = 1;
        let before = typed.now;
        typed.clock_time_get(1, Some(1), call).unwrap_err();
        assert!(typed.pending.is_empty());
        let unsupported_call = b"{\"type\":\"call\",\"id\":2,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":2,\"precision\":0}}\n";
        assert_eq!(
            typed.clock_time_get(2, Some(2), unsupported_call).unwrap(),
            None
        );
        assert_eq!(typed.now, before);
    }
    #[test]
    #[ignore = "diagnostic profiling only; not a qualification or performance gate"]
    #[expect(
        clippy::too_many_lines,
        reason = "The profiling test compares all import stages using one shared measurement setup."
    )]
    #[expect(
        clippy::cast_possible_truncation,
        reason = "The fixture's alternative counts are 2 or 32, with indices bounded by those counts."
    )]
    fn profile_local_import_stages_against_json_reference() {
        use crate::engine::{Limits, State};
        use std::{
            sync::{Arc, Mutex},
            time::Instant,
        };
        #[derive(Clone)]
        struct Capture(Arc<Mutex<Vec<u8>>>);
        impl Write for Capture {
            fn write(&mut self, bytes: &[u8]) -> std::io::Result<usize> {
                self.0.lock().unwrap().extend_from_slice(bytes);
                Ok(bytes.len())
            }
            fn flush(&mut self) -> std::io::Result<()> {
                Ok(())
            }
        }
        fn configured(diagnostics: bool) -> Model {
            let mut model = model();
            model.config.limit = 64 * 1024 * 1024;
            model.config.diagnostics = diagnostics;
            model.config.read_step_nanos = 1;
            model.config.limits.calls = 1 << 28;
            model.config.limits.transcript_bytes = 64 * 1024 * 1024;
            model.config.validate().unwrap();
            model
        }
        fn timed(mut body: impl FnMut(usize), iterations: usize) -> u64 {
            let started = Instant::now();
            for index in 0..iterations {
                body(black_box(index));
            }
            crate::metrics::elapsed(started)
        }
        fn sample() -> crate::metrics::Metrics {
            let mut metrics = crate::metrics::Metrics::default();
            metrics.sample_process();
            metrics
        }
        for (alternatives, diagnostics, iterations) in
            [(2, false, 100_000), (32, false, 50_000), (32, true, 50_000)]
        {
            let mut request = vec![0; 32 + alternatives * 32 + if diagnostics { 96 } else { 0 }];
            set32(&mut request, 0, 1);
            set32(&mut request, 4, 1);
            set64(&mut request, 8, 42);
            set32(&mut request, 20, alternatives as u32);
            set32(&mut request, 24, alternatives as u32 - 1);
            for (index, identity) in request[32..32 + alternatives * 32]
                .as_chunks_mut::<32>()
                .0
                .iter_mut()
                .enumerate()
            {
                identity.fill((alternatives - index) as u8);
            }
            let ordered: Vec<[u8; 32]> =
                (1..=alternatives).map(|index| [index as u8; 32]).collect();
            let baseline_nanos = timed(
                |index| {
                    black_box(index);
                },
                iterations,
            );
            let copy_nanos = timed(
                |_| {
                    black_box(black_box(&request).clone());
                },
                iterations,
            );
            let format_nanos = timed(
                |_| {
                    black_box(format!("runtime_{}", black_box("decision")));
                },
                iterations,
            );
            let base64_roundtrip_nanos = timed(
                |_| {
                    let encoded = STANDARD.encode(black_box(&request));
                    black_box(STANDARD.decode(encoded).unwrap());
                },
                iterations,
            );
            let clone_sort_nanos = timed(
                |_| {
                    let alternatives = black_box(&request)[32..32 + alternatives * 32]
                        .as_chunks::<32>()
                        .0
                        .to_vec();
                    let mut ordered = alternatives.clone();
                    ordered.sort_unstable();
                    assert!(
                        !ordered.contains(&[0; 32])
                            && !ordered.windows(2).any(|pair| pair[0] == pair[1])
                    );
                    black_box(
                        ordered
                            .iter()
                            .position(|identity| *identity == alternatives[alternatives.len() - 1])
                            .unwrap(),
                    );
                },
                iterations,
            );
            let alternative_hash_nanos = timed(
                |_| {
                    let mut hash = Sha256::new();
                    hash.update(b"gomad-choice-alternative-set/v1\0");
                    hash.update((alternatives as u64).to_be_bytes());
                    for identity in black_box(&ordered) {
                        hash.update(identity);
                    }
                    black_box(hash.finalize());
                },
                iterations,
            );
            let mut core = configured(diagnostics);
            bootstrap(&mut core);
            let core_decision_nanos = timed(
                |_| {
                    black_box(core.runtime("runtime_decision", request.clone()).unwrap());
                },
                iterations,
            );
            drop(core);
            let mut core = configured(diagnostics);
            bootstrap(&mut core);
            let mut observation = vec![0; if diagnostics { 192 } else { 96 }];
            observation[8] = 3;
            observation[9] = 2;
            observation[11] = 1;
            set32(&mut observation, 12, 1);
            let core_observation_nanos = timed(
                |index| {
                    set64(&mut observation, 0, index as u64);
                    black_box(
                        core.runtime("runtime_observation", observation.clone())
                            .unwrap(),
                    );
                },
                iterations,
            );
            drop(core);
            let mut pending = configured(diagnostics);
            pending.queue("choice", &[0; 96]);
            pending.queue("transcript", &[0; 96]);
            if diagnostics {
                pending.queue("diagnostic", &[0; 96]);
            }
            let pending_flush_checks_nanos = timed(
                |_| {
                    black_box(&mut pending)
                        .flush(&mut std::io::sink(), false)
                        .unwrap();
                },
                iterations,
            );
            let clock = json!({"clock_id":1,"precision":0});
            let mut clocks = configured(diagnostics);
            let clock_handle_nanos = timed(
                |index| {
                    black_box(
                        clocks
                            .handle(index as u64 + 1, "clock_time_get", &clock)
                            .unwrap(),
                    );
                },
                iterations,
            );
            drop(clocks);
            let mut canonical = None;
            let mut paths = Vec::new();
            for (json_reference, typed_clock) in [(false, false), (true, false), (false, true)] {
                let capture = Capture(Arc::new(Mutex::new(Vec::new())));
                let mut state = State {
                    memory: None,
                    input: Box::new(std::io::Cursor::new(Vec::<u8>::new())),
                    output: Box::new(capture.clone()),
                    limits: Limits {
                        max: 65536,
                        peak: 65536,
                        failure: None,
                    },
                    calls: 0,
                    completed_calls: 0,
                    host_nanos: None,
                    stdout_progress: false,
                    output_limit: 1024,
                    output_bytes: 0,
                    terminal: None,
                    argument_sizes: None,
                    environment_sizes: None,
                    model: Some(configured(diagnostics)),
                    dormant_debug: None,
                    namespace: None,
                    request_scratch: Vec::new(),
                    write_scratch: protocol::WriteBuffers::default(),
                    broker: None,
                    network: None,
                };
                let invoke = |state: &mut State, op: &str, bytes: Vec<u8>| {
                    let name = format!("runtime_{op}");
                    if json_reference {
                        let reply = state
                            .callback(&name, json!({"data_base64":STANDARD.encode(bytes)}))
                            .unwrap();
                        black_box(
                            STANDARD
                                .decode(reply.output["data_base64"].as_str().unwrap())
                                .unwrap(),
                        );
                    } else {
                        black_box(state.runtime(&name, bytes).unwrap());
                    }
                };
                invoke(&mut state, "config", vec![0; 16]);
                let before = sample();
                let wall_nanos = timed(
                    |index| {
                        invoke(&mut state, "decision", request.clone());
                        set64(&mut observation, 0, index as u64 * 2 + 1);
                        invoke(&mut state, "observation", observation.clone());
                        if typed_clock {
                            black_box(state.clock_time_get(1, 0).unwrap());
                        } else {
                            black_box(state.callback("clock_time_get", clock.clone()).unwrap());
                        }
                    },
                    iterations,
                );
                let after = sample();
                let mut finish = vec![0; 16];
                set32(&mut finish, 0, 1);
                set32(&mut finish, 4, 5);
                set64(&mut finish, 8, iterations as u64);
                invoke(&mut state, "finish", finish);
                let model = state.model.as_mut().unwrap();
                model.flush(&mut state.output, true).unwrap();
                let metadata = model.metadata(state.completed_calls);
                let encoded = capture.0.lock().unwrap();
                let mut streams: HashMap<String, Vec<u8>> = HashMap::new();
                for frame in encoded
                    .split(|byte| *byte == b'\n')
                    .filter(|line| !line.is_empty())
                {
                    let frame: Value = serde_json::from_slice(frame).unwrap();
                    streams
                        .entry(frame["stream"].as_str().unwrap().to_owned())
                        .or_default()
                        .extend(
                            STANDARD
                                .decode(frame["data_base64"].as_str().unwrap())
                                .unwrap(),
                        );
                }
                let evidence_bytes: usize = streams.values().map(Vec::len).sum();
                if let Some((reference_streams, reference_metadata)) = &canonical {
                    assert_eq!(&streams, reference_streams);
                    assert_eq!(&metadata, reference_metadata);
                } else {
                    canonical = Some((streams, metadata));
                }
                paths.push(json!({"path":if typed_clock {"binary_runtime_with_typed_clock"} else if json_reference {"counterfactual_json_runtime_reference"} else {"binary_runtime_with_json_clock"}, "mixed_imports":iterations * 3,"wall_nanos":wall_nanos,"user_nanos":after.process_user_nanos.saturating_sub(before.process_user_nanos),"system_nanos":after.process_system_nanos.saturating_sub(before.process_system_nanos),"resource_available":after.resource_usage_available,"peak_rss_bytes":after.peak_rss_bytes,"evidence_bytes":evidence_bytes}));
            }
            eprintln!(
                "{}",
                json!({"diagnostic_only":true,"excludes":"Wasmtime guest instructions/import trampoline, actual guest runtime loop, parent callbacks, final flush and evidence comparison","alternatives":alternatives,"diagnostics":diagnostics,"iterations":iterations,"stages_nanos":{"baseline_loop":baseline_nanos,"one_request_vec_copy":copy_nanos,"runtime_op_format":format_nanos,"base64_encode_decode_counterfactual":base64_roundtrip_nanos,"alternatives_clone_sort_validate_rank":clone_sort_nanos,"alternative_set_sha256":alternative_hash_nanos,"core_decision_with_request_copy":core_decision_nanos,"core_observation_with_request_copy":core_observation_nanos,"underfull_flush_checks":pending_flush_checks_nanos,"current_clock_handle_json_no_flush":clock_handle_nanos},"paths":paths,"canonical_concatenated_streams_and_metadata_equal":true})
            );
        }
    }
    #[test]
    fn bounds_trace_transcript_and_evidence_chunks() {
        let mut m = model();
        m.config.limit = 160;
        bootstrap(&mut m);
        m.runtime("runtime_decision", decision()).unwrap();
        m.runtime("runtime_decision", decision()).unwrap_err();
        m.config.limits.transcript_bytes = 51;
        assert!(
            m.handle(3, "clock_time_get", &json!({"clock_id":1,"precision":0}))
                .err()
                .unwrap()
                .to_string()
                .contains("transcript-bytes")
        );
        assert_eq!(m.now, 1);
        m.pending.clear();
        m.queue("transcript", &vec![5; CHUNK + 1]);
        m.flush(&mut Vec::new(), false).unwrap();
        assert_eq!(m.pending["transcript"].len(), 1);
    }
    #[cfg(any(feature = "determinism-fixture", feature = "stress-jitter"))]
    #[test]
    fn witness_event_preserves_signed_virtual_time_without_changing_canonical_bytes() {
        let model = model();
        let session = crate::event_order::LinkedSession::new();
        crate::event_order::start_linked(&session).unwrap();
        let mut state = [0u8; 128];
        state[8..16].copy_from_slice(&(-7i64).to_be_bytes());
        state[112..120].copy_from_slice(&1u64.to_be_bytes());
        let witness = inspection::Witness::decode(&state).unwrap();
        model.witness_event(&witness);
        assert!(model.pending.is_empty());
        assert_eq!(model.records, 0);
        assert_eq!(witness.runtime_state, state);
        let raw = crate::event_order::finish_linked(true).unwrap();
        let frames: Vec<Value> = std::str::from_utf8(&raw)
            .unwrap()
            .lines()
            .map(|line| {
                serde_json::from_str(line.strip_prefix("gomad-wasm-event-order-v1 ").unwrap())
                    .unwrap()
            })
            .collect();
        let event = frames
            .iter()
            .find(|frame| frame["value"]["model"] == "runtime-witness")
            .unwrap();
        assert_eq!(event["value"]["virtual_time"], -7);
        assert_eq!(
            event["value"]["state"],
            crate::event_order::bytes_value(&state)
        );
    }
}
