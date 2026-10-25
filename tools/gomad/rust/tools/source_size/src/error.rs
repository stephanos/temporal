use std::ffi::OsString;
use std::fmt;
use std::process::ExitStatus;

#[derive(Debug)]
pub(super) enum Error {
    Io(std::io::Error),
    Utf8(std::str::Utf8Error),
    ParseSize(std::num::ParseIntError),
    SizeConversion(std::num::TryFromIntError),
    Json(serde_json::Error),
    InvalidRevision,
    InvalidPath(String),
    UnknownArgument(OsString),
    DuplicateArgument(&'static str),
    MissingValue(&'static str),
    MissingRequired(&'static str),
    RevisionEncoding(OsString),
    GitFailed {
        operation: String,
        status: ExitStatus,
    },
    GitProtocol(&'static str),
    Limit {
        kind: &'static str,
        bytes: u64,
    },
}

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Io(error) => error.fmt(f),
            Self::Utf8(error) => error.fmt(f),
            Self::ParseSize(error) => error.fmt(f),
            Self::SizeConversion(error) => error.fmt(f),
            Self::Json(error) => error.fmt(f),
            Self::InvalidRevision => f.write_str("invalid revision"),
            Self::InvalidPath(path) => write!(f, "invalid Git tree path: {path:?}"),
            Self::UnknownArgument(arg) => write!(
                f,
                "unknown argument: {arg:?}; usage: gomad-source-size --repo REPOSITORY --rev REVISION [--output NEW_FILE]"
            ),
            Self::DuplicateArgument(arg) => write!(f, "duplicate argument: {arg}"),
            Self::MissingValue(arg) => write!(f, "missing argument value: {arg}"),
            Self::MissingRequired(arg) => write!(f, "{arg} is required"),
            Self::RevisionEncoding(value) => write!(f, "revision must be UTF-8: {value:?}"),
            Self::GitFailed { operation, status } => write!(f, "git {operation} failed: {status}"),
            Self::GitProtocol(message) => f.write_str(message),
            Self::Limit { kind, bytes } => write!(f, "{kind} exceeds {bytes} byte limit"),
        }
    }
}

impl std::error::Error for Error {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Io(error) => Some(error),
            Self::Utf8(error) => Some(error),
            Self::ParseSize(error) => Some(error),
            Self::SizeConversion(error) => Some(error),
            Self::Json(error) => Some(error),
            _ => None,
        }
    }
}

impl From<std::io::Error> for Error {
    fn from(error: std::io::Error) -> Self {
        Self::Io(error)
    }
}
impl From<std::str::Utf8Error> for Error {
    fn from(error: std::str::Utf8Error) -> Self {
        Self::Utf8(error)
    }
}
impl From<std::num::ParseIntError> for Error {
    fn from(error: std::num::ParseIntError) -> Self {
        Self::ParseSize(error)
    }
}
impl From<std::num::TryFromIntError> for Error {
    fn from(error: std::num::TryFromIntError) -> Self {
        Self::SizeConversion(error)
    }
}
impl From<serde_json::Error> for Error {
    fn from(error: serde_json::Error) -> Self {
        Self::Json(error)
    }
}
