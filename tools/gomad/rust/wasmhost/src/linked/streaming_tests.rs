use super::*;
use serde_json::json;

struct Observe(mpsc::SyncSender<(OutputStream, Vec<u8>)>);

impl OutputConsumer for Observe {
    fn write(&self, stream: OutputStream, bytes: &[u8]) -> io::Result<()> {
        self.0
            .send((stream, bytes.to_vec()))
            .map_err(io::Error::other)
    }
}

fn model_and_namespace() -> (model::Model, namespace::Namespace) {
    let configuration = serde_json::from_value(json!({
        "entropy_key":vec![0_u8;32],"working_directory_sha256":format!("{:x}",Sha256::digest(b"/")),
        "arguments":{"bytes":0,"count":0,"chunks":0},"environment":{"bytes":0,"count":0,"chunks":0},
        "seed":0,"mode":1,"limit":1_u64<<20,"diagnostics":false,"epoch_nanos":0,"read_step_nanos":0,"now":1,
        "limits":{"calls":100,"transcript_bytes":1_u64<<20,"pending_events":16}
    })).unwrap();
    let model = model::Model::new_linked(configuration);
    let namespace = namespace::Namespace::new(NamespaceImage {
        entries: Vec::new(),
        writable_directories: Vec::new(),
        preopens: vec![b"/".to_vec()],
        working_directory: b"/".to_vec(),
        stdin: Vec::new(),
        epoch_nanos: 0,
        limits: NamespaceLimits {
            files: 16,
            descriptors: 16,
            filesystem_bytes: 4096,
            output_bytes: 4096,
        },
    })
    .unwrap();
    (model, namespace)
}

#[test]
fn short_markers_are_forwarded_before_guest_waits_for_controller_release() {
    const WAIT: Duration = Duration::from_secs(10);
    let module = br#"(module
        (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
        (import "test_control" "wait" (func $wait (param i32)))
        (memory (export "memory") 1)
        (data (i32.const 128) "short-marker")
        (func (export "_start")
            (i32.store (i32.const 0) (i32.const 128))
            (i32.store (i32.const 4) (i32.const 12))
            (drop (call $write (i32.const 1) (i32.const 0) (i32.const 1) (i32.const 16)))
            (call $wait (i32.const 1))
            (drop (call $write (i32.const 2) (i32.const 0) (i32.const 1) (i32.const 16)))
            (call $wait (i32.const 2))))"#;
    let (model, namespace) = model_and_namespace();
    let (observe, observed) = mpsc::sync_channel(2);
    let sink =
        Sink::new(1 << 20).with_output(Some(Arc::new(Observe(observe))), Cancellation::default());
    let (reach, reached) = mpsc::sync_channel(1);
    let (release, released) = mpsc::sync_channel(1);
    let (complete, completed) = mpsc::sync_channel(1);
    thread::scope(|scope| {
        let sink = sink.clone();
        scope.spawn(move || {
            let engine = Engine::new(&engine::configuration(false)).unwrap();
            let module = wasmtime::Module::new(&engine, module).unwrap();
            let mut linker = wasmtime::Linker::new(&engine);
            crate::abi::Operation::resolve("wasi_snapshot_preview1", "fd_write")
                .unwrap()
                .link(&mut linker)
                .unwrap();
            let released = Mutex::new(released);
            linker
                .func_wrap(
                    "test_control",
                    "wait",
                    move |phase: i32| -> wasmtime::Result<()> {
                        reach.send(phase).map_err(Error::new)?;
                        released
                            .lock()
                            .unwrap()
                            .recv_timeout(WAIT)
                            .map_err(Error::new)
                    },
                )
                .unwrap();
            let mut model = model;
            model.linked_evidence = Some(sink.clone());
            let mut store = Store::new(
                &engine,
                engine::new_state(
                    Box::new(Cursor::new(Vec::<u8>::new())),
                    Box::new(NoRemote {
                        #[cfg(feature = "diagnostic-profile")]
                        attempts: Arc::new(std::sync::atomic::AtomicU64::new(0)),
                    }),
                    65536,
                    4096,
                    Some(model),
                    Some(namespace),
                ),
            );
            store.limiter(|state| &mut state.limits);
            store.set_fuel(10000).unwrap();
            let instance = linker.instantiate(&mut store, &module).unwrap();
            let start = instance
                .get_typed_func::<(), ()>(&mut store, "_start")
                .unwrap();
            let executed = start.call(&mut store, ());
            let mut outcome = engine::terminal_outcome(&mut store, executed, 10000).unwrap();
            engine::finish_model(store.data_mut(), &mut outcome).unwrap();
            drop(store);
            complete.send(outcome).unwrap();
        });
        for (phase, stream) in [(1, OutputStream::Stdout), (2, OutputStream::Stderr)] {
            assert_eq!(reached.recv_timeout(WAIT).unwrap(), phase);
            assert_eq!(
                observed.try_recv().unwrap(),
                (stream, b"short-marker".to_vec())
            );
            assert!(matches!(
                completed.try_recv(),
                Err(mpsc::TryRecvError::Empty)
            ));
            release.send(()).unwrap();
        }
        let outcome = completed.recv_timeout(WAIT).unwrap();
        assert_eq!(outcome.termination, "exit");
        assert_eq!(outcome.exit_code, Some(0));
    });
    let evidence = sink.into_streams().unwrap();
    assert_eq!(evidence["stdout"], b"short-marker");
    assert_eq!(evidence["stderr"], b"short-marker");
}
