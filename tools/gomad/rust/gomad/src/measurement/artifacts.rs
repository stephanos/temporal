use super::{build::CommandObservation, evidence::ReadLimits};
use crate::evidence::{canonical, store::open};
use serde_json::{Value, json};
use std::{collections::BTreeMap, fs::File, io::Write, path::Path};

const MEANINGS: [(&str, &str, &str); 5] = [
    ("canonical-wasi", "-canonical.bin", "wasi"),
    ("choice", "-choice.bin", "choice"),
    ("network", "-network.jsonl", "network"),
    ("guest-stdout", "-guest-stdout.txt", "stdout"),
    ("guest-stderr", "-guest-stderr.txt", "stderr"),
];

struct Inventory {
    root: File,
    limits: ReadLimits,
    remaining: u64,
    files: BTreeMap<String, Value>,
}
impl Inventory {
    fn read(&mut self, name: &str, bound: u64) -> Result<Option<Vec<u8>>, String> {
        open::components(name)?;
        match rustix::fs::statat(&self.root, name, rustix::fs::AtFlags::SYMLINK_NOFOLLOW) {
            Err(rustix::io::Errno::NOENT) => return Ok(None),
            Err(error) => return Err(error.to_string()),
            Ok(_) => {}
        }
        let bytes = open::read(&self.root, name, bound.min(self.remaining), false)?;
        self.remaining = self
            .remaining
            .checked_sub(bytes.len() as u64)
            .ok_or("variant artifact inventory exceeds total bound")?;
        self.files.insert(
            name.into(),
            json!({"file":name,"bytes":bytes.len(),"sha256":canonical::digest(&bytes)}),
        );
        Ok(Some(bytes))
    }
    fn json(&mut self, name: &str) -> Result<Option<Value>, String> {
        self.read(name, self.limits.json_bytes)?
            .map(|bytes| crate::qualify::spec::strict(&bytes))
            .transpose()
    }
    fn declared(&mut self, name: &str, declaration: &Value) -> Result<Value, String> {
        let size = declaration["bytes"]
            .as_u64()
            .ok_or("variant stream size absent")?;
        let digest = declaration["sha256"]
            .as_str()
            .filter(|digest| canonical::valid_digest(digest))
            .ok_or("variant stream digest absent or malformed")?;
        if size > self.limits.stream_bytes {
            return Err("variant stream exceeds retained stream bound".into());
        }
        let bytes = self
            .read(name, size)?
            .ok_or_else(|| format!("declared variant stream absent: {name}"))?;
        if bytes.len() as u64 != size || canonical::digest(&bytes) != digest {
            return Err(format!("variant stream size or digest mismatch: {name}"));
        }
        Ok(self.files[name].clone())
    }
    fn hash_only_wasi(
        &mut self,
        prefix: &str,
        request_name: &str,
        request_sha256: &str,
        declaration: &Value,
        outcome: &Value,
    ) -> Result<Value, String> {
        let request = self
            .json(request_name)?
            .ok_or("digest-only WASI request absent")?;
        if self.files[request_name]["sha256"] != request_sha256
            || request["kind"] != "standalone"
            || request["guest"]["model"]["configuration"]["digest_only"] != true
            || declaration
                .as_object()
                .is_none_or(|fields| fields.len() != 4 || !fields.contains_key("file"))
            || !declaration["file"].is_null()
            || declaration["bytes"]
                .as_u64()
                .is_none_or(|bytes| bytes > 1 << 30)
            || !declaration["sha256"]
                .as_str()
                .is_some_and(canonical::valid_digest)
            || outcome["local_model"]["transcript_bytes"] != declaration["bytes"]
            || outcome["local_model"]["transcript_sha256"] != declaration["sha256"]
        {
            return Err("digest-only WASI request/outcome commitment mismatch".into());
        }
        for name in ["transcript", "wasi"] {
            match rustix::fs::statat(
                &self.root,
                format!("{prefix}{name}"),
                rustix::fs::AtFlags::SYMLINK_NOFOLLOW,
            ) {
                Err(rustix::io::Errno::NOENT) => {}
                Err(error) => return Err(error.to_string()),
                Ok(_) => {
                    return Err("digest-only WASI conflicts with a retained raw payload".into());
                }
            }
        }
        Ok(
            json!({"status":"digest-only","file":null,"bytes":declaration["bytes"],
            "sha256":declaration["sha256"],"source":"outcome:local_model.transcript_sha256"}),
        )
    }
    fn closed_streams(
        &mut self,
        prefix: &str,
        request_name: &str,
        request_sha256: Option<&str>,
        report: Option<&Value>,
    ) -> Result<Option<BTreeMap<String, Value>>, String> {
        let catalog_name = format!("{prefix}canonical-streams.json");
        let Some(catalog) = self.json(&catalog_name)? else {
            return Ok(None);
        };
        if catalog["schema"] != "gomad.native-canonical-streams/v1"
            || catalog["source"]
                != "finalized linked Execution.evidence from closed Sink::into_streams"
            || catalog["transformation"] != "identity bytes; no content normalization"
            || !catalog["streams"].is_object()
        {
            return Err(
                "variant canonical custody source is not a finalized linked Execution".into(),
            );
        }
        let request_sha256 = request_sha256.ok_or("variant request binding unavailable")?;
        if report.is_none_or(|report| {
            report["schema"] != "gomad.linked-profile-run/v1"
                || report["request_sha256"] != request_sha256
        }) {
            return Err("variant report does not authenticate the observed request".into());
        }
        let native_name = format!("{prefix}streams.json");
        let outcome_name = format!("{prefix}outcome.json");
        let native = self
            .json(&native_name)?
            .ok_or("variant native stream catalogue absent")?;
        let outcome = self
            .json(&outcome_name)?
            .ok_or("variant finalized outcome absent")?;
        if outcome["type"] != "result"
            || outcome["termination"].as_str().is_none_or(str::is_empty)
            || outcome["message"].as_str().is_none()
        {
            return Err("variant finalized outcome is malformed".into());
        }
        if self.files[&native_name]["sha256"] != catalog["native_streams_sha256"]
            || self.files[&outcome_name]["sha256"] != catalog["outcome_sha256"]
            || !native.is_object()
        {
            return Err("variant canonical custody metadata digest mismatch".into());
        }
        let mut entries = BTreeMap::new();
        for (meaning, suffix, name) in MEANINGS {
            let Some(declaration) = catalog["streams"].get(name) else {
                entries.insert(
                    meaning.into(),
                    unavailable(suffix, "finalized Execution did not emit this stream"),
                );
                continue;
            };
            let source = declaration["source"]
                .as_str()
                .ok_or("variant canonical stream source absent")?;
            if name == "wasi" && source == "outcome:local_model.transcript_sha256" {
                if native.get("transcript").is_some() || native.get("wasi").is_some() {
                    return Err("digest-only WASI conflicts with native raw declarations".into());
                }
                let mut entry = self.hash_only_wasi(
                    prefix,
                    request_name,
                    request_sha256,
                    declaration,
                    &outcome,
                )?;
                entry["original_suffix"] = json!(suffix);
                entry["canonical_catalog_sha256"] = self.files[&catalog_name]["sha256"].clone();
                entries.insert(meaning.into(), entry);
                continue;
            }
            if declaration["file"] != name {
                return Err("variant canonical stream aliases another payload".into());
            }
            if source == format!("native:{name}") {
                same_declaration(declaration, &native[name])?;
            } else if source == format!("closed-sink-empty:{name}")
                && matches!(name, "stdout" | "stderr" | "network")
            {
                if native.get(name).is_some()
                    || declaration["bytes"] != 0
                    || declaration["sha256"] != canonical::digest(&[])
                {
                    return Err("variant empty stream lacks closed-sink absence proof".into());
                }
            } else if name == "wasi" && source == "binary-transcript-alias:transcript" {
                same_declaration(declaration, &native["transcript"])?;
                self.declared(&format!("{prefix}transcript"), &native["transcript"])?;
                if let Some(wasi) = native.get("wasi") {
                    same_declaration(declaration, wasi)?;
                }
            } else {
                return Err("variant canonical stream has an unsupported custody source".into());
            }
            let mut entry = self.declared(&format!("{prefix}{name}"), declaration)?;
            entry["status"] = json!("available");
            entry["original_suffix"] = json!(suffix);
            entry["source"] = json!(source);
            entry["canonical_catalog_sha256"] = self.files[&catalog_name]["sha256"].clone();
            entries.insert(meaning.into(), entry);
        }
        Ok(Some(entries))
    }
    fn helper_stderr(&mut self, root: &Path, name: &str) -> Result<Value, String> {
        let suffix = "-helper-stderr.txt";
        let observation_name = format!("{name}-driver.json");
        let Some(raw) = self.read(&observation_name, self.limits.json_bytes)? else {
            return Ok(unavailable(
                suffix,
                "no supervised native process observation retained",
            ));
        };
        let observation: CommandObservation = crate::qualify::spec::strict(&raw)?;
        if observation.schema != "gomad.native-process-observation/v1" {
            return Err("variant transport observation schema mismatch".into());
        }
        if observation.pid.is_none() {
            return Ok(unavailable(
                suffix,
                "native child was not spawned; no child stderr was observed",
            ));
        }
        if (observation.exit_code.is_none() && observation.signal.is_none())
            || !observation.cleanup_errors.is_empty()
        {
            return Ok(unavailable(
                suffix,
                "native child transport was not proven closed after cleanup",
            ));
        }
        let capture = &observation.stderr;
        if let Some(error) = &capture.error {
            return Ok(unavailable(
                suffix,
                &format!("native stderr capture unavailable: {error}"),
            ));
        }
        let stderr_name = format!("{name}-driver.stderr");
        if capture.path != root.join(&stderr_name) {
            return Err("variant transport stderr path changed".into());
        }
        let size = capture
            .captured_bytes
            .ok_or("variant transport capture size absent")?;
        if size > 64 << 20
            || capture.bytes.is_none_or(|bytes| bytes < size)
            || capture.truncated != capture.bytes.is_some_and(|bytes| bytes > size)
        {
            return Err("variant transport capture size or truncation mismatch".into());
        }
        let mut entry =
            self.declared(&stderr_name, &json!({"bytes":size,"sha256":capture.sha256}))?;
        if observation.stderr_sha256 != capture.sha256 {
            return Err("variant transport capture digest aliases disagree".into());
        }
        entry["status"] = json!("available");
        entry["original_suffix"] = json!(suffix);
        entry["source"] = json!("raw-supervised-child-stderr");
        entry["observed_bytes"] = json!(capture.bytes);
        entry["truncated"] = json!(capture.truncated);
        entry["process_observation_sha256"] = self.files[&observation_name]["sha256"].clone();
        Ok(entry)
    }
}
fn same_declaration(canonical: &Value, native: &Value) -> Result<(), String> {
    if canonical["bytes"] != native["bytes"]
        || canonical["sha256"] != native["sha256"]
        || native["bytes"].as_u64().is_none()
        || native["sha256"].as_str().is_none()
    {
        return Err("variant canonical stream differs from native stream declaration".into());
    }
    Ok(())
}
fn unavailable(suffix: &str, cause: &str) -> Value {
    json!({"status":"unavailable","original_suffix":suffix,"cause":cause})
}

fn unavailable_canonical(cause: &str) -> BTreeMap<String, Value> {
    MEANINGS
        .into_iter()
        .map(|(meaning, suffix, _)| (meaning.into(), unavailable(suffix, cause)))
        .collect()
}

pub(super) fn retain_execution(root: &Path, failure: Option<&str>) -> Result<(), String> {
    retain(
        root,
        "",
        "request.json",
        None,
        "variant-artifacts.json",
        failure,
    )
}

pub(super) fn retain_supervised(
    root: &Path,
    name: &str,
    failure: Option<&str>,
) -> Result<(), String> {
    if open::components(name)?.len() != 1 {
        return Err("variant sample name must be one component".into());
    }
    retain(
        root,
        &format!("{name}/"),
        &format!("{name}.input.json"),
        Some(name),
        &format!("{name}.variant-artifacts.json"),
        failure,
    )
}

fn retain(
    root: &Path,
    prefix: &str,
    request: &str,
    transport: Option<&str>,
    destination: &str,
    failure: Option<&str>,
) -> Result<(), String> {
    let limits = ReadLimits::default();
    let mut inventory = Inventory {
        root: open::pin(root)?,
        limits,
        remaining: limits.total_bytes,
        files: BTreeMap::new(),
    };
    let mut errors = Vec::new();
    let request_sha256 = match inventory.read(request, limits.json_bytes) {
        Ok(Some(bytes)) => Some(canonical::digest(&bytes)),
        Ok(None) => None,
        Err(error) => {
            errors.push(error);
            None
        }
    };
    let report = match inventory.json(&format!("{prefix}report.json")) {
        Ok(report) => report,
        Err(error) => {
            errors.push(error);
            None
        }
    };
    let canonical =
        match inventory.closed_streams(prefix, request, request_sha256.as_deref(), report.as_ref())
        {
            Ok(Some(entries)) => entries,
            Ok(None) => unavailable_canonical(
                "no finalized linked Execution canonical custody was observed",
            ),
            Err(error) => {
                let entries = unavailable_canonical(&format!(
                    "canonical custody authentication failed: {error}"
                ));
                errors.push(error);
                entries
            }
        };
    let helper = if let Some(name) = transport {
        match inventory.helper_stderr(root, name) {
            Ok(entry) => entry,
            Err(error) => {
                let entry = unavailable(
                    "-helper-stderr.txt",
                    &format!("transport custody authentication failed: {error}"),
                );
                errors.push(error);
                entry
            }
        }
    } else {
        unavailable(
            "-helper-stderr.txt",
            "in-process linked producer has no supervised child stderr transport",
        )
    };
    let mut entries = canonical;
    entries.insert("helper-stderr".into(), helper);
    let receipt = json!({"schema":"gomad.native-variant-artifacts/v1","availability_only":true,
        "architecture_mapping":"six original diagnostic variant stream meanings; absence is unavailable, never an invented empty stream",
        "transport":if transport.is_some(){"supervised-native-child"}else{"in-process-linked-run"},
        "failure":failure,"catalog_errors":errors,"files":inventory.files,"artifacts":entries});
    let mut file = File::from(
        rustix::fs::openat(
            &inventory.root,
            destination,
            rustix::fs::OFlags::WRONLY
                | rustix::fs::OFlags::CREATE
                | rustix::fs::OFlags::EXCL
                | rustix::fs::OFlags::NOFOLLOW
                | rustix::fs::OFlags::CLOEXEC,
            rustix::fs::Mode::from_raw_mode(0o600),
        )
        .map_err(|error| error.to_string())?,
    );
    file.write_all(&canonical::encode(&receipt)?)
        .and_then(|()| file.sync_all())
        .map_err(|error| error.to_string())?;
    inventory
        .root
        .sync_all()
        .map_err(|error| error.to_string())?;
    if errors.is_empty() {
        Ok(())
    } else {
        Err(errors.join("; "))
    }
}

#[cfg(test)]
#[path = "artifacts_tests.rs"]
mod tests;
