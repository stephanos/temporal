# Adapter command BASE observations

Task40's bounded execution must preserve the public helper's measured ordinary errors. Root independently reproduced the research probe at source commit 0c2e091b73325368b22ec142e76da5ed5ab4cad2. Both runs completed 29 observations with exit 0. These observations resolve the cancellation/startup design fork; production remains unchanged.

## Actual boundary

The fixture calls target.AdapterPreparedSourceSetSHA256 from the unchanged nested module. It launches the probe executable as the Go-command stand-in, emits controlled stdout/stderr, records an acknowledged started marker, and observes the public digest or error. The helper still performs its real command setup, JSON decoding, projection and checked GOPATH cleanup.

| Case | Observed cause and precedence |
| --- | --- |
| After acknowledged startup, caller cancel or deadline | fmt.wrapError -> actual exec.ExitError, SIGKILL 9, ExitCode -1; both context Is checks false |
| Before startup, existing executable or absolute missing executable | Corresponding context sentinel, outer message ends with a colon and space |
| Before startup, missing bare PATH executable | exec.Error -> exec.ErrNotFound wins over canceled/expired context |
| Exit 7 with valid JSON, malformed JSON or listed error | exec.ExitError wins before decoding or listed-error handling |
| Ordinary SIGTERM | exec.ExitError retains signal 15 and ExitCode -1 |
| Missing, nonexecutable or invalid-format executable | os.PathError operation fork/exec, original command path, errno 2/13/8 |
| Invalid directory | os.PathError operation chdir, original directory path, errno 2 |
| Empty command | exec: no command |
| Empty directory or dot | Inherits the nested-module cwd |
| Relative directory and ./ executable | Resolves using the supplied child directory |

Captured stderr appears trimmed in the existing outer error. The actual exec.ExitError.Stderr field remains empty. Converting these outcomes to ctx.Err or a status-only synthetic ExitError would change the public contract.

Both completed runs observed 18 started children, found every one gone through kill(pid, 0) returning ESRCH, and found their 18 observed temporary GOPATH paths absent after return. Cases without a started marker do not expose the GOPATH; their false ChildGone/GOPATHRemoved fields mean unobserved, not a detected leak.

## Evidence and reproduction

probe.go is the exact final scratch source. original-run.sh is the executed bounded runner with its original machine-specific paths. research and conductor hold raw JSONL, stderr, actual exit, host and before/after input/status bindings. Root's conductor run took 2.671 seconds and produced the same case classes and precedence. Its volatile PIDs, temporary paths, timestamps and executable paths differ from the research run as expected.

To reproduce locally, use a fresh mktemp output directory, the pinned cached stock Go1.27.1, the nested-module cwd and the environment recorded in original-run.sh. Run probe.go through go run -tags test_dep with a finite external timeout. Do not reuse an output directory; each case creates its own fixture directory.

The first harness run exited 1 on a fixture-name collision after producing earlier observations. Its raw partial output/error and input bindings are retained under initial-failed-harness. The source changed to fix that scratch collision before the final run; its original preimage was not retained. This harness failure is neither a production behavioral RED nor evidence for the final source.

Root checked all 26 archive copies against their originals, both complete raw JSONL runs, unchanged helper/module/tool/probe input bindings, unchanged repository status during execution and the all-started-child/GOPATH predicates. Source-tree hashes bind the shared hostexec/gocommand and target inputs. Independent plan review reached SHIP after the task40/task9 completion-dependency fix. Both review rounds are retained here. The review supplies no implementation or formal source verdict.

## Limits and next action

This is developmental linux/arm64 stock-Go evidence. The darwin/arm64 strings are helper arguments, not the execution host. The controlled successful JSON has an empty source list and produces the observed digest sha256:2d1d7ba39a9b940ad19d0e739e4d4d01e1c14b3aa19b1ff11309260484dec980. It proves no actual Go/foreign file selection, pin reproduction, largest-listing size or native qualification. The fixture exercises self-contained children, not descendants or stream overflow.

The long-relative-path case's name claims child-versus-parent resolution, but its many parent traversals converge at the filesystem root. Treat it only as an observed relative-path success; the ./ cases establish resolution inside the supplied directory. There is no claim of race-complete or universal cancellation equivalence.

Implement task40's narrowly opted-in raw-cause compatibility over the existing bounded mechanism. Keep default hostexec/Structured/Diagnostic consumers unchanged; preserve real leader Kill/Wait outcomes before descendant cleanup. Task9 retains adapter routing, nonempty source-set controls, the measured capacity choice and architecture guidance. Its existing task8 dependency remains. All original source-owned qualification stays required; transferred Linux proof stays under fn128 and nonblocking.
