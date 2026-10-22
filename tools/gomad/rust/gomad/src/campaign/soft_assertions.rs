use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BTreeSet};

const PREFIX: &str = "failed assertion: ";

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Report {
    pub messages: Vec<String>,
    #[expect(
        clippy::struct_field_names,
        reason = "report_only is the retained soft assertion report schema field"
    )]
    pub report_only: bool,
    pub inconclusive: bool,
}

pub(super) fn assessed(
    prepared: &super::prepare::Prepared,
    observed: &gomad_wasmhost::linked::Execution,
) -> Result<Option<Report>, String> {
    let provenance: serde_json::Value =
        serde_json::from_slice(&prepared.provenance).map_err(|error| error.to_string())?;
    if provenance["inputs"]["kind"] != "go-test" {
        return Ok(None);
    }
    Ok(observe(
        observed.evidence.get("stdout").map_or(&[], Vec::as_slice),
        observed.evidence.get("stderr").map_or(&[], Vec::as_slice),
        matches!(observed.outcome.termination, "exit" | "trap"),
        prepared.soft_assertion_report_only,
    ))
}

pub(crate) fn observe(
    stdout: &[u8],
    stderr: &[u8],
    complete: bool,
    report_only: bool,
) -> Option<Report> {
    let mut complete = complete;
    let mut messages = Vec::new();
    for stream in [stdout, stderr] {
        let Ok(stream) = std::str::from_utf8(stream) else {
            complete = false;
            continue;
        };
        for line in stream.split('\n') {
            let line = line
                .strip_suffix('\r')
                .unwrap_or(line)
                .trim_start_matches([' ', '\t']);
            match observe_line(line) {
                Ok(Some(message)) => messages.push(message),
                Ok(None) => (),
                Err(()) => complete = false,
            }
        }
    }
    messages.sort();
    messages.dedup();
    if messages.is_empty() && complete && !report_only {
        return None;
    }
    Some(Report {
        messages,
        report_only,
        inconclusive: !complete,
    })
}

fn observe_line(line: &str) -> Result<Option<String>, ()> {
    let record = json_record_start(line);
    let at = record.or_else(|| line.rfind("\t{").map(|at| at + 1));
    let mut line = line;
    if let Some(at) = at {
        if let Ok(fields) = Json::fields(&line[at..]) {
            let mut tagged = false;
            for key in ["failed-assertion", "FailedAssertion"] {
                if let Some(value) = fields.get(key) {
                    tagged |= match *value {
                        "true" => true,
                        "false" | "null" => false,
                        _ => return Err(()),
                    };
                }
            }
            if record.is_some() {
                let mut message = String::new();
                for key in ["msg", "message"] {
                    if let Some(value) = fields.get(key) {
                        if *value != "null" {
                            match serde_json::from_str::<String>(value) {
                                Ok(value) => message = value,
                                Err(_) => return if tagged { Err(()) } else { Ok(None) },
                            }
                        }
                        break;
                    }
                }
                let assertion = tagged || message.starts_with(PREFIX);
                return static_message(&message, assertion);
            }
            line = line[..at].strip_suffix('\t').unwrap_or(&line[..at]);
            if tagged && !line.contains(PREFIX) {
                return static_message(line.rsplit('\t').next().unwrap_or(""), true);
            }
        } else if line.contains(PREFIX)
            || line.contains("failed-assertion")
            || line.contains("FailedAssertion")
        {
            return Err(());
        }
    }
    if let Some(at) = line.find(PREFIX) {
        return static_message(&line[at..], true);
    }
    Ok(None)
}

fn json_record_start(line: &str) -> Option<usize> {
    let at = line.find('{')?;
    if at == 0 {
        return Some(at);
    }
    let prefix = line[..at].trim();
    let prefix = prefix.strip_suffix(':')?;
    let (source, number) = prefix.rsplit_once(':')?;
    if source.is_empty()
        || number.is_empty()
        || !number.bytes().all(|byte| byte.is_ascii_digit())
        || source.contains([' ', '\t'])
        || source.contains(PREFIX)
    {
        return None;
    }
    Some(at)
}

fn static_message(message: &str, assertion: bool) -> Result<Option<String>, ()> {
    if !assertion {
        return Ok(None);
    }
    let message = message.strip_prefix(PREFIX).unwrap_or(message);
    if message.is_empty() || message.len() > 4096 {
        return Err(());
    }
    Ok(Some(message.into()))
}

struct Json<'a> {
    text: &'a str,
    at: usize,
}

enum Container {
    Array {
        after: bool,
        first: bool,
    },
    Object {
        after: bool,
        first: bool,
        keys: BTreeSet<String>,
    },
}

impl<'a> Json<'a> {
    fn fields(text: &'a str) -> Result<BTreeMap<String, &'a str>, ()> {
        let mut parser = Self { text, at: 0 };
        parser.expect(b'{')?;
        let mut fields = BTreeMap::new();
        if parser.peek() == Some(b'}') {
            parser.at += 1;
        } else {
            loop {
                let key = parser.string()?;
                parser.expect(b':')?;
                parser.space();
                let start = parser.at;
                parser.value()?;
                if fields.insert(key, &text[start..parser.at]).is_some() {
                    return Err(());
                }
                match parser.peek() {
                    Some(b',') => parser.at += 1,
                    Some(b'}') => {
                        parser.at += 1;
                        break;
                    }
                    _ => return Err(()),
                }
            }
        }
        if parser.peek().is_some() {
            return Err(());
        }
        Ok(fields)
    }

    fn space(&mut self) {
        while self
            .text
            .as_bytes()
            .get(self.at)
            .is_some_and(|byte| matches!(byte, b' ' | b'\t' | b'\n' | b'\r'))
        {
            self.at += 1;
        }
    }

    fn peek(&mut self) -> Option<u8> {
        self.space();
        self.text.as_bytes().get(self.at).copied()
    }

    fn expect(&mut self, byte: u8) -> Result<(), ()> {
        if self.peek() != Some(byte) {
            return Err(());
        }
        self.at += 1;
        Ok(())
    }

    fn string(&mut self) -> Result<String, ()> {
        self.space();
        let start = self.at;
        self.expect(b'"')?;
        while let Some(byte) = self.text.as_bytes().get(self.at).copied() {
            self.at += 1;
            match byte {
                b'\\' => {
                    if self.at == self.text.len() {
                        return Err(());
                    }
                    self.at += 1;
                }
                b'"' => return serde_json::from_str(&self.text[start..self.at]).map_err(|_| ()),
                _ => (),
            }
        }
        Err(())
    }

    fn start_value(&mut self) -> Result<Option<Container>, ()> {
        match self.peek().ok_or(())? {
            b'{' => {
                self.at += 1;
                Ok(Some(Container::Object {
                    after: false,
                    first: true,
                    keys: BTreeSet::new(),
                }))
            }
            b'[' => {
                self.at += 1;
                Ok(Some(Container::Array {
                    after: false,
                    first: true,
                }))
            }
            b'"' => {
                self.string()?;
                Ok(None)
            }
            b't' => {
                self.literal("true")?;
                Ok(None)
            }
            b'f' => {
                self.literal("false")?;
                Ok(None)
            }
            b'n' => {
                self.literal("null")?;
                Ok(None)
            }
            b'-' | b'0'..=b'9' => {
                self.number()?;
                Ok(None)
            }
            _ => Err(()),
        }
    }

    fn value(&mut self) -> Result<(), ()> {
        let Some(root) = self.start_value()? else {
            return Ok(());
        };
        let mut stack = vec![root];
        while let Some(container) = stack.last_mut() {
            let mut descend = false;
            let mut finished = false;
            match container {
                Container::Array { after, first } => {
                    if *after {
                        match self.peek() {
                            Some(b',') => {
                                self.at += 1;
                                *after = false;
                            }
                            Some(b']') => {
                                self.at += 1;
                                finished = true;
                            }
                            _ => return Err(()),
                        }
                    } else if *first && self.peek() == Some(b']') {
                        self.at += 1;
                        finished = true;
                    } else {
                        *after = true;
                        *first = false;
                        descend = true;
                    }
                }
                Container::Object { after, first, keys } => {
                    if *after {
                        match self.peek() {
                            Some(b',') => {
                                self.at += 1;
                                *after = false;
                            }
                            Some(b'}') => {
                                self.at += 1;
                                finished = true;
                            }
                            _ => return Err(()),
                        }
                    } else if *first && self.peek() == Some(b'}') {
                        self.at += 1;
                        finished = true;
                    } else {
                        if !keys.insert(self.string()?) {
                            return Err(());
                        }
                        self.expect(b':')?;
                        *after = true;
                        *first = false;
                        descend = true;
                    }
                }
            }
            if finished {
                stack.pop();
            }
            if descend && let Some(child) = self.start_value()? {
                stack.push(child);
            }
            if stack.len() >= 10_000 {
                return Err(());
            }
        }
        Ok(())
    }

    fn literal(&mut self, literal: &str) -> Result<(), ()> {
        if !self.text[self.at..].starts_with(literal) {
            return Err(());
        }
        self.at += literal.len();
        Ok(())
    }

    fn number(&mut self) -> Result<(), ()> {
        let bytes = self.text.as_bytes();
        if bytes.get(self.at) == Some(&b'-') {
            self.at += 1;
        }
        match bytes.get(self.at) {
            Some(b'0') => self.at += 1,
            Some(b'1'..=b'9') => {
                self.at += 1;
                while bytes.get(self.at).is_some_and(u8::is_ascii_digit) {
                    self.at += 1;
                }
            }
            _ => return Err(()),
        }
        if bytes.get(self.at) == Some(&b'.') {
            self.at += 1;
            self.digits()?;
        }
        if matches!(self.text.as_bytes().get(self.at), Some(b'e' | b'E')) {
            self.at += 1;
            if matches!(self.text.as_bytes().get(self.at), Some(b'+' | b'-')) {
                self.at += 1;
            }
            self.digits()?;
        }
        Ok(())
    }

    fn digits(&mut self) -> Result<(), ()> {
        let start = self.at;
        while self
            .text
            .as_bytes()
            .get(self.at)
            .is_some_and(u8::is_ascii_digit)
        {
            self.at += 1;
        }
        if self.at == start { Err(()) } else { Ok(()) }
    }
}

pub(super) fn progress(
    output: &mut dyn std::io::Write,
    campaign: &std::path::Path,
    seed: u64,
    report: &serde_json::Value,
) -> Result<(), String> {
    writeln!(
        output,
        "{}",
        serde_json::json!({
            "schema":"gomad.explore-event/v3", "type":"soft_assertion",
            "campaign_path":campaign, "seed":seed, "soft_assertions":report
        })
    )
    .map_err(|error| error.to_string())?;
    output.flush().map_err(|error| error.to_string())?;
    #[cfg(feature = "determinism-fixture")]
    crate::live_backend::progress_acknowledged()?;
    Ok(())
}

pub(crate) fn reason(report: Option<&Report>, message: String) -> String {
    match report {
        Some(report) if report.inconclusive => "soft_assertion_inconclusive".into(),
        Some(report) if !report.report_only && !report.messages.is_empty() => {
            "soft_assertion".into()
        }
        _ => message,
    }
}

#[cfg(test)]
mod live_backend_tests {
    #[test]
    fn soft_assertion_pass_exit_has_oracle_reason() {
        for (report_only, inconclusive, expected) in [
            (false, false, "soft_assertion"),
            (true, false, ""),
            (true, true, "soft_assertion_inconclusive"),
        ] {
            let report = super::Report {
                messages: vec!["invalid fixture state".into()],
                report_only,
                inconclusive,
            };
            assert_eq!(super::reason(Some(&report), String::new()), expected);
        }
    }
}
