//! The same transition system as an SMT-LIB 2 bounded model check: BTOR2 lines unrolled frame
//! by frame, every value a bitvector, piped to a solver binary (what `easy-smt` wraps).
use std::collections::BTreeMap;
use std::fmt::Write as _;
use std::io::Write as _;
use std::process::{Command, Stdio};

struct Node {
    op: String,
    width: u32,
    args: Vec<String>,
}

/// Parses the emitted BTOR2 text into nodes by id; sorts resolve to widths.
fn parse(text: &str) -> Result<(BTreeMap<u64, Node>, Vec<(u64, u64)>, Vec<(u64, u64)>, Vec<u64>, Vec<u64>), String> {
    let mut sorts = BTreeMap::new();
    let mut nodes = BTreeMap::new();
    let mut inits = Vec::new();
    let mut nexts = Vec::new();
    let mut constraints = Vec::new();
    let mut bads = Vec::new();
    for line in text.lines() {
        let line = line.trim();
        if line.is_empty() || line.starts_with(';') {
            continue;
        }
        let parts: Vec<&str> = line.split_whitespace().collect();
        let id: u64 = parts[0].parse().map_err(|_| format!("bad id in {line}"))?;
        match parts[1] {
            "sort" => {
                sorts.insert(id, parts[3].parse::<u32>().map_err(|_| "bad width")?);
            }
            "init" => inits.push((parts[3].parse().map_err(|_| "init")?, parts[4].parse().map_err(|_| "init")?)),
            "next" => nexts.push((parts[3].parse().map_err(|_| "next")?, parts[4].parse().map_err(|_| "next")?)),
            "constraint" => constraints.push(parts[2].parse().map_err(|_| "constraint")?),
            "bad" => bads.push(parts[2].parse().map_err(|_| "bad")?),
            op => {
                let width = *sorts.get(&parts[2].parse::<u64>().map_err(|_| "sort ref")?).ok_or("unknown sort")?;
                nodes.insert(id, Node { op: op.to_owned(), width, args: parts[3..].iter().map(|s| (*s).to_owned()).collect() });
            }
        }
    }
    Ok((nodes, inits, nexts, constraints, bads))
}

fn bool_of(e: &str) -> String {
    format!("(= {e} #b1)")
}
fn bv_of(cond: &str) -> String {
    format!("(ite {cond} #b1 #b0)")
}

fn term(node: &Node, k: usize) -> String {
    let n = |a: &str| format!("n{a}_{k}");
    let a = node.args.first().map(|s| n(s)).unwrap_or_default();
    let b = node.args.get(1).map(|s| n(s)).unwrap_or_default();
    let c = node.args.get(2).map(|s| n(s)).unwrap_or_default();
    match node.op.as_str() {
        "const" => format!("#b{}", node.args[0]),
        "not" => format!("(bvnot {a})"),
        "neg" => format!("(bvneg {a})"),
        "and" => format!("(bvand {a} {b})"),
        "or" => format!("(bvor {a} {b})"),
        "implies" => format!("(bvor (bvnot {a}) {b})"),
        "eq" => bv_of(&format!("(= {a} {b})")),
        "neq" => bv_of(&format!("(not (= {a} {b}))")),
        "ult" | "ulte" | "ugt" | "ugte" | "slt" | "slte" | "sgt" | "sgte" => {
            let rel = match node.op.as_str() {
                "ult" => "bvult",
                "ulte" => "bvule",
                "ugt" => "bvugt",
                "ugte" => "bvuge",
                "slt" => "bvslt",
                "slte" => "bvsle",
                "sgt" => "bvsgt",
                _ => "bvsge",
            };
            bv_of(&format!("({rel} {a} {b})"))
        }
        "add" => format!("(bvadd {a} {b})"),
        "sub" => format!("(bvsub {a} {b})"),
        "mul" => format!("(bvmul {a} {b})"),
        "sdiv" => format!("(bvsdiv {a} {b})"),
        "srem" => format!("(bvsrem {a} {b})"),
        "udiv" => format!("(bvudiv {a} {b})"),
        "urem" => format!("(bvurem {a} {b})"),
        "ite" => format!("(ite {} {b} {c})", bool_of(&a)),
        "sext" => format!("((_ sign_extend {}) {a})", node.args[1]),
        "uext" => format!("((_ zero_extend {}) {a})", node.args[1]),
        other => format!("; unsupported {other}"),
    }
}

/// Bounded check of the single `bad` in `btor` up to `bound` steps; returns the depth of the
/// first violation, or `None`, and the solver wall time.
pub fn bmc(solver: &str, btor: &str, bound: usize) -> Result<(Option<usize>, std::time::Duration), String> {
    let (nodes, inits, nexts, constraints, bads) = parse(btor)?;
    let bad = *bads.first().ok_or("no bad")?;
    let mut script = String::from("(set-logic QF_BV)\n(set-option :produce-models false)\n");
    let frame = |script: &mut String, k: usize| {
        for (id, node) in &nodes {
            match node.op.as_str() {
                "state" | "input" => {
                    let _ = writeln!(script, "(declare-const n{id}_{k} (_ BitVec {}))", node.width);
                }
                _ => {
                    let _ = writeln!(script, "(define-fun n{id}_{k} () (_ BitVec {}) {})", node.width, term(node, k));
                }
            }
        }
        for c in &constraints {
            let _ = writeln!(script, "(assert (= n{c}_{k} #b1))");
        }
    };
    frame(&mut script, 0);
    for (state, value) in &inits {
        let _ = writeln!(script, "(assert (= n{state}_0 n{value}_0))");
    }
    for k in 0..=bound {
        let _ = writeln!(script, "(push 1)\n(assert (= n{bad}_{k} #b1))\n(check-sat)\n(pop 1)");
        if k < bound {
            frame(&mut script, k + 1);
            for (state, value) in &nexts {
                let _ = writeln!(script, "(assert (= n{state}_{} n{value}_{k}))", k + 1);
            }
        }
    }
    let started = std::time::Instant::now();
    let mut child = Command::new(solver).arg("-in").stdin(Stdio::piped()).stdout(Stdio::piped()).stderr(Stdio::piped()).spawn().map_err(|e| format!("{solver}: {e}"))?;
    child.stdin.take().ok_or("stdin")?.write_all(script.as_bytes()).map_err(|e| e.to_string())?;
    let output = child.wait_with_output().map_err(|e| e.to_string())?;
    let elapsed = started.elapsed();
    let stdout = String::from_utf8_lossy(&output.stdout);
    let mut depth = None;
    for (k, line) in stdout.lines().enumerate() {
        match line.trim() {
            "sat" => {
                depth = Some(k);
                break;
            }
            "unsat" => {}
            other => return Err(format!("solver said {other}: {}", String::from_utf8_lossy(&output.stderr))),
        }
    }
    Ok((depth, elapsed))
}
