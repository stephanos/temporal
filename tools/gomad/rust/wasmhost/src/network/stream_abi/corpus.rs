use super::*;
use serde::Deserialize;
use std::path::Path;

#[derive(Clone, Debug, Deserialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
struct Fields {
    op: u32,
    id: u64,
    handle: u64,
    deadline: u64,
    port: u32,
    count: u32,
    address: u32,
}
impl Fields {
    fn call(&self) -> Call {
        Call {
            op: self.op,
            id: self.id,
            handle: self.handle,
            deadline: self.deadline,
            port: self.port,
            count: self.count,
            address: self.address,
        }
    }
}
#[derive(Clone, Copy, Debug, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
enum Dimension {
    Layout,
    WritePayload,
    Payload,
}
impl Dimension {
    fn invalid(self) -> Invalid {
        Invalid(match self {
            Self::Layout => "network v2 stream layout",
            Self::WritePayload => "network v2 write payload",
            Self::Payload => "network v2 payload",
        })
    }
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Input {
    prefix_hex: String,
    zero_pad_to: usize,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Layout {
    name: String,
    input: Input,
    call: Option<Fields>,
    invalid: Option<Dimension>,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Payload {
    name: String,
    call: Fields,
    length: usize,
    invalid: Option<Dimension>,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Errno {
    message: String,
    value: u32,
}
#[derive(Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct Corpus {
    schema: String,
    layout: Vec<Layout>,
    payload: Vec<Payload>,
    errno: Vec<Errno>,
}

fn fields(op: u32, id: u64, count: u32, address: u32) -> Fields {
    Fields {
        op,
        id,
        handle: 0,
        deadline: 0,
        port: 0,
        count,
        address,
    }
}

#[expect(
    clippy::too_many_lines,
    reason = "The corpus validator preserves its complete ordered literal and observation checks."
)]
fn literals(c: &Corpus) -> Result<(), &'static str> {
    if c.schema != "gomad.rust-network-stream-wire/v1"
        || c.layout.len() != 15
        || c.payload.len() != 6
        || c.errno.len() != 10
    {
        return Err("wire schema/roster differs");
    }
    let layout = [
        (
            "listen-header",
            "0000000200000001",
            64,
            Some(fields(1, 0, 0, 0)),
        ),
        (
            "reserved-40",
            "0000000200000001000000000000000000000000000000000000000000000000000000000000000001",
            64,
            None,
        ),
        (
            "reserved-44",
            "000000020000000100000000000000000000000000000000000000000000000000000000000000000000000001",
            64,
            None,
        ),
        (
            "reserved-47",
            "000000020000000100000000000000000000000000000000000000000000000000000000000000000000000000000001",
            64,
            None,
        ),
        (
            "reserved-52",
            "0000000200000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001",
            64,
            None,
        ),
        (
            "reserved-55",
            "0000000200000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001",
            64,
            None,
        ),
        (
            "reserved-60",
            "00000002000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001",
            64,
            None,
        ),
        (
            "reserved-63",
            "00000002000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001",
            64,
            None,
        ),
        ("version-three", "0000000300000001", 64, None),
        ("short-63", "0000000300000001", 63, None),
        ("overlong-65601", "", 65601, None),
        (
            "valid-maximum-65600",
            "0000000200000001",
            65600,
            Some(fields(1, 0, 0, 0)),
        ),
        ("valid-overlong-65601", "0000000200000001", 65601, None),
        (
            "dial-address",
            "00000002000000020000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000002",
            64,
            Some(fields(2, 0, 0, 0x0a00_0002)),
        ),
        (
            "nonzero-fields",
            "00000002000000020102030405060708111213141516171821222324252627283132333441424344000000000000000051525354000000000a000002",
            64,
            Some(Fields {
                op: 2,
                id: 0x0102_0304_0506_0708,
                handle: 0x1112_1314_1516_1718,
                deadline: 0x2122_2324_2526_2728,
                port: 0x3132_3334,
                count: 0x4142_4344,
                address: 0x0a00_0002,
            }),
        ),
    ];
    for (row, (name, prefix, length, call)) in c.layout.iter().zip(layout.iter()) {
        if row.name != *name
            || row.input.prefix_hex != *prefix
            || row.input.zero_pad_to != *length
            || row.call != *call
            || row.invalid != call.is_none().then_some(Dimension::Layout)
        {
            return Err("wire layout literal differs");
        }
    }
    let payload = [
        ("write-three", 5, 3, 67, None),
        ("write-short", 5, 3, 66, Some(Dimension::WritePayload)),
        ("read-eight", 4, 8, 72, None),
        ("read-short", 4, 8, 64, Some(Dimension::Payload)),
        ("poll-buffer", 12, 0, 164, None),
        ("listen-extra", 1, 0, 65, Some(Dimension::Payload)),
    ];
    for (row, (name, op, count, length, invalid)) in c.payload.iter().zip(payload) {
        if row.name != name
            || row.call != fields(op, 1, count, 0)
            || row.length != length
            || row.invalid != invalid
        {
            return Err("wire payload literal differs");
        }
    }
    let errors = [
        ("", 0),
        ("EOF", 0),
        ("context canceled", 11),
        ("connection timed out", 65558),
        ("i/o timeout", 73),
        ("read deadline passed", 73),
        ("connection refused", 14),
        ("network resources exhausted", 42),
        ("use of closed network connection", 15),
        ("stale file handle", 8),
    ];
    for (row, (message, value)) in c.errno.iter().zip(errors) {
        if row.message != message || row.value != value {
            return Err("wire errno literal differs");
        }
    }
    Ok(())
}

fn bytes(input: &Input) -> Result<Vec<u8>, &'static str> {
    if input.zero_pad_to > 65601
        || !input.prefix_hex.len().is_multiple_of(2)
        || input.prefix_hex.len() / 2 > input.zero_pad_to
    {
        return Err("wire input bound");
    }
    let mut data = Vec::with_capacity(input.zero_pad_to);
    for pair in input.prefix_hex.as_bytes().as_chunks::<2>().0 {
        let digits = std::str::from_utf8(pair).map_err(|_| "wire hex")?;
        data.push(u8::from_str_radix(digits, 16).map_err(|_| "wire hex")?);
    }
    data.resize(input.zero_pad_to, 0);
    Ok(data)
}

fn verify_outputs(c: &Corpus) -> Result<(), String> {
    for row in &c.layout {
        let actual = decode(&bytes(&row.input).map_err(str::to_owned)?);
        let expected = match (&row.call, row.invalid) {
            (Some(call), None) => Ok(call.call()),
            (None, Some(error)) => Err(error.invalid()),
            _ => return Err("wire layout expected union".into()),
        };
        if actual != expected {
            return Err(format!(
                "{} decoded Call/Invalid differs: {actual:?}",
                row.name
            ));
        }
    }
    for row in &c.payload {
        let actual = check_payload(&row.call.call(), row.length);
        let expected = row.invalid.map_or(Ok(()), |error| Err(error.invalid()));
        if actual != expected {
            return Err(format!("{} payload Invalid differs", row.name));
        }
    }
    for row in &c.errno {
        let actual = errno(&row.message);
        if actual != row.value {
            return Err(format!("{} errno differs", row.message));
        }
    }
    Ok(())
}

fn verify(c: &Corpus) -> Result<(), String> {
    literals(c).map_err(str::to_owned)?;
    verify_outputs(c)
}

#[expect(
    clippy::case_sensitive_file_extension_comparisons,
    reason = "Corpus membership intentionally requires exact lowercase JSON filenames."
)]
fn read() -> Corpus {
    let directory =
        Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/testdata/network_stream_wire");
    let names: Vec<_> = std::fs::read_dir(&directory)
        .unwrap()
        .map(|entry| entry.unwrap().file_name().into_string().unwrap())
        .filter(|name| name.ends_with(".json"))
        .collect();
    assert_eq!(names, ["original-layout-payload-errno.json"]);
    serde_json::from_slice(&std::fs::read(directory.join(&names[0])).unwrap()).unwrap()
}

#[test]
fn stream_wire_literals_and_real_consumer_reject_changed_vectors() {
    let case = read();
    verify(&case).unwrap();
    for mutation in [
        "reserved-byte",
        "call-field",
        "payload-dimension",
        "errno",
        "missing-row",
        "allocation-bound",
    ] {
        let mut changed = case.clone();
        match mutation {
            "reserved-byte" => {
                changed.layout[1].input.prefix_hex.pop();
                changed.layout[1].input.prefix_hex.push('2');
            }
            "call-field" => changed.layout[0].call.as_mut().unwrap().id = 1,
            "payload-dimension" => changed.payload[1].invalid = Some(Dimension::Layout),
            "errno" => changed.errno[3].value = 73,
            "missing-row" => {
                changed.layout.pop();
            }
            "allocation-bound" => changed.layout[10].input.zero_pad_to = 65602,
            _ => unreachable!(),
        }
        let result = if matches!(mutation, "call-field" | "payload-dimension" | "errno") {
            verify_outputs(&changed)
        } else {
            verify(&changed)
        };
        assert!(result.is_err(), "changed wire {mutation} accepted");
    }
    assert!(
        bytes(&Input {
            prefix_hex: String::new(),
            zero_pad_to: 65602
        })
        .is_err()
    );
    assert!(
        bytes(&Input {
            prefix_hex: "0".into(),
            zero_pad_to: 1
        })
        .is_err()
    );
    assert!(
        bytes(&Input {
            prefix_hex: "zz".into(),
            zero_pad_to: 1
        })
        .is_err()
    );
}

#[test]
fn rust_network_stream_wire_corpus() {
    verify(&read()).unwrap();
}
