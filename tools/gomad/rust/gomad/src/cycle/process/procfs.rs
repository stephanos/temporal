use std::{fs, path::Path};
#[derive(Debug, Eq, PartialEq)]
pub(super) struct Process {
    pub state: u8,
    pub group: u32,
    pub session: u32,
}
pub(super) fn parse(stat: &str) -> Result<Process, String> {
    let (_, tail) = stat
        .rsplit_once(") ")
        .ok_or("invalid process group observation")?;
    let fields: Vec<_> = tail.split_whitespace().take(4).collect();
    if fields.len() != 4
        || fields[0].len() != 1
        || !b"RSDZTtXxKWPI".contains(&fields[0].as_bytes()[0])
    {
        return Err("invalid process group observation".into());
    }
    let decimal = |value: &str| -> Result<u32, String> {
        if value.is_empty() || !value.bytes().all(|b| b.is_ascii_digit()) {
            return Err("invalid process group observation".into());
        }
        value
            .parse()
            .map_err(|_| "invalid process group observation".into())
    };
    decimal(fields[1])?;
    Ok(Process {
        state: fields[0].as_bytes()[0],
        group: decimal(fields[2])?,
        session: decimal(fields[3])?,
    })
}
pub(super) fn quiescent(root: &Path, group: u32, session: u32) -> Result<(), String> {
    for entry in fs::read_dir(root)
        .map_err(|error| format!("cannot observe process group quiescence: {error}"))?
    {
        let entry = entry.map_err(|error| error.to_string())?;
        if !entry
            .file_name()
            .as_encoded_bytes()
            .iter()
            .all(u8::is_ascii_digit)
        {
            continue;
        }
        let stat = match fs::read_to_string(entry.path().join("stat")) {
            Ok(stat) => stat,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => continue,
            Err(error) => return Err(format!("cannot observe process group quiescence: {error}")),
        };
        let observed = parse(&stat)?;
        if !b"ZX".contains(&observed.state)
            && (observed.group == group || observed.session == session)
        {
            return Err("owned process group/session remains live".into());
        }
    }
    Ok(())
}
