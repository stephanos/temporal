/// Replaces exactly one literal mutation target, refusing missing or duplicate matches.
pub fn replace_once(source: &str, from: &str, to: &str) -> String {
    assert_eq!(source.matches(from).count(), 1, "mutation target: {from}");
    source.replacen(from, to, 1)
}

/// Replaces one mutation target within a uniquely delimited source region.
pub fn replace_in(source: &str, start: &str, end: &str, from: &str, to: &str) -> String {
    assert_eq!(source.matches(start).count(), 1, "region start: {start}");
    let begin = source.find(start).unwrap_or_else(|| panic!("missing {start}"));
    assert_eq!(source[begin..].matches(end).count(), 1, "region end: {end}");
    let finish = begin + source[begin..].find(end).unwrap_or_else(|| panic!("missing {end}"));
    let changed = replace_once(&source[begin..finish], from, to);
    format!("{}{}{}", &source[..begin], changed, &source[finish..])
}

/// Replaces exactly one code anchor while ignoring whitespace and preserving surrounding text.
pub fn replace_code(source: &str, from: &str, to: &str) -> String {
    let offsets: Vec<_> = source.char_indices().filter(|(_, ch)| !ch.is_whitespace()).collect();
    let compact: String = offsets.iter().map(|(_, ch)| *ch).collect();
    let anchor: String = from.chars().filter(|ch| !ch.is_whitespace()).collect();
    assert!(!anchor.is_empty(), "empty mutation anchor");
    assert_eq!(compact.matches(&anchor).count(), 1, "mutation anchor: {from}");
    let byte = compact.find(&anchor).unwrap_or_else(|| unreachable!());
    let start = compact[..byte].chars().count();
    let last = start + anchor.chars().count() - 1;
    let mut changed = source.to_owned();
    changed.replace_range(offsets[start].0..offsets[last].0 + offsets[last].1.len_utf8(), to);
    changed
}
