//! Independent v2 inspection oracle decoded from canonical GMNETEV1 evidence.
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;

struct Reader<'a> {
    bytes: &'a [u8],
    at: usize,
}
impl<'a> Reader<'a> {
    fn raw(&mut self, n: usize) -> Result<&'a [u8], String> {
        let end = self
            .at
            .checked_add(n)
            .ok_or("network oracle length overflow")?;
        let out = self
            .bytes
            .get(self.at..end)
            .ok_or("short network oracle input")?;
        self.at = end;
        Ok(out)
    }
    fn number(&mut self) -> Result<u64, String> {
        let mut value = 0;
        for shift in (0..64).step_by(7) {
            let byte = self.raw(1)?[0];
            value |= u64::from(byte & 127) << shift;
            if byte & 128 == 0 {
                return Ok(value);
            }
        }
        Err("network oracle varint overflow".into())
    }
    fn blob(&mut self) -> Result<&'a [u8], String> {
        let n = usize::try_from(self.number()?).map_err(|_| "network oracle host width")?;
        self.raw(n)
    }
    fn flag(&mut self) -> Result<u64, String> {
        match self.raw(1)?[0] {
            0 => Ok(0),
            1 => Ok(1),
            _ => Err("network oracle flag".into()),
        }
    }
    fn count(&mut self) -> Result<u64, String> {
        if self.flag()? == 0 {
            Ok(0)
        } else {
            self.number()
        }
    }
    fn fields(&mut self, pattern: &str, out: &mut Vec<u8>) -> Result<(), String> {
        for field in pattern.bytes() {
            match field {
                b'u' => uint(out, self.number()?),
                b'b' => uint(out, self.flag()?),
                b's' => blob(out, self.blob()?),
                _ => unreachable!(),
            }
        }
        Ok(())
    }
    fn endpoint(&mut self, out: &mut Vec<u8>) -> Result<String, String> {
        let node = self.blob()?;
        let incarnation = self.number()?;
        let address = self.blob()?;
        let port = self.number()?;
        blob(out, node);
        uint(out, incarnation);
        blob(out, address);
        uint(out, port);
        Ok(format!(
            "{}\0{incarnation:020}\0{}\0{port:05}",
            std::str::from_utf8(node).map_err(|e| e.to_string())?,
            std::str::from_utf8(address).map_err(|e| e.to_string())?
        ))
    }
}
fn uint(out: &mut Vec<u8>, n: u64) {
    out.extend_from_slice(&n.to_le_bytes());
}
fn blob(out: &mut Vec<u8>, value: &[u8]) {
    uint(out, value.len() as u64);
    out.extend_from_slice(value);
}
fn hash(domain: &[u8], bytes: &[u8]) -> [u8; 32] {
    let mut hash = Sha256::new();
    hash.update(domain);
    hash.update(bytes);
    hash.finalize().into()
}
fn hash_text(domain: &[u8], bytes: &[u8]) -> String {
    format!(
        "sha256:{:x}",
        Sha256::new()
            .chain_update(domain)
            .chain_update(bytes)
            .finalize()
    )
}
fn world(wire: &[u8]) -> Result<[u8; 32], String> {
    if wire.len() < 208 || &wire[..8] != b"GMWSNAP1" {
        return Err("network oracle World header".into());
    }
    let mut bytes = wire[8..104].to_vec();
    bytes.extend_from_slice(&(wire.len() as u64).to_be_bytes());
    bytes.extend_from_slice(&64u64.to_be_bytes());
    for byte in &wire[120..152] {
        bytes.extend_from_slice(format!("{byte:02x}").as_bytes());
    }
    Ok(hash(b"gomad/world/inspection-chain/v2\0", &bytes))
}

fn skip_log(r: &mut Reader<'_>, entries: u64) -> Result<(), String> {
    for _ in 0..entries {
        match r.raw(1)?[0] {
            1 => {
                r.number()?;
                r.fields("usussususuuu", &mut Vec::new())?;
                r.fields("ususususuus", &mut Vec::new())?;
            }
            3 => {
                r.fields("suu", &mut Vec::new())?;
                for _ in 0..r.number()? {
                    r.fields("uuuss", &mut Vec::new())?;
                }
                for _ in 0..r.number()? {
                    r.number()?;
                }
            }
            _ => return Err("network oracle operation kind".into()),
        }
    }
    Ok(())
}

pub fn material(wire: &[u8]) -> Result<Vec<u8>, String> {
    let mut r = Reader { bytes: wire, at: 0 };
    if r.raw(8)? != b"GMNETEV1" || r.number()? != 1 {
        return Err("network oracle header".into());
    }
    let profile = r.blob()?;
    let log_bytes = r.number()?;
    let entries = r.number()?;
    let start = r.at;
    skip_log(&mut r, entries)?;
    let operations = &wire[start..r.at];
    let mut lanes: BTreeMap<String, (u64, [u8; 32])> = BTreeMap::new();
    for _ in 0..r.count()? {
        r.number()?;
        let kind = std::str::from_utf8(r.blob()?).map_err(|e| e.to_string())?;
        let mut encoded = Vec::new();
        uint(&mut encoded, 0);
        blob(&mut encoded, kind.as_bytes());
        let source = r.endpoint(&mut encoded)?;
        let destination = r.endpoint(&mut encoded)?;
        let connection = r.number()?;
        uint(&mut encoded, connection);
        r.fields("uuus", &mut encoded)?;
        let lane = match kind {
            "partition" | "heal" | "delay" | "disconnect" | "reconnect" | "directional_delay" => {
                "topology".into()
            }
            "listen" | "listener_close" => format!("listener\0{source}"),
            "dial" | "accept" | "write" | "close" | "reset" if connection != 0 => {
                format!("connection\0{connection:020}")
            }
            "dial" | "accept" | "write" | "close" | "reset" => {
                format!("operation\0{kind}\0{source}\0{destination}")
            }
            "stop" | "crash" => format!("lifecycle\0{source}"),
            _ => return Err("network oracle transition kind".into()),
        };
        let (count, head) = lanes.entry(lane).or_default();
        let mut chain = head.to_vec();
        uint(&mut chain, *count);
        blob(&mut chain, &encoded);
        *head = hash(b"gomad/network/inspection-lane/v2\0", &chain);
        *count += 1;
    }
    let mut chain = Vec::new();
    uint(&mut chain, lanes.len() as u64);
    for (lane, (count, head)) in lanes {
        blob(&mut chain, lane.as_bytes());
        uint(&mut chain, count);
        chain.extend_from_slice(&head);
    }
    let transitions = hash_text(b"gomad/network/inspection-history/v2\0", &chain);
    let mut snapshot = Vec::new();
    for pattern in ["ssuuu", "ssbu", "susub"] {
        let count = r.count()?;
        uint(&mut snapshot, count);
        for _ in 0..count {
            r.fields(pattern, &mut snapshot)?;
        }
    }
    let count = r.count()?;
    uint(&mut snapshot, count);
    for _ in 0..count {
        let client_bytes = r.number()?;
        let server_bytes = r.number()?;
        r.fields("ususususubb", &mut snapshot)?;
        uint(&mut snapshot, client_bytes);
        uint(&mut snapshot, server_bytes);
    }
    r.fields("u", &mut snapshot)?;
    r.blob()?;
    r.blob()?;
    blob(&mut snapshot, transitions.as_bytes());
    blob(&mut snapshot, b"");
    let native = hash_text(b"gomad-simulation-network-snapshot/v3\0", &snapshot);
    if r.flag()? != 1 {
        return Err("network oracle missing World".into());
    }
    let world = world(r.blob()?)?;
    if r.at != wire.len() {
        return Err("network oracle trailing input".into());
    }
    let mut out = b"gomad/network/inspection-material/v2\0".to_vec();
    blob(&mut out, profile);
    blob(&mut out, native.as_bytes());
    for n in [log_bytes, entries, operations.len() as u64] {
        uint(&mut out, n);
    }
    out.extend_from_slice(&Sha256::digest(operations));
    out.extend_from_slice(&world);
    Ok(out)
}
