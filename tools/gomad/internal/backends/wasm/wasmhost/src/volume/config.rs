//! The WASI storage model: Go's `wasi.StorageModel`, its admission rules and
//! its Go-compatible JSON identity.

use super::state::{VolumeConfig, VolumeLimits};
use serde::Deserialize;
use sha2::{Digest, Sha256};
use std::fmt::Write as _;

pub(crate) const MAXIMUM_PATH_BYTES: usize = 4096;
const BUFFER_LIMIT: u64 = 4 << 20;

#[derive(Clone, Debug, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub(crate) struct StorageFault {
    pub(crate) name: String,
    pub(crate) resource: String,
    pub(crate) operation: String,
    pub(crate) occurrence: u64,
    pub(crate) action: String,
    #[serde(default)]
    pub(crate) bytes: u64,
}

#[derive(Clone, Debug, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub(crate) struct StorageModel {
    pub(crate) volumes: Option<Vec<VolumeConfig>>,
    pub(crate) limits: VolumeLimits,
    #[serde(default)]
    pub(crate) faults: Option<Vec<StorageFault>>,
    #[serde(default)]
    pub(crate) sqlite_lock_databases: Option<Vec<String>>,
}

impl StorageModel {
    pub(crate) fn volumes(&self) -> &[VolumeConfig] {
        self.volumes.as_deref().unwrap_or(&[])
    }
    pub(crate) fn faults(&self) -> &[StorageFault] {
        self.faults.as_deref().unwrap_or(&[])
    }
    pub(crate) fn lock_databases(&self) -> &[String] {
        self.sqlite_lock_databases.as_deref().unwrap_or(&[])
    }

    /// Go `json.Marshal(StorageModel)` bytes, field order and omitempty
    /// included. The recovery image binds their SHA-256.
    pub(crate) fn go_json(&self) -> String {
        let mut out = String::from("{\"volumes\":");
        match &self.volumes {
            None => out.push_str("null"),
            Some(volumes) => {
                out.push('[');
                for (index, volume) in volumes.iter().enumerate() {
                    if index > 0 {
                        out.push(',');
                    }
                    out.push_str("{\"ID\":");
                    go_string(&mut out, &volume.id);
                    out.push_str(",\"Path\":");
                    go_string(&mut out, &volume.mount);
                    write!(out, ",\"CapacityBytes\":{}}}", volume.capacity_bytes).unwrap();
                }
                out.push(']');
            }
        }
        write!(
            out,
            ",\"limits\":{{\"PendingOperations\":{},\"Transitions\":{}}}",
            self.limits.pending_operations, self.limits.transitions
        )
        .unwrap();
        if !self.faults().is_empty() {
            out.push_str(",\"faults\":[");
            for (index, fault) in self.faults().iter().enumerate() {
                if index > 0 {
                    out.push(',');
                }
                out.push_str("{\"name\":");
                go_string(&mut out, &fault.name);
                out.push_str(",\"resource\":");
                go_string(&mut out, &fault.resource);
                out.push_str(",\"operation\":");
                go_string(&mut out, &fault.operation);
                write!(out, ",\"occurrence\":{},\"action\":", fault.occurrence).unwrap();
                go_string(&mut out, &fault.action);
                if fault.bytes != 0 {
                    write!(out, ",\"bytes\":{}", fault.bytes).unwrap();
                }
                out.push('}');
            }
            out.push(']');
        }
        if !self.lock_databases().is_empty() {
            out.push_str(",\"sqlite_lock_databases\":[");
            for (index, database) in self.lock_databases().iter().enumerate() {
                if index > 0 {
                    out.push(',');
                }
                go_string(&mut out, database);
            }
            out.push(']');
        }
        out.push('}');
        out
    }

    /// `sha256:` + hex SHA-256 of the Go JSON encoding.
    pub(crate) fn identity(&self) -> String {
        format!("sha256:{:x}", Sha256::digest(self.go_json().as_bytes()))
    }

    pub(crate) fn lock_path(&self, resource: &str) -> bool {
        self.lock_databases().iter().any(|database| {
            resource.len() > database.len()
                && resource.starts_with(database.as_str())
                && matches!(&resource[database.len()..], ".lock" | "-shm")
                || resource.starts_with(&format!("{database}-shm/"))
        })
    }

    /// Go's `validateStorageModel`, in its rule order. Every refusal is an
    /// `invalid` boundary with Go's dimension text.
    #[expect(
        clippy::too_many_lines,
        reason = "The ordered model transition preserves Go admission, mutation and refusal order"
    )]
    pub(crate) fn validate(
        &self,
        writable_directories: &[Vec<u8>],
        filesystem_bytes: u64,
    ) -> Result<(), &'static str> {
        let volumes = self.volumes();
        if volumes.is_empty() || volumes.len() > 64 || self.faults().len() > 4096 {
            return Err("storage model bounds");
        }
        let databases = self.lock_databases();
        if databases.len() > 64 {
            return Err("SQLite lock adapter bounds");
        }
        for (index, database) in databases.iter().enumerate() {
            let mounted = volumes
                .iter()
                .any(|mount| database.starts_with(&format!("{}/", mount.mount)));
            if !absolute_clean(database)
                || database.len() + 5 > MAXIMUM_PATH_BYTES
                || !mounted
                || index > 0 && database.as_str() <= databases[index - 1].as_str()
            {
                return Err("SQLite lock adapter database paths");
            }
            for other in databases {
                if database != other
                    && (self.lock_path(database) || other.starts_with(&format!("{database}/")))
                {
                    return Err("SQLite lock adapter path collision");
                }
            }
        }
        let mut capacity_bytes = 0u64;
        for mount in volumes {
            if !writable_directories
                .iter()
                .any(|directory| directory.as_slice() == mount.mount.as_bytes())
                || mount.capacity_bytes > filesystem_bytes
            {
                return Err("storage volume writable preopen or capacity");
            }
            if mount.capacity_bytes > filesystem_bytes - capacity_bytes {
                return Err("storage total volume capacity");
            }
            capacity_bytes += mount.capacity_bytes;
        }
        let mut seen = std::collections::BTreeSet::new();
        for fault in self.faults() {
            let mounted = volumes.iter().any(|mount| {
                fault.resource == mount.mount
                    || fault.resource.starts_with(&format!("{}/", mount.mount))
            });
            if fault.name.is_empty()
                || fault.name.len() > 128
                || fault.name.contains('\0')
                || seen.contains(fault.name.as_str())
                || !absolute_clean(&fault.resource)
                || fault.resource.len() > MAXIMUM_PATH_BYTES
                || !mounted
                || fault.occurrence == 0
            {
                return Err("storage fault identity");
            }
            if self.lock_path(&fault.resource) {
                return Err("storage fault targets process-owned SQLite lock");
            }
            seen.insert(fault.name.as_str());
            if !matches!(
                fault.operation.as_str(),
                "read" | "write" | "sync" | "truncate" | "open" | "mkdir" | "remove" | "rename"
            ) {
                return Err("storage fault operation");
            }
            match fault.action.as_str() {
                "io-error" | "crash-before" => {
                    if fault.bytes != 0 {
                        return Err("storage error payload");
                    }
                }
                "sync-error" => {
                    if fault.operation != "sync" || fault.bytes != 0 {
                        return Err("storage sync fault");
                    }
                }
                "no-space" => {
                    if !matches!(
                        fault.operation.as_str(),
                        "write" | "truncate" | "open" | "mkdir"
                    ) || fault.bytes != 0
                    {
                        return Err("storage space fault");
                    }
                }
                "short-write" | "torn-write" => {
                    if fault.operation != "write" || fault.bytes == 0 || fault.bytes > BUFFER_LIMIT
                    {
                        return Err("storage bounded write fault");
                    }
                }
                // `crash-after` and every other action stay refused, as in Go.
                _ => return Err("storage fault action"),
            }
            for prior in self.faults() {
                if prior.name != fault.name
                    && prior.resource == fault.resource
                    && prior.operation == fault.operation
                    && prior.occurrence == fault.occurrence
                {
                    return Err("ambiguous storage fault match");
                }
            }
        }
        Ok(())
    }
}

/// Go's `absoluteClean`: valid UTF-8, absolute, `path.Clean`-stable, no NUL.
pub(crate) fn absolute_clean(value: &str) -> bool {
    value.starts_with('/')
        && !value.contains('\0')
        && (value == "/"
            || !value.ends_with('/')
                && value[1..]
                    .split('/')
                    .all(|name| !name.is_empty() && name != "." && name != ".."))
}

/// Appends a JSON string exactly as Go's `encoding/json` writes it (HTML
/// escaping on, `\b`/`\f` short escapes, lower-case `\u00xx`, U+2028/9).
pub(crate) fn go_string(out: &mut String, value: &str) {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    out.push('"');
    for character in value.chars() {
        match character {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            '\u{8}' => out.push_str("\\b"),
            '\u{c}' => out.push_str("\\f"),
            '<' | '>' | '&' | '\u{0}'..='\u{1f}' => {
                let code = character as u32;
                out.push_str("\\u00");
                out.push(HEX[(code >> 4) as usize] as char);
                out.push(HEX[(code & 15) as usize] as char);
            }
            '\u{2028}' => out.push_str("\\u2028"),
            '\u{2029}' => out.push_str("\\u2029"),
            _ => out.push(character),
        }
    }
    out.push('"');
}

#[cfg(test)]
mod tests {
    use super::*;

    fn model(faults: &str) -> StorageModel {
        serde_json::from_str(&format!(
            r#"{{"volumes":[{{"ID":"data","Path":"/workspace","CapacityBytes":1048576}}],"limits":{{"PendingOperations":1000,"Transitions":1000}}{faults}}}"#
        ))
        .unwrap()
    }
    fn fault(operation: &str, action: &str, bytes: u64) -> String {
        format!(
            r#","faults":[{{"name":"f","resource":"/workspace/db","operation":"{operation}","occurrence":1,"action":"{action}","bytes":{bytes}}}]"#
        )
    }
    fn writable() -> Vec<Vec<u8>> {
        vec![b"/workspace".to_vec(), b"/tmp".to_vec()]
    }

    #[test]
    fn go_json_identity_matches_go_marshal() {
        let model = model("");
        assert_eq!(
            model.go_json(),
            r#"{"volumes":[{"ID":"data","Path":"/workspace","CapacityBytes":1048576}],"limits":{"PendingOperations":1000,"Transitions":1000}}"#
        );
        let mut out = String::new();
        go_string(&mut out, "a<b>&\u{8}\u{c}\u{1}\u{2028}\"\\");
        assert_eq!(out, r#""a\u003cb\u003e\u0026\b\f\u0001\u2028\"\\""#);
    }

    #[test]
    #[expect(
        clippy::too_many_lines,
        reason = "The scenario retains the complete ordered Go conformance sequence"
    )]
    fn validation_mirrors_every_go_rule_including_crash_after() {
        assert_eq!(model("").validate(&writable(), 8 << 20), Ok(()));
        for (operation, action, bytes, expected) in [
            ("write", "crash-after", 0, "storage fault action"),
            ("write", "unknown", 0, "storage fault action"),
            ("chmod", "io-error", 0, "storage fault operation"),
            ("write", "io-error", 1, "storage error payload"),
            ("write", "crash-before", 1, "storage error payload"),
            ("write", "sync-error", 0, "storage sync fault"),
            ("sync", "sync-error", 1, "storage sync fault"),
            ("read", "no-space", 0, "storage space fault"),
            ("write", "no-space", 1, "storage space fault"),
            ("sync", "short-write", 2, "storage bounded write fault"),
            ("write", "torn-write", 0, "storage bounded write fault"),
            (
                "write",
                "torn-write",
                (4 << 20) + 1,
                "storage bounded write fault",
            ),
        ] {
            assert_eq!(
                model(&fault(operation, action, bytes)).validate(&writable(), 8 << 20),
                Err(expected),
                "{operation} {action}"
            );
        }
        for (operation, action, bytes) in [
            ("write", "io-error", 0),
            ("sync", "sync-error", 0),
            ("open", "no-space", 0),
            ("write", "short-write", 2),
            ("rename", "crash-before", 0),
        ] {
            assert_eq!(
                model(&fault(operation, action, bytes)).validate(&writable(), 8 << 20),
                Ok(())
            );
        }
        assert_eq!(
            model("").validate(&[b"/tmp".to_vec()], 8 << 20),
            Err("storage volume writable preopen or capacity")
        );
        assert_eq!(
            model("").validate(&writable(), 1024),
            Err("storage volume writable preopen or capacity")
        );
        let ambiguous = r#","faults":[{"name":"a","resource":"/workspace/db","operation":"write","occurrence":1,"action":"io-error"},{"name":"b","resource":"/workspace/db","operation":"write","occurrence":1,"action":"no-space"}]"#;
        assert_eq!(
            model(ambiguous).validate(&writable(), 8 << 20),
            Err("ambiguous storage fault match")
        );
        let duplicate = r#","faults":[{"name":"a","resource":"/workspace/db","operation":"write","occurrence":1,"action":"io-error"},{"name":"a","resource":"/workspace/db","operation":"write","occurrence":2,"action":"io-error"}]"#;
        assert_eq!(
            model(duplicate).validate(&writable(), 8 << 20),
            Err("storage fault identity")
        );
        for resource in [
            "/other/db",
            "/workspace/../db",
            "relative",
            "/workspace/db/",
        ] {
            let identity = format!(
                r#","faults":[{{"name":"a","resource":"{resource}","operation":"write","occurrence":1,"action":"io-error"}}]"#
            );
            assert_eq!(
                model(&identity).validate(&writable(), 8 << 20),
                Err("storage fault identity"),
                "{resource}"
            );
        }
        let zero = r#","faults":[{"name":"a","resource":"/workspace/db","operation":"write","occurrence":0,"action":"io-error"}]"#;
        assert_eq!(
            model(zero).validate(&writable(), 8 << 20),
            Err("storage fault identity")
        );
        let locks = r#","faults":[{"name":"a","resource":"/workspace/db.lock","operation":"write","occurrence":1,"action":"io-error"}],"sqlite_lock_databases":["/workspace/db"]"#;
        assert_eq!(
            model(locks).validate(&writable(), 8 << 20),
            Err("storage fault targets process-owned SQLite lock")
        );
        for databases in [
            r#"["/other/db"]"#,
            r#"["/workspace/b","/workspace/a"]"#,
            r#"["/workspace/db","/workspace/db"]"#,
            r#"["/workspace/../db"]"#,
        ] {
            let lock = format!(r#","sqlite_lock_databases":{databases}"#);
            assert_eq!(
                model(&lock).validate(&writable(), 8 << 20),
                Err("SQLite lock adapter database paths"),
                "{databases}"
            );
        }
        for databases in [
            r#"["/workspace/db","/workspace/db.lock"]"#,
            r#"["/workspace/db","/workspace/db/inner"]"#,
        ] {
            let lock = format!(r#","sqlite_lock_databases":{databases}"#);
            assert_eq!(
                model(&lock).validate(&writable(), 8 << 20),
                Err("SQLite lock adapter path collision"),
                "{databases}"
            );
        }
        let bare = |volumes: &str, faults: &str| -> StorageModel {
            serde_json::from_str(&format!(
                r#"{{"volumes":{volumes},"limits":{{"PendingOperations":1,"Transitions":1}}{faults}}}"#
            ))
            .unwrap()
        };
        assert_eq!(
            bare("[]", "").validate(&writable(), 8 << 20),
            Err("storage model bounds")
        );
        let volumes: Vec<_> = (0..65)
            .map(|i| format!(r#"{{"ID":"v{i}","Path":"/workspace","CapacityBytes":1}}"#))
            .collect();
        assert_eq!(
            bare(&format!("[{}]", volumes.join(",")), "").validate(&writable(), 8 << 20),
            Err("storage model bounds")
        );
        let faults: Vec<_> = (0..4097)
            .map(|i| format!(r#"{{"name":"f{i}","resource":"/workspace/db","operation":"write","occurrence":{},"action":"io-error"}}"#, i + 1))
            .collect();
        assert_eq!(
            model(&format!(r#","faults":[{}]"#, faults.join(","))).validate(&writable(), 8 << 20),
            Err("storage model bounds")
        );
        let two = r#"[{"ID":"a","Path":"/workspace","CapacityBytes":5242880},{"ID":"b","Path":"/tmp","CapacityBytes":5242880}]"#;
        assert_eq!(
            bare(two, "").validate(&writable(), 8 << 20),
            Err("storage total volume capacity")
        );
        let long_name = "n".repeat(129);
        let long_resource = format!("/workspace/{}", "r".repeat(4096));
        for (name, resource) in [
            ("", "/workspace/db"),
            (long_name.as_str(), "/workspace/db"),
            ("nul\\u0000name", "/workspace/db"),
            ("ok", "/workspace/nul\\u0000path"),
            ("ok", long_resource.as_str()),
        ] {
            let fault = format!(
                r#","faults":[{{"name":"{name}","resource":"{resource}","operation":"write","occurrence":1,"action":"io-error"}}]"#
            );
            assert_eq!(
                model(&fault).validate(&writable(), 8 << 20),
                Err("storage fault identity"),
                "{name}"
            );
        }
        let long_database = format!(r#"["/workspace/{}"]"#, "d".repeat(4096 - 15));
        assert_eq!(
            model(&format!(r#","sqlite_lock_databases":{long_database}"#))
                .validate(&writable(), 8 << 20),
            Err("SQLite lock adapter database paths")
        );
        for databases in [
            r#"["/workspace/db","/workspace/db-shm"]"#,
            r#"["/workspace/db","/workspace/db-shm/x"]"#,
        ] {
            assert_eq!(
                model(&format!(r#","sqlite_lock_databases":{databases}"#))
                    .validate(&writable(), 8 << 20),
                Err("SQLite lock adapter path collision"),
                "{databases}"
            );
        }
        let many: Vec<_> = (0..65).map(|i| format!("\"/workspace/d{i:03}\"")).collect();
        assert_eq!(
            model(&format!(r#","sqlite_lock_databases":[{}]"#, many.join(",")))
                .validate(&writable(), 8 << 20),
            Err("SQLite lock adapter bounds")
        );
    }
}
