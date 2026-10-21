pub mod explore;
pub mod inspect;
pub mod merge;
pub mod replay;
pub mod resume;

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
