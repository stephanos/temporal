#![cfg(not(feature = "conformance"))]

use gomad_wasmhost::linked::{self, OutputConsumer, OutputStream};
use serde_json::json;
use sha2::{Digest as _, Sha256};
use std::{
    io,
    sync::{Arc, Mutex, mpsc},
    thread,
    time::Duration,
};

const WAIT: Duration = Duration::from_secs(10);

fn request(marker: &str, tail: &str) -> linked::Request {
    let module = format!(
        r#"(module
            (import "wasi_snapshot_preview1" "args_sizes_get" (func $sizes (param i32 i32) (result i32)))
            (import "wasi_snapshot_preview1" "args_get" (func $args (param i32 i32) (result i32)))
            (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
            (memory (export "memory") 1)
            (data (i32.const 256) "{marker}")
            (func (export "_start")
                (drop (call $sizes (i32.const 40) (i32.const 44)))
                (drop (call $args (i32.const 32) (i32.const 128)))
                (i32.store (i32.const 0) (i32.const 128))
                (i32.store (i32.const 4) (i32.const {length}))
                (drop (call $write (i32.const 1) (i32.const 0) (i32.const 1) (i32.const 16)))
                (i32.store (i32.const 0) (i32.const 256))
                (drop (call $write (i32.const 2) (i32.const 0) (i32.const 1) (i32.const 16)))
                {tail}))"#,
        length = marker.len(),
    )
    .into_bytes();
    linked::Request {
        module_sha256: format!("{:x}", Sha256::digest(&module)),
        module,
        limits: linked::Limits {
            memory_bytes: 65536,
            fuel: 1_000_000,
            output_bytes: 4096,
            evidence_bytes: 1 << 20,
            wall_timeout: WAIT,
        },
        namespace: Some(linked::NamespaceImage {
            entries: Vec::new(),
            writable_directories: Vec::new(),
            preopens: vec![b"/".to_vec()],
            working_directory: b"/".to_vec(),
            stdin: Vec::new(),
            epoch_nanos: 0,
            limits: linked::NamespaceLimits {
                files: 16,
                descriptors: 16,
                filesystem_bytes: 4096,
                output_bytes: 4096,
            },
        }),
        storage: None,
        model: Some(linked::ModelInput {
            configuration: serde_json::from_value(json!({
                "entropy_key":vec![0_u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),
                "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
                "seed":0,"mode":1,"limit":1_u64<<20,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,
                "limits":{"calls":100,"transcript_bytes":1_u64<<20,"pending_events":16}
            })).unwrap(),
            arguments: vec![marker.into()],
            environment: vec!["PWD=/".into()],
            replay_trace: None,
        }),
    }
}

#[derive(Default)]
struct Capture(Mutex<Vec<(OutputStream, Vec<u8>)>>);

impl OutputConsumer for Capture {
    fn write(&self, stream: OutputStream, bytes: &[u8]) -> io::Result<()> {
        self.0.lock().unwrap().push((stream, bytes.to_vec()));
        Ok(())
    }
}

struct Gate {
    observed: mpsc::SyncSender<(OutputStream, Vec<u8>)>,
    release: Mutex<mpsc::Receiver<()>>,
}

impl OutputConsumer for Gate {
    fn write(&self, stream: OutputStream, bytes: &[u8]) -> io::Result<()> {
        self.observed
            .send((stream, bytes.to_vec()))
            .map_err(io::Error::other)?;
        self.release
            .lock()
            .unwrap()
            .recv_timeout(WAIT)
            .map_err(io::Error::other)
    }
}

#[test]
fn short_output_callbacks_apply_backpressure_to_completion() {
    let mut prepared = linked::prepare(request("short-marker", "")).unwrap();
    prepared.conditions(linked::Conditions::compiled());
    let (observe, observed) = mpsc::sync_channel(2);
    let (release, released) = mpsc::sync_channel(1);
    prepared.output(Arc::new(Gate {
        observed: observe,
        release: Mutex::new(released),
    }));
    let (complete, completed) = mpsc::sync_channel(1);
    thread::scope(|scope| {
        scope.spawn(move || complete.send(linked::execute_prepared(prepared)).unwrap());
        for stream in [OutputStream::Stdout, OutputStream::Stderr] {
            assert_eq!(
                observed.recv_timeout(WAIT).unwrap(),
                (stream, b"short-marker".to_vec())
            );
            assert!(matches!(
                completed.try_recv(),
                Err(mpsc::TryRecvError::Empty)
            ));
            release.send(()).unwrap();
        }
        let result = completed.recv_timeout(WAIT).unwrap().unwrap();
        assert_eq!(result.outcome.termination, "exit");
        assert_eq!(result.outcome.exit_code, Some(0));
        assert_eq!(result.evidence["stdout"], b"short-marker");
        assert_eq!(result.evidence["stderr"], b"short-marker");
    });
}

#[test]
fn concurrent_requests_keep_streams_arguments_and_completion_isolated() {
    let mut first = linked::prepare(request("first", "")).unwrap();
    first.conditions(linked::Conditions::compiled());
    let mut second = linked::prepare(request("second", "")).unwrap();
    second.conditions(linked::Conditions::compiled());
    let (observe_a, output_a) = mpsc::sync_channel(2);
    let (observe_b, output_b) = mpsc::sync_channel(2);
    let (release_a, permission_a) = mpsc::sync_channel(1);
    let (release_b, permission_b) = mpsc::sync_channel(1);
    first.output(Arc::new(Gate {
        observed: observe_a,
        release: Mutex::new(permission_a),
    }));
    second.output(Arc::new(Gate {
        observed: observe_b,
        release: Mutex::new(permission_b),
    }));
    thread::scope(|scope| {
        let first = scope.spawn(move || linked::execute_prepared(first).unwrap());
        let second = scope.spawn(move || linked::execute_prepared(second).unwrap());
        for stream in [OutputStream::Stdout, OutputStream::Stderr] {
            assert_eq!(
                output_a.recv_timeout(WAIT).unwrap(),
                (stream, b"first".to_vec())
            );
            assert_eq!(
                output_b.recv_timeout(WAIT).unwrap(),
                (stream, b"second".to_vec())
            );
            release_b.send(()).unwrap();
            release_a.send(()).unwrap();
        }
        for (result, marker) in [
            (first.join().unwrap(), b"first".as_slice()),
            (second.join().unwrap(), b"second".as_slice()),
        ] {
            assert_eq!(result.outcome.termination, "exit");
            assert_eq!(result.evidence["stdout"], marker);
            assert_eq!(result.evidence["stderr"], marker);
        }
    });
}

struct Failing(Mutex<usize>);
impl OutputConsumer for Failing {
    fn write(&self, _: OutputStream, _: &[u8]) -> io::Result<()> {
        *self.0.lock().unwrap() += 1;
        Err(io::Error::other("controlled output failure"))
    }
}

#[test]
fn consumer_failure_retains_admitted_bytes_once_and_stops_execution() {
    let consumer = Arc::new(Failing(Mutex::new(0)));
    let mut prepared = linked::prepare(request("before-error", "")).unwrap();
    prepared.conditions(linked::Conditions::compiled());
    prepared.output(consumer.clone());
    let result = linked::execute_prepared(prepared).unwrap();
    assert_eq!(result.outcome.termination, "infrastructure");
    assert!(result.outcome.message.contains("controlled output failure"));
    assert_eq!(result.evidence["stdout"], b"before-error");
    assert!(result.evidence.get("stderr").is_none_or(Vec::is_empty));
    assert_eq!(*consumer.0.lock().unwrap(), 1);
    assert!(result.outcome.local_model.is_some());
}

#[test]
fn streaming_does_not_forward_output_rejected_by_namespace_limit() {
    let consumer = Arc::new(Capture::default());
    let mut input = request("too-long", "");
    input.limits.output_bytes = 1;
    input.namespace.as_mut().unwrap().limits.output_bytes = 1;
    let mut prepared = linked::prepare(input).unwrap();
    prepared.conditions(linked::Conditions::compiled());
    prepared.output(consumer.clone());
    let result = linked::execute_prepared(prepared).unwrap();
    assert_eq!(result.outcome.termination, "capacity");
    assert!(consumer.0.lock().unwrap().is_empty());
    assert!(result.evidence.get("stdout").is_none_or(Vec::is_empty));
}

#[test]
fn streaming_does_not_forward_output_rejected_by_evidence_limit() {
    let consumer = Arc::new(Capture::default());
    let mut input = request("too-long", "");
    input.limits.evidence_bytes = 1;
    let mut prepared = linked::prepare(input).unwrap();
    prepared.conditions(linked::Conditions::compiled());
    prepared.output(consumer.clone());
    let result = linked::execute_prepared(prepared).unwrap();
    assert_eq!(result.outcome.termination, "capacity");
    assert!(consumer.0.lock().unwrap().is_empty());
    assert!(result.evidence.get("stdout").is_none_or(Vec::is_empty));
}

struct CancelAfterOutput {
    cancellation: linked::Cancellation,
    capture: Capture,
}
impl OutputConsumer for CancelAfterOutput {
    fn write(&self, stream: OutputStream, bytes: &[u8]) -> io::Result<()> {
        self.capture.write(stream, bytes)?;
        self.cancellation.cancel();
        Ok(())
    }
}

#[test]
fn cancellation_retains_and_forwards_only_available_output() {
    let cancellation = linked::Cancellation::default();
    let consumer = Arc::new(CancelAfterOutput {
        cancellation: cancellation.clone(),
        capture: Capture::default(),
    });
    let mut prepared = linked::prepare(request("before-cancel", "(loop br 0)")).unwrap();
    prepared.conditions(linked::Conditions::compiled());
    prepared.cancellation(cancellation);
    prepared.output(consumer.clone());
    let result = linked::execute_prepared(prepared).unwrap();
    assert_eq!(result.outcome.termination, "cancelled");
    assert_eq!(result.evidence["stdout"], b"before-cancel");
    let captured = consumer.capture.0.lock().unwrap();
    for (stream, bytes) in captured.iter() {
        let stream = match stream {
            OutputStream::Stdout => "stdout",
            OutputStream::Stderr => "stderr",
        };
        assert_eq!(result.evidence[stream], *bytes);
    }
}

#[test]
fn streaming_and_capture_only_retain_identical_evidence() {
    let mut prepared = linked::prepare(request("identical", "")).unwrap();
    prepared.conditions(linked::Conditions::compiled());
    let captured = linked::execute_prepared(prepared).unwrap();
    let consumer = Arc::new(Capture::default());
    let mut prepared = linked::prepare(request("identical", "")).unwrap();
    prepared.conditions(linked::Conditions::compiled());
    prepared.output(consumer.clone());
    let streamed = linked::execute_prepared(prepared).unwrap();
    assert_eq!(streamed.evidence, captured.evidence);
    assert_eq!(
        *consumer.0.lock().unwrap(),
        vec![
            (OutputStream::Stdout, b"identical".to_vec()),
            (OutputStream::Stderr, b"identical".to_vec())
        ]
    );
}
