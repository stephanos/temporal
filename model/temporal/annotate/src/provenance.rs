//! Compile-time checks for declared validator provenance.

/// Whether a validator identity contains anything except Unicode `White_Space`.
pub const fn validator_has_name(value: &str) -> bool {
    let bytes = value.as_bytes();
    let mut index = 0;
    while index < bytes.len() {
        let first = bytes[index];
        let width = if matches!(first, b'\t'..=b'\r' | b' ') {
            1
        } else if first == 0xc2
            && index + 1 < bytes.len()
            && matches!(bytes[index + 1], 0x85 | 0xa0)
        {
            2
        } else if index + 2 < bytes.len()
            && ((first == 0xe1 && bytes[index + 1] == 0x9a && bytes[index + 2] == 0x80)
                || (first == 0xe2
                    && bytes[index + 1] == 0x80
                    && matches!(bytes[index + 2], 0x80..=0x8a | 0xa8 | 0xa9 | 0xaf))
                || (first == 0xe2 && bytes[index + 1] == 0x81 && bytes[index + 2] == 0x9f)
                || (first == 0xe3 && bytes[index + 1] == 0x80 && bytes[index + 2] == 0x80))
        {
            3
        } else {
            return true;
        };
        index += width;
    }
    false
}
