use crate::abi::Operation;
use crate::model::{Config, Model};
use crate::namespace::{self, Image, ImageEntry, Limits, Namespace, Open, Operation as N, Output};
use crate::transcript::Subscription;
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};

const EPOCH: u64 = 946_684_800_000_000_000;

fn config() -> Config {
    let mut config: Config = serde_json::from_value(json!({
        "seed":7,"mode":1,"limit":4096,"diagnostics":false,
        "entropy_key":[1,2,3,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],
        "working_directory_sha256":format!("{:x}",Sha256::digest(b"/workspace")),
        "arguments":{"bytes":0,"count":0,"chunks":0},
        "environment":{"bytes":0,"count":0,"chunks":0},
        "epoch_nanos":EPOCH,"read_step_nanos":1000,"now":0,
        "limits":{"calls":10000,"transcript_bytes":8388608,"pending_events":4096}
    }))
    .unwrap();
    config
        .set_inputs(
            &["guest".into(), "literal $arg".into()],
            &[
                "PWD=/workspace".into(),
                "TZ=UTC".into(),
                "ONLY=captured".into(),
            ],
        )
        .unwrap();
    config
}

fn image() -> Image {
    Image {
        entries: Vec::new(),
        writable_directories: vec![b"/tmp".to_vec(), b"/workspace".to_vec()],
        preopens: vec![b"/tmp".to_vec(), b"/workspace".to_vec()],
        working_directory: b"/tmp".to_vec(),
        stdin: Vec::new(),
        limits: Limits {
            files: 100,
            descriptors: 100,
            filesystem_bytes: 65536,
            output_bytes: 65536,
        },
        epoch_nanos: EPOCH,
    }
}

fn opened(namespace: &mut Namespace, name: &[u8], oflags: u16, rights: u64) -> u32 {
    let reply = namespace
        .execute(
            N::Open(Open {
                fd: 3,
                path: name.to_vec(),
                dirflags: 0,
                oflags,
                base: rights,
                inheriting: 0,
                flags: 0,
            }),
            0,
        )
        .unwrap();
    assert_eq!(reply.errno, 0);
    let Some(Output::Fd(fd)) = reply.output else {
        panic!("open returned no fd")
    };
    fd
}

fn output(namespace: &mut Namespace, operation: N) -> Output {
    let reply = namespace.execute(operation, 0).unwrap();
    assert_eq!((reply.errno, reply.unsupported), (0, None));
    reply.output.unwrap()
}

fn bytes(output: Output) -> Vec<u8> {
    let Output::Bytes(data) = output else {
        panic!("expected bytes")
    };
    data
}

fn transcript(model: &mut Model) -> Vec<u8> {
    let mut wire = Vec::new();
    model.flush(&mut wire, true).unwrap();
    let mut input = std::io::Cursor::new(wire);
    let mut bytes = Vec::new();
    while input.position() < input.get_ref().len() as u64 {
        let frame: Value =
            serde_json::from_slice(&crate::protocol::read_frame(&mut input).unwrap()).unwrap();
        if frame["stream"] == "transcript" {
            bytes.extend(
                STANDARD
                    .decode(frame["data_base64"].as_str().unwrap())
                    .unwrap(),
            );
        }
    }
    bytes
}

#[test]
fn captured_arguments_environment_and_entropy_preserve_original_values_and_chunking() {
    let mut original = config();
    let mut model = Model::new(original.clone());
    original
        .set_inputs(&["mutated".into()], &["PWD=/workspace".into()])
        .unwrap();
    let (args, offsets) = model.strings(1, Operation::ArgsGet).unwrap();
    assert_eq!(args, b"guest\0literal $arg\0");
    assert_eq!(offsets, [0, 6]);
    let (sizes, offsets) = model.strings(2, Operation::ArgsSizesGet).unwrap();
    assert_eq!((offsets.len(), sizes.len()), (2, 19));
    let (environment, offsets) = model.strings(3, Operation::EnvironGet).unwrap();
    assert_eq!(environment, b"PWD=/workspace\0TZ=UTC\0ONLY=captured\0");
    assert_eq!(offsets, [0, 15, 22]);
    let first = model.random(4, 7).unwrap();
    let second = model.random(5, 57).unwrap();
    let mut fresh = Model::new(config());
    assert_eq!(
        [first.as_slice(), second.as_slice()].concat(),
        fresh.random(1, 64).unwrap()
    );
    assert_ne!(first, second[..7]);
    assert_eq!(
        model.strings(6, Operation::ArgsGet).unwrap(),
        (args, vec![0, 6])
    );
}

#[test]
fn environment_rejects_original_ambiguous_configuration_variants() {
    for (arguments, environment) in [
        (vec!["guest".into()], vec!["PWD=/other".into()]),
        (
            vec!["guest".into()],
            vec![
                "PWD=/workspace".into(),
                "ONLY=captured".into(),
                "ONLY=again".into(),
            ],
        ),
        (vec!["bad\0arg".into()], vec!["PWD=/workspace".into()]),
    ] {
        assert!(config().set_inputs(&arguments, &environment).is_err());
    }
}

#[test]
fn transcript_clock_run_preserves_original_literal_digest_modes_and_pending_inspection() {
    let mut expected = b"GMWT\x01\0\0\0".to_vec();
    expected.extend_from_slice(&[33, 0, 0, 0]);
    expected.extend_from_slice(&1_u64.to_le_bytes());
    expected.extend_from_slice(&28_u32.to_le_bytes());
    expected.extend_from_slice(&1_u32.to_le_bytes());
    expected.extend_from_slice(&7_u64.to_le_bytes());
    expected.extend_from_slice(&0_u64.to_le_bytes());
    expected.extend_from_slice(&3_u64.to_le_bytes());
    for digest_only in [false, true] {
        let mut config = config();
        config.digest_only = digest_only;
        let mut model = Model::new(config);
        for id in 1..=3 {
            model.clock_read(id, 1, 7).unwrap();
        }
        assert_eq!(
            model.metadata(3)["transcript_sha256"],
            format!("{:x}", Sha256::digest(&expected))
        );
        assert_eq!(model.transcript_bytes, 52);
        assert_eq!(model.inspection_cut().clock_run, &expected[8..]);
        model.clock_read(4, 1, 7).unwrap();
        assert_eq!(model.transcript_bytes, 52);
        let mut fourth = expected.clone();
        fourth[44..52].copy_from_slice(&4_u64.to_le_bytes());
        assert_eq!(
            model.metadata(4)["transcript_sha256"],
            format!("{:x}", Sha256::digest(&fourth))
        );
        assert_eq!(
            transcript(&mut model),
            if digest_only { Vec::new() } else { fourth }
        );
    }
}

fn clock(userdata: u64, clock_id: u32, timeout: u64, flags: u16) -> Subscription {
    Subscription {
        userdata,
        kind: 0,
        clock_id,
        timeout,
        precision: 0,
        flags,
        fd: 0,
    }
}

#[test]
fn exploratory_clock_and_mixed_poll_preserve_original_virtual_time_and_event_order() {
    let mut model = Model::new(config());
    let namespace = Namespace::new(image()).unwrap();
    assert_eq!(model.clock_read(1, 0, 0).unwrap(), Some(EPOCH));
    assert_eq!(model.clock_read(2, 1, 0).unwrap(), Some(1000));
    assert_eq!(model.now, 2000);
    model.yield_call(3).unwrap();
    let writable = Subscription {
        userdata: 7,
        kind: 2,
        fd: 1,
        ..clock(7, 1, 0, 0)
    };
    let result = model
        .poll_owned_call(4, &[clock(7, 1, 0, 0), writable], Some(&namespace))
        .unwrap();
    assert_eq!(
        (
            model.now,
            result.errno,
            result.unsupported,
            result.events.len()
        ),
        (2000, 0, "", 2)
    );
    assert_eq!(
        result
            .events
            .iter()
            .map(|e| (e.userdata, e.kind))
            .collect::<Vec<_>>(),
        [(7, 0), (7, 2)]
    );
    let mut relative = clock(8, 1, 3000, 0);
    relative.precision = 1;
    let result = model
        .poll_owned_call(
            5,
            &[relative, clock(9, 0, EPOCH + 5000, 1)],
            Some(&namespace),
        )
        .unwrap();
    assert_eq!((model.now, result.errno, result.events.len()), (5000, 0, 2));
    assert_eq!(
        result.events.iter().map(|e| e.userdata).collect::<Vec<_>>(),
        [8, 9]
    );
    let bad = Subscription {
        userdata: 1,
        kind: 1,
        fd: 999,
        ..clock(1, 1, 0, 0)
    };
    let result = model.poll_owned_call(6, &[bad], Some(&namespace)).unwrap();
    assert_eq!(
        (model.now, result.events.len(), result.events[0].errno),
        (5000, 1, namespace::BADF)
    );
    let result = model
        .poll_owned_call(7, &[clock(0, 1, u64::MAX, 0)], Some(&namespace))
        .unwrap();
    assert_eq!((model.now, result.errno), (5000, namespace::INVAL));
    assert_eq!(model.clock_read(8, 2, 0).unwrap(), None);
}

#[test]
fn namespace_rights_offsets_cookies_containment_and_closure_preserve_original_sequence() {
    let mut namespace = Namespace::new(image()).unwrap();
    assert_eq!(output(&mut namespace, N::PrestatGet(3)), Output::Prestat(4));
    assert_eq!(
        output(&mut namespace, N::PrestatDirName { fd: 3, length: 4 }),
        Output::Bytes(b"/tmp".to_vec())
    );
    assert_eq!(
        namespace.execute(N::PrestatGet(5), 0).unwrap().errno,
        namespace::BADF
    );
    let fd = opened(&mut namespace, b"b", 1, namespace::RIGHTS_ALL);
    assert_eq!(
        output(
            &mut namespace,
            N::Write {
                fd,
                data: b"xyz".to_vec(),
                offset: Some(2)
            }
        ),
        Output::Written(3)
    );
    assert_eq!(
        output(
            &mut namespace,
            N::Read {
                fd,
                length: 20,
                offset: None
            }
        ),
        Output::Bytes(b"\0\0xyz".to_vec())
    );
    assert_eq!(
        output(
            &mut namespace,
            N::Read {
                fd,
                length: 20,
                offset: None
            }
        ),
        Output::Bytes(Vec::new())
    );
    assert_eq!(
        output(
            &mut namespace,
            N::Seek {
                fd,
                offset: -2,
                whence: 2
            }
        ),
        Output::Offset(3)
    );
    assert_eq!(
        namespace
            .execute(
                N::Seek {
                    fd,
                    offset: -6,
                    whence: 2
                },
                0
            )
            .unwrap()
            .errno,
        namespace::INVAL
    );
    let readonly = opened(&mut namespace, b"b", 0, namespace::READ);
    assert_eq!(
        namespace
            .execute(
                N::Write {
                    fd: readonly,
                    data: b"x".to_vec(),
                    offset: None
                },
                0
            )
            .unwrap()
            .errno,
        namespace::NOTCAPABLE
    );
    opened(&mut namespace, b"a", 1, namespace::READ);
    let first = bytes(output(
        &mut namespace,
        N::Readdir {
            fd: 3,
            length: 25,
            cookie: 0,
        },
    ));
    assert_eq!(first.len(), 25);
    assert_eq!(&first[24..], b"a");
    assert_eq!(u64::from_le_bytes(first[..8].try_into().unwrap()), 1);
    let next = bytes(output(
        &mut namespace,
        N::Readdir {
            fd: 3,
            length: 100,
            cookie: 1,
        },
    ));
    assert_eq!(&next[24..], b"b");
    assert_eq!(u64::from_le_bytes(next[..8].try_into().unwrap()), 2);
    assert_eq!(
        namespace
            .execute(
                N::Readdir {
                    fd: 3,
                    length: 1,
                    cookie: 9
                },
                0
            )
            .unwrap()
            .errno,
        namespace::INVAL
    );
    for path in [
        b"../escape".as_slice(),
        b"/absolute",
        b"a/../../escape",
        b"bad\0name",
    ] {
        assert_eq!(
            namespace
                .execute(
                    N::CreateDirectory {
                        fd: 3,
                        path: path.to_vec()
                    },
                    0
                )
                .unwrap()
                .errno,
            namespace::NOTCAPABLE,
            "{path:?}"
        );
    }
    assert_eq!(output(&mut namespace, N::Close(fd)), Output::Empty);
    assert_eq!(
        namespace.execute(N::Close(fd), 0).unwrap().errno,
        namespace::BADF
    );
    for operation in [
        N::Unsupported("sock_accept".into()),
        N::Symlink {
            old_path: b"x".to_vec(),
            fd: 3,
            path: b"y".to_vec(),
        },
    ] {
        assert!(
            namespace
                .execute(operation, 0)
                .unwrap()
                .unsupported
                .is_some()
        );
    }
}

#[test]
fn captured_namespace_owns_payload_and_infers_absence_without_host_access() {
    let mut captured = image();
    captured.preopens = vec![b"/fixtures".to_vec(), b"/tmp".to_vec()];
    captured.entries = vec![
        ImageEntry {
            path: b"/fixtures".to_vec(),
            directory: true,
            readonly: true,
            data: Vec::new(),
        },
        ImageEntry {
            path: b"/fixtures/input".to_vec(),
            directory: false,
            readonly: true,
            data: b"frozen".to_vec(),
        },
    ];
    let mut namespace = Namespace::new(captured.clone()).unwrap();
    captured.entries[1].data.fill(0);
    captured.entries.clear();
    let fd = opened(&mut namespace, b"input", 0, namespace::READ);
    assert_eq!(
        output(
            &mut namespace,
            N::Read {
                fd,
                length: 20,
                offset: None
            }
        ),
        Output::Bytes(b"frozen".to_vec())
    );
    assert_eq!(
        namespace
            .execute(
                N::PathStat {
                    fd: 3,
                    path: b"missing".to_vec(),
                    flags: 0
                },
                0
            )
            .unwrap()
            .errno,
        namespace::NOENT
    );
    assert_eq!(
        namespace
            .execute(
                N::CreateDirectory {
                    fd: 3,
                    path: b"new".to_vec()
                },
                0
            )
            .unwrap()
            .errno,
        namespace::ROFS
    );
}

#[test]
fn network_v2_rejects_original_retired_profile_before_state_construction() {
    let mut config: crate::cluster::network::ModelV2 = serde_json::from_value(json!({
        "profile":"gomad.simulation-network-stream/v3",
        "limits":{"listeners":128,"connections":4096,"bytes":134217728,
            "connection_bytes":4194304,"stall_nanos":10000000000_u64,"transitions":1048576},
        "faults":[]
    }))
    .unwrap();
    crate::cluster::network::validate_model(&config).unwrap();
    config.profile = "gomad.simulation-network-stream/v2".into();
    assert_eq!(
        crate::cluster::network::validate_model(&config),
        Err(crate::cluster::network::Failure::Invalid(
            "network v2 profile"
        ))
    );
}

#[test]
fn transcript_clock_run_retains_original_51_52_byte_capacity_boundary() {
    for limit in [51, 52] {
        let mut config = config();
        config.limits.transcript_bytes = limit;
        let mut model = Model::new(config);
        let result = model.clock_read(1, 1, 0);
        if limit == 51 {
            assert!(result.is_err());
            assert_eq!((model.now, model.transcript_bytes), (0, 0));
            assert!(transcript(&mut model).is_empty());
        } else {
            result.unwrap();
            model.clock_read(2, 1, 0).unwrap();
            assert_eq!((model.now, model.transcript_bytes), (2000, 52));
            assert_eq!(transcript(&mut model).len(), 52);
        }
    }
}

#[test]
fn namespace_sync_emits_original_versioned_binary_transcript() {
    let mut model = Model::new(config());
    let mut namespace = Namespace::new(image()).unwrap();
    let reply = model
        .namespace_call(1, &mut namespace, namespace::call::Call::new(N::Sync(3)))
        .unwrap();
    assert_eq!(reply.errno, 0);
    assert_eq!(
        transcript(&mut model),
        b"GMWT\x01\0\0\0\x0d\0\0\0\x01\0\0\0\0\0\0\0\x04\0\0\0\x03\0\0\0"
    );
}
