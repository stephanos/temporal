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

fn publication_record(seed: u64) -> Result<Publication, String> {
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

pub(crate) fn publish_bundle(
    root: &std::path::Path,
    name: &str,
    records: Vec<Publication>,
) -> Result<(), String> {
    use serde_json::json;
    std::os::unix::fs::DirBuilderExt::mode(&mut std::fs::DirBuilder::new(), 0o700)
        .create(root)
        .map_err(|error| error.to_string())?;
    let store = Store {
        root: root.join("artifacts"),
        maximum_bytes: 1 << 20,
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
    if !matches!(
        command,
        "fixture-execution-bundle"
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
        if command == "fixture-watchdog-cancel" {
            if arguments.len() != 3 || arguments[1] != "--corpus" {
                return Err("fixture-watchdog-cancel --corpus CORPUS".into());
            }
            return crate::fixture_execute::watchdog_cancellation(std::path::Path::new(
                &arguments[2],
            ))
            .map_err(Into::into);
        }
        if command == "fixture-execution-bundle" {
            if arguments.len() != 5 || arguments[1] != "--root" || arguments[3] != "--corpus" {
                return Err("fixture-execution-bundle --root ROOT --corpus CORPUS".into());
            }
            return crate::fixture_execute::bundle(
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
