use super::{
    Reason,
    bytes::{Budget, Memory, STRING_LIMIT, add},
    wasm::Module,
};
use std::collections::{BTreeMap, BTreeSet};

const START: [u8; 16] = [
    0x30, 0x77, 0xaf, 0x0c, 0x92, 0x74, 0x08, 0x02, 0x41, 0xe1, 0xc1, 0x07, 0xe6, 0xd6, 0x18, 0xe6,
];
const END: [u8; 16] = [
    0xf9, 0x32, 0x43, 0x31, 0x86, 0x18, 0x20, 0x72, 0x00, 0x82, 0x42, 0x10, 0x41, 0x16, 0xd8, 0xf2,
];

#[derive(serde::Serialize)]
pub(crate) struct BuildInfo {
    pub compiler_version: String,
    pub path: String,
    pub settings: BTreeMap<String, String>,
}

pub(crate) fn read(bytes: &[u8]) -> Result<BuildInfo, String> {
    read_with_budget(bytes, &mut Budget::default())
        .map_err(|reason| format!("invalid pinned WASM build information: {reason:?}"))
}

fn read_with_budget(bytes: &[u8], budget: &mut Budget) -> Result<BuildInfo, Reason> {
    let module = Module::read(bytes, budget)?;
    if !module.pinned_go_producer {
        return Err(Reason::NoSymbols);
    }
    let starts = module.memory.matches(&START, false, budget)?;
    if starts.len() != 1 {
        return Err(Reason::SymbolMismatch);
    }
    parse(&framed(&module.memory, starts[0], budget)?).map_err(|_| Reason::SymbolMismatch)
}

fn framed(memory: &Memory<'_>, start: u64, budget: &mut Budget) -> Result<String, Reason> {
    let start = add(start, START.len() as u64)?;
    let mut bytes = Vec::new();
    for offset in 0..=STRING_LIMIT {
        let address = add(start, offset)?;
        if address >= memory.end {
            return Err(Reason::SymbolMismatch);
        }
        if memory.byte(address, budget)? == END[0] && memory.uint::<16>(address, budget)? == END {
            return String::from_utf8(bytes).map_err(|_| Reason::SymbolMismatch);
        }
        if offset == STRING_LIMIT {
            return Err(Reason::SymbolTableTooLarge);
        }
        budget.bytes(4)?;
        bytes.push(memory.byte(address, budget)?);
    }
    Err(Reason::SymbolTableTooLarge)
}

fn parse(text: &str) -> Result<BuildInfo, String> {
    if !text.ends_with('\n') {
        return Err("truncated module information".into());
    }
    let mut path = None;
    let mut main = false;
    let mut dependencies = BTreeSet::new();
    let mut replaceable = false;
    let mut settings = BTreeMap::new();
    let mut builds = false;
    for line in text.split_terminator('\n') {
        let (kind, value) = line
            .split_once('\t')
            .ok_or("invalid module information row")?;
        match kind {
            "path"
                if path.is_none() && !value.is_empty() && !value.contains(['\t', '\r', '\0']) =>
            {
                path = Some(value.to_owned());
            }
            "mod" if path.is_some() && !main && dependencies.is_empty() && !builds => {
                module_row(value)?;
                main = true;
                replaceable = true;
            }
            "dep" if path.is_some() && !builds => {
                let dependency = module_row(value)?;
                if !dependencies.insert(dependency.to_owned()) {
                    return Err("duplicate module dependency".into());
                }
                replaceable = true;
            }
            "=>" if replaceable && !builds => {
                module_row(value)?;
                replaceable = false;
            }
            "build" if path.is_some() => {
                builds = true;
                replaceable = false;
                let (key, value) = setting(value)?;
                if settings.insert(key, value).is_some() {
                    return Err("duplicate module build setting".into());
                }
            }
            _ => return Err("invalid module information row or ordering".into()),
        }
    }
    Ok(BuildInfo {
        compiler_version: "go1.27.1".into(),
        path: path.ok_or("module package path missing")?,
        settings,
    })
}

fn module_row(value: &str) -> Result<&str, String> {
    let fields = value.split('\t').collect::<Vec<_>>();
    if !(2..=3).contains(&fields.len())
        || fields[..2]
            .iter()
            .any(|field| field.is_empty() || field.contains(['\r', '\0']))
    {
        return Err("invalid module dependency row".into());
    }
    Ok(fields[0])
}

fn setting(text: &str) -> Result<(String, String), String> {
    let (key, value) = if text.starts_with(['"', '`']) {
        let (key, consumed) = quoted(text)?;
        (
            key,
            text.get(consumed..)
                .and_then(|tail| tail.strip_prefix('='))
                .ok_or("build setting equals missing")?,
        )
    } else {
        let (key, value) = text.split_once('=').ok_or("build setting equals missing")?;
        if key.is_empty() || key.contains([' ', '\t', '\r', '\n', '"', '`']) {
            return Err("invalid unquoted build key".into());
        }
        (key.to_owned(), value)
    };
    let value = if value.starts_with(['"', '`']) {
        let (decoded, consumed) = quoted(value)?;
        if consumed != value.len() {
            return Err("trailing quoted build value".into());
        }
        decoded
    } else {
        if value.contains([' ', '\t', '\r', '\n', '"', '`']) {
            return Err("invalid unquoted build value".into());
        }
        value.to_owned()
    };
    if key.is_empty() || key.contains('\0') || value.contains('\0') {
        return Err("invalid build setting text".into());
    }
    Ok((key, value))
}

fn quoted(text: &str) -> Result<(String, usize), String> {
    let input = text.as_bytes();
    let quote = *input.first().ok_or("missing quote")?;
    if !matches!(quote, b'"' | b'`') {
        return Err("invalid string quote".into());
    }
    let mut bytes = Vec::new();
    let mut cursor = 1;
    while let Some(&byte) = input.get(cursor) {
        cursor += 1;
        if byte == quote {
            return Ok((
                String::from_utf8(bytes).map_err(|_| "build setting is not UTF-8")?,
                cursor,
            ));
        }
        if byte == b'\n' {
            return Err("newline in quoted build setting".into());
        }
        if quote == b'`' {
            if byte != b'\r' {
                bytes.push(byte);
            }
            continue;
        }
        if byte != b'\\' {
            bytes.push(byte);
            continue;
        }
        let escape = *input.get(cursor).ok_or("truncated quoted escape")?;
        cursor += 1;
        match escape {
            b'a' => bytes.push(7),
            b'b' => bytes.push(8),
            b'f' => bytes.push(12),
            b'n' => bytes.push(b'\n'),
            b'r' => bytes.push(b'\r'),
            b't' => bytes.push(b'\t'),
            b'v' => bytes.push(11),
            b'\\' => bytes.push(b'\\'),
            b'"' => bytes.push(b'"'),
            b'x' | b'u' | b'U' => {
                let count = match escape {
                    b'x' => 2,
                    b'u' => 4,
                    _ => 8,
                };
                let end = cursor.checked_add(count).ok_or("escape overflow")?;
                let digits = text.get(cursor..end).ok_or("truncated hex escape")?;
                if !digits.bytes().all(|b| b.is_ascii_hexdigit()) {
                    return Err("invalid hex escape".into());
                }
                let value = u32::from_str_radix(digits, 16).map_err(|_| "hex escape overflow")?;
                if escape == b'x' {
                    bytes.push(u8::try_from(value).map_err(|_| "hex byte overflow")?);
                } else {
                    let character = char::from_u32(value).ok_or("invalid unicode escape")?;
                    let mut encoded = [0; 4];
                    bytes.extend_from_slice(character.encode_utf8(&mut encoded).as_bytes());
                }
                cursor = end;
            }
            b'0'..=b'7' => {
                let end = cursor.checked_add(2).ok_or("escape overflow")?;
                let digits = text.get(cursor - 1..end).ok_or("truncated octal escape")?;
                if !digits.bytes().all(|b| (b'0'..=b'7').contains(&b)) {
                    return Err("invalid octal escape".into());
                }
                bytes.push(u8::from_str_radix(digits, 8).map_err(|_| "octal byte overflow")?);
                cursor = end;
            }
            _ => return Err("invalid Go quoted escape".into()),
        }
    }
    Err("unterminated Go build setting quote".into())
}

#[cfg(test)]
mod tests {
    use super::*;
    const PINNED: &[u8] = include_bytes!("../../../../../testdata/parity/symbolize/guest.wasm");

    #[test]
    fn real_pinned_wasm_has_framed_settings_without_native_buildinfo_header() {
        assert!(
            !PINNED
                .windows(14)
                .any(|bytes| bytes == b"\xff Go buildinf:")
        );
        let info = read(PINNED).unwrap();
        assert_eq!(info.path, "gomad.fixture/symbolize");
        assert_eq!(info.settings["GOOS"], "wasip1");
        assert_eq!(info.settings["GOARCH"], "wasm");
        assert_eq!(info.settings["GOEXPERIMENT"], "nogreenteagc");
        assert_eq!(info.settings["-tags"], "test_dep");
    }

    #[test]
    fn duplicate_truncated_malformed_and_budget_fail_closed() {
        let mut duplicated = PINNED.to_vec();
        duplicated.extend_from_slice(&[0, 9, 4, b'n', b'a', b'm', b'e', 1, 2, 0, 0]);
        assert!(read(&duplicated).is_err());
        assert!(read(&PINNED[..PINNED.len() / 2]).is_err());
        assert!(
            read_with_budget(
                PINNED,
                &mut Budget {
                    steps: super::super::bytes::STEP_LIMIT,
                    ..Budget::default()
                }
            )
            .is_err()
        );
        for text in [
            "path\tfirst\npath\tsecond\n",
            "path\ta\nbuild\tGOOS=wasip1\nbuild\tGOOS=wasip1\n",
            "path\ta\nunknown\tb\n",
            "path\ta",
            "path\ta\nbuild\tGOOS=\"unterminated\n",
        ] {
            assert!(parse(text).is_err(), "accepted {text:?}");
        }
    }

    #[test]
    fn sparse_zero_elision_preserves_complete_sentinel_and_duplicate_refusal() {
        use super::super::bytes::Segment;
        let text = b"path\tactual/package\nbuild\tGOOS=wasip1\n";
        let mut bytes = Vec::from(START);
        bytes.extend_from_slice(text);
        bytes.extend_from_slice(&END);
        let split = bytes.len() - 8;
        assert_eq!(bytes[split], 0);
        let memory = Memory {
            end: (bytes.len() + 64) as u64,
            segments: vec![
                Segment {
                    start: 0,
                    bytes: &bytes[..split],
                },
                Segment {
                    start: (split + 1) as u64,
                    bytes: &bytes[split + 1..],
                },
            ],
        };
        let framed = framed(&memory, 0, &mut Budget::default()).unwrap();
        assert_eq!(framed.as_bytes(), text);
        let repeated = Memory {
            end: 256,
            segments: vec![
                Segment {
                    start: 0,
                    bytes: &bytes,
                },
                Segment {
                    start: 128,
                    bytes: &bytes,
                },
            ],
        };
        assert_eq!(
            repeated
                .matches(&START, false, &mut Budget::default())
                .unwrap(),
            [0, 128]
        );
        let truncated = Memory {
            end: split as u64,
            segments: vec![Segment {
                start: 0,
                bytes: &bytes[..split],
            }],
        };
        assert!(super::framed(&truncated, 0, &mut Budget::default()).is_err());
    }

    #[test]
    fn pinned_go_quoted_setting_grammar_is_exact() {
        assert_eq!(
            setting("key=\"a\\x20b\\u0021\\U0000003f\\041\"").unwrap(),
            ("key".into(), "a b!?!".into())
        );
        assert_eq!(
            setting("`quoted key`=`literal value`").unwrap(),
            ("quoted key".into(), "literal value".into())
        );
        for invalid in [
            "a=unquoted value",
            "a=\"b\"trailing",
            "a=\"\\x0\"",
            "a=\"\\uD800\"",
            "a=\"\\400\"",
            "=x",
        ] {
            assert!(setting(invalid).is_err(), "accepted {invalid:?}");
        }
    }
}

#[cfg(test)]
mod wasm_boundary_tests {
    use super::*;
    fn unsigned(mut value: u32, output: &mut Vec<u8>) {
        loop {
            let byte = u8::try_from(value & 127).unwrap();
            value >>= 7;
            output.push(byte | if value == 0 { 0 } else { 128 });
            if value == 0 {
                break;
            }
        }
    }
    fn string(value: &str, output: &mut Vec<u8>) {
        unsigned(u32::try_from(value.len()).unwrap(), output);
        output.extend_from_slice(value.as_bytes());
    }
    fn section(id: u8, bytes: &[u8], output: &mut Vec<u8>) {
        output.push(id);
        unsigned(u32::try_from(bytes.len()).unwrap(), output);
        output.extend_from_slice(bytes);
    }
    fn wasm(data: &[(i32, &[u8])], producer: bool) -> Vec<u8> {
        let mut output = b"\0asm\x01\0\0\0".to_vec();
        section(5, &[1, 0, 1], &mut output);
        let mut payload = Vec::new();
        unsigned(u32::try_from(data.len()).unwrap(), &mut payload);
        for (offset, bytes) in data {
            payload.extend_from_slice(&[0, 0x41]);
            let mut value = *offset;
            loop {
                let byte = u8::try_from(value & 127).unwrap();
                value >>= 7;
                let done = value == 0 && byte & 64 == 0 || value == -1 && byte & 64 != 0;
                payload.push(byte | if done { 0 } else { 128 });
                if done {
                    break;
                }
            }
            payload.push(0x0b);
            unsigned(u32::try_from(bytes.len()).unwrap(), &mut payload);
            payload.extend_from_slice(bytes);
        }
        section(11, &payload, &mut output);
        if producer {
            let mut payload = Vec::new();
            string("producers", &mut payload);
            payload.push(2);
            for (field, name) in [("language", "Go"), ("processed-by", "Go cmd/compile")] {
                string(field, &mut payload);
                payload.push(1);
                string(name, &mut payload);
                string("go1.27.1", &mut payload);
            }
            section(0, &payload, &mut output);
        }
        output
    }
    #[test]
    fn actual_wasm_data_duplicate_overlap_truncation_and_producer_refusals() {
        let mut frame = Vec::from(START);
        frame.extend_from_slice(b"path\texact/package\nbuild\tGOOS=wasip1\n");
        frame.extend_from_slice(&END);
        assert_eq!(
            read(&wasm(&[(0, &frame)], true)).unwrap().path,
            "exact/package"
        );
        assert!(read(&wasm(&[(0, &frame), (256, &frame)], true)).is_err());
        assert!(read(&wasm(&[(0, &frame), (0, &frame)], true)).is_err());
        assert!(read(&wasm(&[(0, &frame[..frame.len() - 1])], true)).is_err());
        assert!(read(&wasm(&[(0, &frame)], false)).is_err());
        let mut conflicting = wasm(&[(0, &frame)], true);
        let version = conflicting
            .windows(8)
            .position(|bytes| bytes == b"go1.27.1")
            .unwrap();
        conflicting[version + 7] = b'0';
        assert!(read(&conflicting).is_err());
        let split = frame.len() - 8;
        assert_eq!(frame[split], 0);
        let sparse = wasm(
            &[
                (0, &frame[..split]),
                (i32::try_from(split + 1).unwrap(), &frame[split + 1..]),
            ],
            true,
        );
        assert_eq!(read(&sparse).unwrap().path, "exact/package");
    }
}
