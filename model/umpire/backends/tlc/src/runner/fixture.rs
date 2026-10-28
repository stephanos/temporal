//! Closed protocol-only subprocesses. All shell structure is owned here; cases are data.
use super::*;
use std::os::unix::fs::PermissionsExt;
use umpire_backend_process::work::{FixtureAssertions, ProtocolFixture, TlcInvocation};

#[derive(Debug)]
pub(super) enum Case {
    Protocol {
        output: String,
        status: u8,
    },
    Resource {
        events: PathBuf,
        exploration: String,
        oversized: String,
    },
    WorkerPeak {
        events: PathBuf,
        complete: String,
    },
    Recovery {
        events: PathBuf,
        exploration: String,
        batch: String,
        singletons: [(String, u8); 2],
        fallback: String,
    },
    Cache,
}
fn quoted(text: &str) -> String {
    format!("'{}'", text.replace('\'', "'\\''"))
}
impl Case {
    fn render(&self) -> String {
        let body = match self {
            Self::Protocol { output, status } => {
                format!("printf '%s' {}\nexit {status}", quoted(output))
            }
            Self::Resource { events, exploration, oversized } => format!(
                "count=0\nwhile read -r kind property; do\n case $kind in INVARIANT|PROPERTY) count=$((count + 1));; esac\ndone < Run.cfg\nprintf '%s\\n' \"$count\" >> {}\nif [ \"$count\" = 0 ]; then printf '%s' {}; else printf '%s' {}; fi",
                quoted(&events.to_string_lossy()),
                quoted(exploration),
                quoted(oversized),
            ),
            Self::WorkerPeak { events, complete } => format!(
                "events={}\nprintf 'start %s\\n' \"$PWD\" >> \"$events\"\nround=0\nwhile [ \"$round\" -lt 100000 ]; do\n starts=0\n while read -r event path; do\n  if [ \"$event\" = start ]; then starts=$((starts + 1)); fi\n done < \"$events\"\n if [ \"$starts\" -ge 2 ]; then break; fi\n round=$((round + 1))\ndone\nround=0\nwhile [ \"$round\" -lt 10000 ]; do round=$((round + 1)); done\nproperty=''\nwhile read -r kind value; do\n case $kind in INVARIANT|PROPERTY) property=$value;; esac\ndone < Run.cfg\nprintf '%s\\n' \"$property\"\nprintf '%s' {}\nprintf 'end %s\\n' \"$PWD\" >> \"$events\"",
                quoted(&events.to_string_lossy()),
                quoted(complete),
            ),
            Self::Recovery { events, exploration, batch, singletons, fallback } => {
                let [(p0, p0_status), (p1, p1_status)] = singletons;
                format!(
                    "events={}\ncount=0\nproperty=''\nwhile read -r kind value; do\n case $kind in INVARIANT|PROPERTY) count=$((count + 1)); property=$value;; esac\ndone < Run.cfg\ncase $count in\n 0) printf 'explore\\n' >> \"$events\"; printf '%s' {};;\n 1) printf '%s\\n' \"$property\" >> \"$events\"\n    case $property in\n     P0) printf '%s' {}; exit {p0_status};;\n     P1) printf '%s' {}; exit {p1_status};;\n     *) printf '%s' {};;\n    esac;;\n *) printf 'batch\\n' >> \"$events\"; printf '%s' {}; exit 12;;\nesac",
                    quoted(&events.to_string_lossy()),
                    quoted(exploration),
                    quoted(p0),
                    quoted(p1),
                    quoted(fallback),
                    quoted(batch),
                )
            }
            Self::Cache => {
                return r#"#!/bin/bash
set -eu
base=$(dirname "$0")
if [ "${1:-}" = '-version' ]; then
 if [ "$(cat "$base/mode")" = 'identity-failure' ]; then exit 1; fi
 cat "$base/java-version"; echo version >> "$base/identities"; exit 0
fi
config=''
dump=''
while [ "$#" -gt 0 ]; do
 case "$1" in
 -dump) shift; dump=$1 ;;
 -config) shift; config=$1 ;;
 esac
 shift
done
echo run >> "$base/runs"
mode=$(cat "$base/mode")
if [ "$mode" = 'failure' ]; then exit 1; fi
if [ "$mode" = 'module-failure' ] && /usr/bin/grep -q 'Progress_' "$config"; then exit 1; fi
message() { printf '@!@!@STARTMSG %s:%s @!@!@\n%s\n@!@!@ENDMSG %s @!@!@\n' "$1" "$2" "$3" "$1"; }
if [ "$mode" = 'module-failure' ]; then
 : > "$dump"
 message 2199 0 '0 states generated, 0 distinct states found, 0 states left on queue.'
else
 printf 'State 1:\nx = TRUE\n' > "$dump"
 message 2199 0 '1 states generated, 1 distinct states found, 0 states left on queue.'
fi
if [ "$mode" = 'bad-dump' ]; then printf 'State 1:\nx = 99\n' > "$dump"; fi
if [ "$mode" = 'evaluation' ] && /usr/bin/grep -q 'INVARIANT A' "$config"; then
 message 2111 1 $'Evaluating invariant A failed.\nOverflow'
 exit 1
fi
if [ "$mode" = 'refuted' ] && /usr/bin/grep -q 'INVARIANT A' "$config"; then
 message 2110 1 'Invariant A is violated.'
 message 2217 4 $'1: <Initial predicate>\nx = TRUE'
 message 2186 0 Finished
 exit 12
fi
if [ "$mode" = 'mixed' ] && /usr/bin/grep -q 'INVARIANT B' "$config"; then
 message 2110 1 'Invariant B is violated.'
 message 2217 4 $'1: <Initial predicate>\nx = TRUE'
 message 2186 0 Finished
 exit 12
fi
message 2193 0 'Model checking completed. No error has been found.'
message 2186 0 Finished
"#
                .to_owned();
            }
        };
        format!(
            "#!/bin/bash\nset -eu\ndump=''\nwhile [ \"$#\" -gt 0 ]; do\n if [ \"$1\" = '-dump' ]; then shift; dump=$1; fi\n shift\ndone\n: > \"$dump\"\n{body}\n"
        )
    }
    fn family(&self) -> &'static str {
        if matches!(self, Self::Cache) { "tlc-cache-v1" } else { "tlc-batch-v1" }
    }
}

fn assertions(root: &Path) -> Result<FixtureAssertions, Refusal> {
    let test = fs::read_to_string(root.join("fixture-test"))?;
    Ok(FixtureAssertions::pin(root, &test)?)
}
pub(super) fn install(root: &Path, case: &Case) -> Result<TlcInvocation, Refusal> {
    if !root.join("fixture-test").exists() {
        let thread = std::thread::current();
        let name = thread
            .name()
            .ok_or_else(|| Refusal("protocol fixture lacks named assertion owner".into()))?;
        fs::write(root.join("fixture-test"), name)?;
    }
    fs::write(root.join("java"), case.render())?;
    fs::set_permissions(root.join("java"), fs::Permissions::from_mode(0o700))?;
    invocation(root, case)
}
pub(super) fn invocation(root: &Path, case: &Case) -> Result<TlcInvocation, Refusal> {
    if fs::read(root.join("java"))? != case.render().as_bytes() {
        return Err(Refusal("closed protocol fixture renderer changed".into()));
    }
    let data = if matches!(case, Case::Cache) {
        let mode = fs::read_to_string(root.join("mode"))?;
        if ![
            "verified",
            "identity-failure",
            "failure",
            "module-failure",
            "bad-dump",
            "evaluation",
            "refuted",
            "mixed",
        ]
        .contains(&mode.as_str())
        {
            return Err(Refusal("unknown closed cache protocol mode".into()));
        }
        vec![root.join("mode"), root.join("java-version"), root.join("tlc.jar")]
    } else {
        vec![]
    };
    Ok(TlcInvocation::fixture(ProtocolFixture::pin(
        assertions(root)?,
        &root.join("java"),
        case.family(),
        include_bytes!("fixture.rs"),
        format!("{case:?}").as_bytes(),
        &data,
    )?))
}
pub(super) fn complete(root: &Path) -> Result<(), Refusal> {
    if root.join("fixture-test").exists() {
        assertions(root)?.complete()?;
    }
    Ok(())
}

#[test]
fn closed_protocol_treats_case_as_data_and_clears_shell_startup_environment() -> Result<(), Refusal>
{
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
    let scratch = Scratch::new(&root)?;
    let result = (|| {
        let marker = scratch.path().join("unexpected-startup");
        let startup = scratch.path().join("startup");
        fs::write(&startup, format!("printf bad > {}\n", quoted(&marker.to_string_lossy())))?;
        let output = format!(
            "literal ' quote $(printf injected > {}) ${{HOME}}\n",
            quoted(&marker.to_string_lossy())
        );
        let origin =
            install(scratch.path(), &Case::Protocol { output: output.clone(), status: 37 })?;
        let mut command = std::process::Command::new(scratch.path().join("java"));
        command.args(["tlc2.TLC", "-dump"]).arg(scratch.path().join("dump"));
        command.env("BASH_ENV", startup).env("BASH_FUNC_printf%%", "() { exit 99; }");
        let (status, captured) = run_process_observed(
            &mut command,
            scratch.path(),
            "clean-fixture",
            Limits::default(),
            None,
            &origin,
        )?;
        assert_eq!(status.code(), Some(37));
        assert_eq!(captured, output);
        assert!(!marker.exists());
        complete(scratch.path())?;
        Ok(())
    })();
    scratch.finish(result)
}

#[test]
fn changed_renderer_cannot_be_reclassified_as_closed_protocol() -> Result<(), Refusal> {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("../..").canonicalize()?;
    let scratch = Scratch::new(&root)?;
    let result = (|| {
        let case = Case::Protocol { output: String::new(), status: 0 };
        install(scratch.path(), &case)?;
        fs::write(scratch.path().join("java"), "#!/bin/bash\nset -eu\nexit 0\n")?;
        assert!(invocation(scratch.path(), &case).is_err());
        Ok(())
    })();
    scratch.finish(result)
}
