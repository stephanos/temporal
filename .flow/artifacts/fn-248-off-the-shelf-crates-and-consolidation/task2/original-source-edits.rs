fn replace_once(source: &str, from: &str, to: &str) -> String {
    assert_eq!(source.matches(from).count(), 1, "mutation target: {from}");
    source.replacen(from, to, 1)
}

fn replace_in(source: &str, start: &str, end: &str, from: &str, to: &str) -> String {
    assert_eq!(source.matches(start).count(), 1, "region start: {start}");
    let begin = source.find(start).unwrap_or_else(|| panic!("missing {start}"));
    let finish = begin + source[begin..].find(end).unwrap_or_else(|| panic!("missing {end}"));
    let changed = replace_once(&source[begin..finish], from, to);
    format!("{}{}{}", &source[..begin], changed, &source[finish..])
}

fn replace_code(source: &str, from: &str, to: &str) -> String {
    let offsets: Vec<_> = source.char_indices().filter(|(_, ch)| !ch.is_whitespace()).collect();
    let compact: String = offsets.iter().map(|(_, ch)| *ch).collect();
    let anchor: String = from.chars().filter(|ch| !ch.is_whitespace()).collect();
    assert_eq!(compact.matches(&anchor).count(), 1, "mutation anchor: {from}");
    let byte = compact.find(&anchor).unwrap_or_else(|| unreachable!());
    let start = compact[..byte].chars().count();
    let last = start + anchor.chars().count() - 1;
    let mut changed = source.to_owned();
    changed.replace_range(offsets[start].0..offsets[last].0 + offsets[last].1.len_utf8(), to);
    changed
}



#[test]
fn source_edits_preserve_surrounding_text_and_unicode() {
    assert_eq!(replace_once("before target after", "target", "new"), "before new after");
    assert_eq!(replace_in("pre [one] post", "[", "]", "one", "two"), "pre [two] post");
    assert_eq!(replace_code("before α  + β after", "α+β", "new"), "before new after");
}

#[test]
fn source_edits_refuse_missing_duplicate_and_empty_anchors() {
    for source in ["absent", "target target"] {
        assert!(std::panic::catch_unwind(|| replace_once(source, "target", "new")).is_err());
        assert!(std::panic::catch_unwind(|| replace_code(source, "target", "new")).is_err());
    }
    for (source, start, end, from) in [
        ("missing", "[", "]", "x"),
        ("[x][x]", "[", "]", "x"),
        ("[x", "[", "]", "x"),
        ("[x]]", "[", "]", "x"),
        ("[y]", "[", "]", "x"),
        ("[xx]", "[", "]", "x"),
    ] {
        assert!(std::panic::catch_unwind(|| replace_in(source, start, end, from, "new")).is_err());
    }
    assert!(std::panic::catch_unwind(|| replace_code("", "", "new")).is_err());
}
