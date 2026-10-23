use crate::evidence::{
    canonical,
    identity::Identity,
    record::{Limits, Outcome, Record, Stream},
    store::{Opened, Publication, Store},
};
use std::{collections::BTreeMap, fs::OpenOptions, io::Write, os::unix::fs::OpenOptionsExt};

pub struct Error {
    pub status: i32,
    pub message: String,
}

impl From<String> for Error {
    fn from(message: String) -> Self {
        Self { status: 2, message }
    }
}

impl From<&str> for Error {
    fn from(message: &str) -> Self {
        message.to_owned().into()
    }
}

fn record(seed: u64) -> Result<Record, String> {
    let identity = Identity {
        target: canonical::digest(b"(module)"),
        toolchain: canonical::digest(b"foundation-toolchain-v1"),
        platform: canonical::digest(b"foundation-platform-v1"),
        implementation: canonical::digest(b"foundation-implementation-v1"),
        configuration: canonical::digest(b"foundation-configuration-v1"),
    };
    Ok(Record {
        schema: "gomad.execution-record/v2".into(),
        execution_identity: identity.digest()?,
        identity,
        profile: "gomad.foundation/v1".into(),
        seed,
        arguments: Vec::new(),
        environment: BTreeMap::new(),
        limits: Limits {
            fuel: 100_000,
            memory_bytes: 65_536,
            wall_millis: 100,
            output_bytes: 1024,
            evidence_bytes: 4096,
        },
        virtual_time_nanos: 0,
        outcome: Outcome {
            class: "success".into(),
            termination: "exit".into(),
            exit_code: Some(0),
            reason: String::new(),
        },
        streams: BTreeMap::new(),
    })
}

pub(crate) fn publication_record(seed: u64) -> Result<Publication, String> {
    let mut record = record(seed)?;
    let stdout = b"foundation-output".to_vec();
    record.streams.insert(
        "output/stdout".into(),
        Stream {
            bytes: stdout.len() as u64,
            sha256: canonical::digest(&stdout),
            records: 0,
        },
    );
    let payloads = BTreeMap::from([
        ("target/module.wasm".into(), b"(module)".to_vec()),
        ("output/stdout".into(), stdout),
        ("record.json".into(), record.bytes()?),
    ]);
    Ok(Publication { record, payloads })
}

fn bundle(root: &std::path::Path) -> Result<(), String> {
    let records = crate::slots::run(&[7, 8], 2, |seed| {
        let publication = publication_record(seed)?;
        if publication.record.bytes()? != publication_record(seed)?.record.bytes()? {
            return Err("repeated record differs".into());
        }
        Ok(publication)
    })?;
    publish_bundle(root, "foundation-record", records)
}

fn inspection_details(root: &std::path::Path, input: &std::path::Path) -> Result<(), String> {
    #[derive(serde::Deserialize)]
    #[serde(deny_unknown_fields)]
    struct Input {
        seed: u64,
        details: crate::evidence::inspection::Details,
        payloads: BTreeMap<String, Vec<u8>>,
    }
    let input: Input =
        serde_json::from_slice(&std::fs::read(input).map_err(|error| error.to_string())?)
            .map_err(|error| error.to_string())?;
    let mut publication = publication_record(input.seed)?;
    for (name, bytes) in input.payloads {
        if publication.payloads.contains_key(&name) || name == "evidence/inspection.json" {
            return Err("duplicate fixture detail payload".into());
        }
        publication.payloads.insert(name, bytes);
    }
    let bytes = canonical::encode(&input.details)?;
    publication.record.streams.insert(
        "evidence/inspection.json".into(),
        Stream {
            bytes: bytes.len() as u64,
            sha256: canonical::digest(&bytes),
            records: 1,
        },
    );
    publication
        .payloads
        .insert("evidence/inspection.json".into(), bytes);
    publication
        .payloads
        .insert("record.json".into(), publication.record.bytes()?);
    publish_bundle(root, "inspect-details-consumer", vec![publication])
}

pub(crate) fn publish_bundle(
    root: &std::path::Path,
    name: &str,
    records: Vec<Publication>,
) -> Result<(), String> {
    publish_bundle_with_budget(root, name, records, 64 << 20)
}

pub(crate) fn publish_bundle_with_budget(
    root: &std::path::Path,
    name: &str,
    records: Vec<Publication>,
    maximum_bytes: u64,
) -> Result<(), String> {
    use serde_json::json;
    std::os::unix::fs::DirBuilderExt::mode(&mut std::fs::DirBuilder::new(), 0o700)
        .create(root)
        .map_err(|error| error.to_string())?;
    let store = Store {
        root: root.join("artifacts"),
        maximum_bytes,
        target_pool: Some(root.join("pool")),
        fault: None,
    };
    let mut reports = Vec::new();
    let mut ledger = Vec::new();
    let mut previous = "0".repeat(64);
    for (ordinal, publication) in records.into_iter().enumerate() {
        let record = publication.record.clone();
        let published = store.publish(publication)?;
        let relative = published
            .path
            .strip_prefix(root)
            .map_err(|error| error.to_string())?
            .to_string_lossy();
        let mut row = json!({"ordinal":ordinal,"record_sha256":published.record_sha256,"previous_sha256":previous});
        previous = canonical::digest(&canonical::encode(&row)?);
        row["sha256"] = previous.clone().into();
        ledger.extend(canonical::encode(&row)?);
        ledger.push(b'\n');
        reports.push(json!({"record_file":format!("{relative}/record.json"),"record_sha256":published.record_sha256,"artifact":relative,"outcome":record.outcome.class,"termination":record.outcome.termination,"seed":record.seed,"failure_signature":record.failure_signature()?,"execution_identity":record.execution_identity}));
    }
    let result = json!({"schema":"gomad.foundation-result/v1","case":name,"records":reports,"ledger_file":"ledger.jsonl"});
    for (name, bytes) in [
        ("ledger.jsonl", ledger),
        ("foundation-result.json", canonical::encode(&result)?),
    ] {
        let mut file = OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(root.join(name))
            .map_err(|error| error.to_string())?;
        file.write_all(&bytes).map_err(|error| error.to_string())?;
        file.sync_all().map_err(|error| error.to_string())?;
    }
    println!("{result}");
    Ok(())
}

#[must_use]
#[expect(
    clippy::too_many_lines,
    reason = "Fixture commands share one feature-gated grammar and never enter release builds."
)]
pub fn dispatch(arguments: &[String]) -> Option<Result<(), Error>> {
    let command = arguments.first()?.as_str();
    if matches!(
        command,
        "fixture-live-runtime" | "fixture-live-broker" | "fixture-live-cluster"
    ) {
        return Some((|| {
            if arguments.len() != 5 || arguments[1] != "--root" || arguments[3] != "--input" {
                return Err("live fixture requires --root ROOT --input FILE".into());
            }
            let root = std::path::Path::new(&arguments[2]);
            let input = std::path::Path::new(&arguments[4]);
            match command {
                "fixture-live-runtime" => crate::live_runtime::run(input, root),
                "fixture-live-broker" => crate::live_broker::input(root, input),
                _ => crate::live_cluster::input(root, input),
            }
            .map_err(Error::from)
        })());
    }
    if command == "fixture-driver-run" {
        return Some(
            crate::cli::run_guest::fixture_driver_run(&arguments[1..]).map_err(Error::from),
        );
    }
    if command == "fixture-driver-prepare" {
        return Some(crate::cli::run_guest::fixture_prepare(&arguments[1..]).map_err(Error::from));
    }
    if command == "fixture-choice-implementation" {
        return Some((|| {
            if arguments.len() != 3
                || arguments[1] != "--build-key"
                || !canonical::valid_digest(&arguments[2])
            {
                return Err("fixture-choice-implementation requires --build-key SHA256".into());
            }
            let implementation = crate::campaign::prepare::choice_implementation(&arguments[2])
                .map_err(Error::from)?;
            println!(
                "{}",
                serde_json::json!({"runtime_sha256":format!("{implementation:x}"),
                "implementation_sha256":implementation.as_slice()})
            );
            Ok(())
        })());
    }
    if command == "fixture-preparation-open" {
        return Some((|| {
            if arguments.len() != 5 || arguments[1] != "--receipt" || arguments[3] != "--sha256" {
                return Err(
                    "fixture-preparation-open requires --receipt FILE --sha256 HASH".into(),
                );
            }
            let (receipt, prepared) = crate::campaign::preparation::open_receipt(
                std::path::Path::new(&arguments[2]),
                &arguments[4],
            )
            .map_err(Error::from)?;
            println!(
                "{}",
                serde_json::json!({"schema":"gomad.preparation-open-check/v1",
                "module_sha256":canonical::digest(&prepared.module),
                "provenance_sha256":canonical::digest(&prepared.provenance),
                "guest_executions":receipt.guest_executions,"record_sha256":receipt.record_sha256})
            );
            Ok(())
        })());
    }
    if command == "fixture-qualify-phases" {
        return Some(if arguments.len() == 1 {
            crate::qualify::fixture::phases().map_err(Error::from)
        } else {
            Err("fixture-qualify-phases takes no arguments".into())
        });
    }
    if matches!(
        command,
        "fixture-simulation-controller"
            | "fixture-choice-controller"
            | "fixture-minimize-controller"
            | "fixture-qualify-accounting"
    ) {
        return Some((|| {
            if arguments.len() != 3 || arguments[1] != "--input" {
                return Err("fixture-simulation-controller requires --input FILE".into());
            }
            let path = std::path::Path::new(&arguments[2]);
            if command == "fixture-qualify-accounting" {
                crate::qualify::fixture::run(path).map_err(Error::from)
            } else if command == "fixture-minimize-controller" {
                crate::minimize::fixture::run(path).map_err(Error::from)
            } else if command == "fixture-choice-controller" {
                crate::campaign::exploration::choice_corpus::run(path).map_err(Error::from)
            } else {
                crate::campaign::exploration::simulation_corpus::run(path).map_err(Error::from)
            }
        })());
    }
    if !matches!(
        command,
        "fixture-execution-bundle"
            | "fixture-replay-role"
            | "fixture-replay-cluster-execution"
            | "fixture-replay-cluster-input"
            | "fixture-replay-storage-prefix"
            | "fixture-replay-input"
            | "fixture-replay-divergence"
            | "fixture-replay-pinned"
            | "fixture-replay-execution"
            | "fixture-inspect-execution"
            | "fixture-inspect-details"
            | "fixture-watchdog-cancel"
            | "fixture-bundle"
            | "fixture-record"
            | "fixture-publish"
            | "fixture-open"
            | "fixture-pinned"
            | "fixture-root-substitution"
            | "fixture-payload-substitution"
            | "fixture-payload-mutation"
            | "fixture-bound"
    ) {
        return None;
    }
    Some((|| {
        if command == "fixture-replay-storage-prefix" {
            if arguments.len() != 5 || arguments[1] != "--root" || arguments[3] != "--input" {
                return Err("fixture-replay-storage-prefix --root ROOT --input FILE".into());
            }
            return crate::fixture_storage::run(
                std::path::Path::new(&arguments[2]),
                std::path::Path::new(&arguments[4]),
            )
            .map_err(Into::into);
        }
        if command == "fixture-replay-cluster-input" {
            if arguments.len() != 5 || arguments[1] != "--root" || arguments[3] != "--input" {
                return Err("fixture-replay-cluster-input --root ROOT --input FILE".into());
            }
            return crate::fixture_cluster::input(
                std::path::Path::new(&arguments[2]),
                std::path::Path::new(&arguments[4]),
            )
            .map_err(Into::into);
        }
        if command == "fixture-replay-role" {
            if !matches!(arguments.len(), 5 | 7)
                || arguments[1] != "--root"
                || arguments[3] != "--corpus"
                || (arguments.len() == 7 && arguments[5] != "--role")
            {
                return Err("fixture-replay-role --root ROOT --corpus CORPUS [--role ROLE]".into());
            }
            return crate::fixture_role::run(
                std::path::Path::new(&arguments[2]),
                std::path::Path::new(&arguments[4]),
                arguments.get(6).map_or("ordinary", String::as_str),
            )
            .map_err(Into::into);
        }
        if command == "fixture-replay-cluster-execution" {
            if arguments.len() != 5 || arguments[1] != "--root" || arguments[3] != "--corpus" {
                return Err("fixture-replay-cluster-execution --root ROOT --corpus CORPUS".into());
            }
            return crate::fixture_cluster::bundle(
                std::path::Path::new(&arguments[2]),
                std::path::Path::new(&arguments[4]),
            )
            .map_err(Into::into);
        }
        if command == "fixture-replay-input" {
            if !matches!(arguments.len(), 5 | 7)
                || arguments[1] != "--root"
                || arguments[3] != "--input"
                || (arguments.len() == 7 && arguments[5] != "--role")
            {
                return Err("fixture-replay-input --root ROOT --input FILE [--role ROLE]".into());
            }
            return crate::fixture_replay::bundle(
                std::path::Path::new(&arguments[2]),
                std::path::Path::new(&arguments[4]),
                arguments.get(6).map_or("ordinary", String::as_str),
            )
            .map_err(Into::into);
        }
        if command == "fixture-replay-divergence" {
            if arguments.len() != 6 || arguments[2] != "--root" || arguments[4] != "--trace" {
                return Err("fixture-replay-divergence ARTIFACT --root ROOT --trace FILE".into());
            }
            return crate::fixture_replay::divergence(
                std::path::Path::new(&arguments[1]),
                std::path::Path::new(&arguments[3]),
                std::path::Path::new(&arguments[5]),
            )
            .map_err(Into::into);
        }
        if command == "fixture-replay-pinned" {
            if arguments.len() != 2 {
                return Err("fixture-replay-pinned ARTIFACT".into());
            }
            let path = std::path::Path::new(&arguments[1]);
            let opened = Opened::open(path)?;
            std::fs::rename(path, path.with_extension("pinned-original"))
                .map_err(|error| error.to_string())?;
            std::os::unix::fs::DirBuilderExt::mode(&mut std::fs::DirBuilder::new(), 0o700)
                .create(path)
                .map_err(|error| error.to_string())?;
            let prepared = crate::replay::preflight::prepare(opened)?;
            let status =
                crate::cli::replay::run_prepared(prepared, None, &mut std::io::stdout().lock())
                    .map_err(|error| Error {
                        status: error.status,
                        message: error.message,
                    })?;
            if status != 0 {
                return Err("pinned replay did not reproduce success".into());
            }
            return Ok(());
        }
        if command == "fixture-inspect-details" {
            if arguments.len() != 5 || arguments[1] != "--root" || arguments[3] != "--input" {
                return Err("fixture-inspect-details --root ROOT --input FILE".into());
            }
            return inspection_details(
                std::path::Path::new(&arguments[2]),
                std::path::Path::new(&arguments[4]),
            )
            .map_err(Into::into);
        }
        if command == "fixture-watchdog-cancel" {
            if arguments.len() != 3 || arguments[1] != "--corpus" {
                return Err("fixture-watchdog-cancel --corpus CORPUS".into());
            }
            return crate::fixture_execute::watchdog_cancellation(std::path::Path::new(
                &arguments[2],
            ))
            .map_err(Into::into);
        }
        if matches!(
            command,
            "fixture-execution-bundle" | "fixture-inspect-execution" | "fixture-replay-execution"
        ) {
            if command == "fixture-inspect-execution"
                && arguments.len() == 5
                && arguments[1] == "--root"
                && arguments[3] == "--input"
            {
                return crate::fixture_execute::inspection_input(
                    std::path::Path::new(&arguments[2]),
                    std::path::Path::new(&arguments[4]),
                )
                .map_err(Into::into);
            }
            if arguments.len() != 5 || arguments[1] != "--root" || arguments[3] != "--corpus" {
                return Err("fixture-execution-bundle --root ROOT --corpus CORPUS".into());
            }
            let bundle = if command == "fixture-replay-execution" {
                crate::fixture_execute::replay_bundle
            } else if command == "fixture-inspect-execution" {
                crate::fixture_execute::inspection_bundle
            } else {
                crate::fixture_execute::bundle
            };
            return bundle(
                std::path::Path::new(&arguments[2]),
                std::path::Path::new(&arguments[4]),
            )
            .map_err(Into::into);
        }
        if command == "fixture-bundle" {
            if arguments.len() != 3 || arguments[1] != "--root" {
                return Err("fixture-bundle --root ROOT".into());
            }
            return bundle(std::path::Path::new(&arguments[2])).map_err(Into::into);
        }
        if matches!(
            command,
            "fixture-payload-substitution" | "fixture-payload-mutation"
        ) {
            if arguments.len() != 2 {
                return Err("fixture-payload-mutation ARTIFACT".into());
            }
            return Opened::fixture_payload_mutation(
                std::path::Path::new(&arguments[1]),
                command == "fixture-payload-substitution",
            )
            .map_err(Into::into);
        }
        if matches!(
            command,
            "fixture-open" | "fixture-pinned" | "fixture-root-substitution" | "fixture-bound"
        ) {
            if arguments.len() != 2 && !(arguments.len() == 4 && arguments[2] == "--pool") {
                return Err("fixture-open ARTIFACT".into());
            }
            let path = std::path::Path::new(&arguments[1]);
            let opened = if command == "fixture-root-substitution" {
                Opened::fixture_root_substitution(path)?
            } else if arguments.len() == 4 {
                Opened::open_with_pool(path, std::path::Path::new(&arguments[3]))?
            } else {
                Opened::open(path)?
            };
            if command == "fixture-bound" {
                opened.payload("output/stdout", 1)?;
            }
            if command == "fixture-pinned" {
                let moved = path.with_file_name(format!(
                    "{}-moved",
                    path.file_name()
                        .ok_or("artifact filename")?
                        .to_string_lossy()
                ));
                std::fs::rename(path, moved).map_err(|error| error.to_string())?;
                std::fs::create_dir(path).map_err(|error| error.to_string())?;
            }
            opened.authenticate_all()?;
            let bytes = opened.payload("record.json", 16 << 20)?;
            if Record::decode(&bytes)? != opened.manifest().record {
                return Err("record payload mismatch".into());
            }
            println!(
                "{}",
                serde_json::json!({"record_sha256": opened.manifest().record_sha256, "stored_bytes": opened.stored_bytes()})
            );
            return Ok(());
        }
        if command == "fixture-publish" {
            let mut options = BTreeMap::new();
            for pair in arguments[1..].chunks(2) {
                if pair.len() != 2
                    || !["--root", "--seed", "--budget", "--pool", "--fault"]
                        .contains(&pair[0].as_str())
                    || options.insert(pair[0].as_str(), pair[1].as_str()).is_some()
                {
                    return Err("invalid fixture-publish option".into());
                }
            }
            let root = options.get("--root").ok_or("missing root")?;
            let seed = options
                .get("--seed")
                .ok_or("missing seed")?
                .parse()
                .map_err(|_| "invalid seed")?;
            let store = Store {
                root: (*root).into(),
                maximum_bytes: options
                    .get("--budget")
                    .unwrap_or(&"0")
                    .parse()
                    .map_err(|_| "invalid budget")?,
                target_pool: options.get("--pool").map(std::path::PathBuf::from),
                fault: options.get("--fault").map(|value| (*value).into()),
            };
            let published = store
                .publish(publication_record(seed)?)
                .map_err(|message| Error { status: 3, message })?;
            println!(
                "{}",
                serde_json::json!({"artifact": published.path, "record_sha256": published.record_sha256, "stored_bytes": published.stored_bytes})
            );
            return Ok(());
        }
        if !matches!(arguments.len(), 5 | 7)
            || arguments[1] != "--output"
            || arguments[3] != "--seed"
        {
            return Err("fixture-record --output FILE --seed N".into());
        }
        let seed = arguments[4].parse().map_err(|_| "invalid seed")?;
        let mut record = record(seed)?;
        if arguments.len() == 7 {
            if arguments[5] != "--fault" {
                return Err("invalid fixture-record option".into());
            }
            record.outcome.reason = match arguments[6].as_str() {
                "timestamp" => std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .map_err(|error| error.to_string())?
                    .as_nanos()
                    .to_string(),
                "path" => arguments[2].clone(),
                _ => return Err("unknown record fault".into()),
            };
        }
        let bytes = record.bytes()?;
        let mut file = OpenOptions::new()
            .write(true)
            .create_new(true)
            .mode(0o600)
            .open(&arguments[2])
            .map_err(|error| error.to_string())?;
        file.write_all(&bytes).map_err(|error| error.to_string())?;
        println!(
            "{}",
            serde_json::json!({"record_sha256": record.hash()?, "failure_signature": record.failure_signature()?})
        );
        Ok(())
    })())
}

pub(crate) fn execution_started() -> Result<(), crate::cli::Error> {
    let Some(path) = std::env::var_os("GOMAD_FIXTURE_GUEST_START_FILE") else {
        return Ok(());
    };
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|error| error.to_string())?;
    file.write_all(b"linked execution entered\n")
        .map_err(|error| error.to_string())?;
    Ok(())
}

static BACKSTOP_WINDOWS: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);

/// Numbers armed backstop windows in this process (fixture builds only).
pub(crate) fn backstop_window() -> u64 {
    BACKSTOP_WINDOWS.fetch_add(1, std::sync::atomic::Ordering::SeqCst)
}

/// Fixture-only stand-in for a native compiler or store teardown that never
/// returns: `GOMAD_FIXTURE_BACKSTOP_STALL=compile|teardown` stalls armed window
/// `GOMAD_FIXTURE_BACKSTOP_STALL_WINDOW` (default 0) forever, after recording
/// `GOMAD_FIXTURE_BACKSTOP_STALL_FILE` when set. Only the deadline backstop
/// can end the process from there.
pub(crate) fn backstop_stall(phase: &str, window: u64) {
    if std::env::var("GOMAD_FIXTURE_BACKSTOP_STALL")
        .ok()
        .as_deref()
        != Some(phase)
    {
        return;
    }
    let selected = std::env::var("GOMAD_FIXTURE_BACKSTOP_STALL_WINDOW")
        .ok()
        .and_then(|value| value.parse::<u64>().ok())
        .unwrap_or(0);
    if selected != window {
        return;
    }
    if let Some(path) = std::env::var_os("GOMAD_FIXTURE_BACKSTOP_STALL_FILE") {
        let _ = std::fs::write(path, format!("{phase} {window}\n"));
    }
    loop {
        std::thread::park();
    }
}
