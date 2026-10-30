use std::{env, fs, process::ExitCode};
fn quoted(value: &str) -> String {
    let mut out = String::from("\"");
    for c in value.chars() {
        match c {
            '"' => out.push_str("\\\""), '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"), '\r' => out.push_str("\\r"), '\t' => out.push_str("\\t"),
            c if c.is_control() => out.push_str(&format!("\\u{:04x}", c as u32)),
            c => out.push(c),
        }
    }
    out.push('"'); out
}
fn report(path: &str, source: &str, outcome: &str, line: usize, column: usize, message: &str) {
    let snippet = source.lines().nth(line.saturating_sub(1)).unwrap_or("");
    println!("{{\"path\":{},\"outcome\":{},\"line\":{line},\"column\":{column},\"snippet\":{},\"message\":{}}}",
        quoted(path), quoted(outcome), quoted(snippet), quoted(message));
}
fn main() -> ExitCode {
    let paths: Vec<_> = env::args().skip(1).collect();
    if paths.is_empty() { return ExitCode::FAILURE; }
    for path in paths {
        let source = match fs::read_to_string(&path) {
            Ok(source) => source,
            Err(error) => { report(&path, "", "io_error", 0, 0, &error.to_string()); return ExitCode::FAILURE; }
        };
        let ast = match syn::parse_file(&source) {
            Ok(ast) => ast,
            Err(error) => { let at = error.span().start(); report(&path, &source, "parse_error", at.line, at.column + 1, &error.to_string()); continue; }
        };
        match umpire_front::read(&ast, &[]) {
            Ok(_) => report(&path, &source, "admitted", 0, 0, ""),
            Err(umpire_front::Refusal::Model(d)) => report(&path, &source, "model_refusal", d.line, d.column, &d.message),
            Err(umpire_front::Refusal::Hook(index, d)) => report(&path, &source, "hook_refusal", d.line, d.column, &format!("hook[{index}]: {}", d.message)),
        }
    }
    ExitCode::SUCCESS
}
