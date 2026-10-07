import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const artifacts = dirname(fileURLToPath(import.meta.url));
const base = 'a936b597b4c62fa50f11a6c16c91111cd52b1ec3';
const toolchain = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
const lint = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0';
const errortype = '/tmp/fn109-lint-tools.ZdNe1t50/errortype';
const env = { ...process.env, PATH: toolchain + ':/usr/local/bin:/usr/bin:/bin', GOPROXY: 'off', GOSUMDB: 'off', GOTOOLCHAIN: 'local', GOWORK: 'off', GOENV: 'off', GOFLAGS: '', GOMAXPROCS: '2' };
for (const name of ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED']) delete env[name];
const sources = ['internal/gomadtool/architecture/standard.go', 'internal/gomadtool/conformance/runtime_choice.go', 'runner/internal/execution/process_unix.go', 'qualification/set/manifestgen/manifestgen.go'].map(p => 'tools/gomad3/' + p);
const pins = ['tools/gomad3/go.mod', 'tools/gomad3/go.sum', '.github/.golangci.yml', 'Makefile', 'tools/gomad3/Makefile', 'tools/gomad3integration/qualification/tests.generator.json', 'tools/gomad3integration/qualification/tests.json'];
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
const compact = (phase, run) => {
  const events = (run.stdout || '').split('\n').flatMap(line => { try { const e = JSON.parse(line); return e.Action && e.Test ? [e] : []; } catch { return []; } });
  const terminal = run.terminal;
  const packages = [...new Set(terminal.map(e => e[0]))];
  run.test_counts = Object.fromEntries(['pass', 'fail', 'skip'].map(a => [a, terminal.filter(e => e[2] === a).length]));
  run.package_terminals = Object.fromEntries(packages.map(p => [p, { count: terminal.filter(e => e[0] === p).length, sha256: hash(JSON.stringify(terminal.filter(e => e[0] === p).sort())) }]));
  run.failures_and_skips = terminal.filter(e => e[2] !== 'pass').map(([pkg, test, action]) => ({ package: pkg, test, action, output: events.filter(e => e.Package === pkg && e.Test === test && e.Action === 'output').map(e => e.Output).join('') }));
  run.raw = {};
  for (const stream of ['stdout', 'stderr']) {
    const data = run[stream] || '';
    const path = '.flow/tmp/task45-' + phase + '-' + run.label + '.' + stream + '.log';
    writeFileSync(resolve(root, path), data);
    run.raw[stream] = { path, bytes: Buffer.byteLength(data), sha256: hash(data) };
    run[stream + '_tail'] = terminal.length ? '' : data.split('\n').slice(-16).join('\n');
    delete run[stream];
  }
  run.diagnostics = readFileSync(resolve(root, run.raw.stdout.path), 'utf8').split('\n').filter(line => /^tools\/gomad3\/.*:\d+:\d+: /.test(line));
  delete run.terminal;
  return run;
};
const identities = () => Object.fromEntries([...sources, ...pins, toolchain + '/go', toolchain + '/gofmt', lint, errortype].map(p => [p, hash(readFileSync(resolve(root, p)))]));
const packages = ['./internal/gomadtool/architecture', './internal/gomadtool/conformance', './runner/internal/execution', './qualification/set/manifestgen'];
const execution = '^Test(RunIOTerminalAfterTermination|ValidateRequestRejectsExpectedIOTranscriptOutsideReplay|ValidateRequestAcceptsNestedExecutionCapabilities|StageFilesAreBuiltOnlyFromTheDescriptorPlan|LaunchResourcesOwnPipeCreationAndInheritance|DescriptorPlanOwnsEveryStageLayout|LaunchDescriptorNumbersRemainStable|ProtocolErrorCleanupRetainsTrustedTargetIdentity|EarlySupervisorCleanupIgnoresUntrustedReportedTargetIdentity|GroupProbeRequiresExplicitESRCHForDisappearance|ProbeFailureDoesNotBypassTargetReap|SupervisorReportStateMachineRejectsMalformedSequences|ClassifyOwnsRecordingAndReplaySemantics|Compose.*|ValidateRejectsSemanticAndTransitionDivergence)$';
const makeTools = ['GOLANGCI_LINT_FIX=false', 'GOLANGCI_LINT=' + lint, 'ERRORTYPE=' + errortype];
if (process.argv[2] === 'proof') {
  const git = args => {
    const r = spawnSync('git', args, { cwd: root, encoding: 'utf8' });
    if (r.status !== 0) throw new Error(r.stderr);
    return r.stdout;
  };
  const baseline = JSON.parse(readFileSync(resolve(artifacts, 'baseline.json')));
  const final = JSON.parse(readFileSync(resolve(artifacts, 'final.json')));
  const expected = sources.map(p => git(['show', base + ':' + p]));
  const once = (text, old, replacement) => {
    if (text.split(old).length !== 2) throw new Error('non-unique expected form: ' + old);
    return text.replace(old, replacement);
  };
  for (const [old, replacement] of [
    ['\t\tif id == "fmt.Errorf" {', '\t\tswitch id {\n\t\tcase "fmt.Errorf":'],
    ['\t\t} else if id == "fmt.Sprintf" || id == "fmt.Printf" {', '\t\tcase "fmt.Sprintf", "fmt.Printf":'],
    ['\t\t} else if id == "fmt.Appendf" {', '\t\tcase "fmt.Appendf":'],
    ['\t\t} else if id == "fmt.Append" || id == "fmt.Appendln" {', '\t\tcase "fmt.Append", "fmt.Appendln":'],
    ['\t\t\tif id == "encoding/json.*Decoder.Decode" {', '\t\t\tswitch id {\n\t\t\tcase "encoding/json.*Decoder.Decode":'],
    ['\t\t\t} else if id == "encoding/json.*Decoder.Token" {', '\t\t\tcase "encoding/json.*Decoder.Token":']
  ]) expected[0] = once(expected[0], old, replacement);
  expected[1] = once(expected[1], 'err != nil && !(result.ExitCode == 125 && errors.Is(err, choice.ErrDiverged))', 'err != nil && (result.ExitCode != 125 || !errors.Is(err, choice.ErrDiverged))');
  for (const [variable, kind] of [['err', 'choice.ErrDiagnosticIncomplete'], ['collected.err', 'deterministicio.ErrTranscriptUnterminated'], ['choiceErr', 'ErrChoiceTraceUnterminated']]) {
    expected[2] = once(expected[2], variable + ' != nil && !(errors.Is(' + variable + ', ' + kind + ') && (result.WatchdogTimeout || result.Cancelled))', variable + ' != nil && (!errors.Is(' + variable + ', ' + kind + ') || (!result.WatchdogTimeout && !result.Cancelled))');
  }
  const receiverStart = expected[3].indexOf('func (resolved retention) validate() error {');
  const receiverEnd = expected[3].indexOf('\n}\n', receiverStart) + 2;
  if (receiverStart < 0 || receiverEnd <= receiverStart) throw new Error('receiver method missing');
  expected[3] = expected[3].slice(0, receiverStart) + expected[3].slice(receiverStart, receiverEnd).replaceAll('resolved', 'inherited') + expected[3].slice(receiverEnd);
  const observations = label => {
    const before = baseline.runs.find(r => r.label === label), after = final.runs.find(r => r.label === label);
    const blocks = run => readFileSync(resolve(root, run.raw.stdout.path), 'utf8').split(/(?=^tools\/gomad3\/.*:\d+:\d+: )/m).filter(s => /^tools\/gomad3\//.test(s)).map(s => s.split(/\n\d+ issues:/)[0].trimEnd().replace(/^(tools\/gomad3\/.*):\d+:\d+: /, '$1:<position>: '));
    const old = blocks(before), current = blocks(after);
    const residual = [...old];
    const added = [];
    for (const block of current) {
      const index = residual.indexOf(block);
      if (index < 0) added.push(block); else residual.splice(index, 1);
    }
    return { baseline_count: old.length, final_count: current.length, removed: residual.map(s => s.split('\n')[0]), added: added.map(s => s.split('\n')[0]), residual_multiset_sha256: hash(JSON.stringify(current.sort())), complete_residual_blocks_match_after_position_normalization: added.length === 0 && residual.length === 7, final_headers: after.diagnostics };
  };
  const changed = git(['diff', '--name-only', base, '--', 'tools/gomad3']).trim().split('\n');
  const countsEqual = (before, after) => Object.entries(after.package_terminals).every(([pkg, data]) => JSON.stringify(before.package_terminals[pkg]) === JSON.stringify(data));
  const checks = {
    exact_seven_forms: sources.every((p, i) => expected[i] === readFileSync(resolve(root, p), 'utf8')),
    only_four_production_files_changed: changed.length === 4 && changed.every(p => sources.includes(p)),
    fixed_inputs_and_tools_preserved: Object.keys(baseline.before).filter(p => !sources.includes(p)).every(p => baseline.before[p] === final.after[p]),
    source_stable_during_commands: baseline.unchanged_inputs && final.unchanged_inputs,
    matching_existing_package_terminals: countsEqual(baseline.runs.find(r => r.label === 'packages'), final.runs.find(r => r.label === 'packages')),
    matching_portable_conformance_execution_architecture_terminals: ['portable-conformance', 'execution', 'architecture'].every(label => countsEqual(baseline.runs.find(r => r.label === label), final.runs.find(r => r.label === label))),
    gofmt_clean: final.runs.find(r => r.label === 'format').exit_code === 0 && final.runs.find(r => r.label === 'format').raw.stdout.bytes === 0,
    standalone_errortype_and_check_only_validation_pass: ['errortype', 'validate'].every(label => final.runs.find(r => r.label === label).exit_code === 0)
  };
  const scoped = observations('lint'), full = observations('full-lint');
  checks.exact_scoped_lint_delta = scoped.baseline_count === 33 && scoped.final_count === 26 && scoped.complete_residual_blocks_match_after_position_normalization;
  checks.exact_original_full_lint_delta = full.baseline_count === 317 && full.final_count === 310 && full.complete_residual_blocks_match_after_position_normalization;
  checks.only_admitted_diagnostics_removed = [scoped, full].every(o => o.removed.filter(s => s.includes('QF1003:')).length === 2 && o.removed.filter(s => s.includes('QF1001:')).length === 4 && o.removed.filter(s => s.includes('ST1016:')).length === 1);
  const output = { base_commit: base, checks, source_hashes: Object.fromEntries(sources.map(p => [p, { baseline: baseline.before[p], final: final.after[p] }])), scoped_lint: { ...scoped, final_headers: undefined }, full_lint: { ...full, final_headers: undefined }, source_diff_sha256: hash(git(['diff', base, '--', ...sources])) };
  writeFileSync(resolve(artifacts, 'proof.json'), JSON.stringify(output, null, 2) + '\n');
  console.log(JSON.stringify(output.checks));
  process.exit(Object.values(checks).every(Boolean) ? 0 : 1);
}
let commands = [
  ['packages', 'tools/gomad3', ['go', 'test', '-json', '-tags', 'test_dep', '-count=1', packages[0], packages[1], packages[3]]],
  ['execution', 'tools/gomad3', ['go', 'test', '-json', '-tags', 'test_dep', '-count=1', '-run', execution, packages[2]]],
  ['architecture', 'tools/gomad3', ['go', 'test', '-json', '-tags', 'test_dep', '-count=1', '-run', 'Test(PackageArchitecture|PureModulesHaveNoHostEffects|ExactModuleEdges|PublicPackagesDoNotExportTypeAliases|DomainModulesDoNotExportWireFraming|RunnerExecutionInjectionIsPrivate)$', '.']],
  ['lint', 'tools/gomad3', [lint, 'run', '--config=../../.github/.golangci.yml', '--build-tags=test_dep', '--timeout=10m', '--fix=false', ...packages]],
  ['errortype', 'tools/gomad3', [errortype, '-test=true', ...packages]],
  ['fast-lint', '.', ['make', 'lint-code-fast', 'GOLANGCI_LINT_BASE_REV=' + base, ...makeTools]],
  ['full-lint', '.', ['make', '--trace', 'lint-code-gomad3', 'GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c', ...makeTools]],
  ['format', '.', [toolchain + '/gofmt', '-l', ...sources]],
  ['validate', '.', ['make', '-C', 'tools/gomad3', 'validate']]
];
const requested = process.argv[2];
if (requested === 'compact-baseline') {
  const path = resolve(artifacts, 'baseline.json');
  const output = JSON.parse(readFileSync(path));
  output.runs = output.runs.map(run => run.raw ? run : compact('baseline', run));
  writeFileSync(path, JSON.stringify(output, null, 2) + '\n');
  console.log('Baseline compacted; raw logs retained under .flow/tmp/task45-*');
  process.exit(0);
}
if (!['baseline', 'portable-baseline', 'final'].includes(requested)) throw new Error('expected baseline, portable-baseline or final');
const phase = requested === 'portable-baseline' ? 'baseline' : requested;
const portable = ['portable-conformance', 'tools/gomad3', ['go', 'test', '-json', '-tags', 'test_dep', '-count=1', '-skip', '^TestRuntimeOwnedControlProbe$', packages[1]]];
if (requested === 'portable-baseline') commands = [portable];
if (phase === 'final') {
  commands[0][2] = ['go', 'test', '-json', '-tags', 'test_dep', '-count=1', packages[0], packages[3]];
  commands.splice(1, 0, portable);
}
const before = identities();
const git = spawnSync('git', ['rev-parse', 'HEAD'], { cwd: root, encoding: 'utf8' });
if (git.status !== 0 || git.stdout.trim() !== base) throw new Error('unexpected HEAD');
const output = requested === 'portable-baseline' ? JSON.parse(readFileSync(resolve(artifacts, 'baseline.json'))) : { base_commit: base, phase, before, runs: [] };
if (requested === 'portable-baseline' && JSON.stringify(output.before) !== JSON.stringify(before)) throw new Error('baseline inputs changed');
output.environment = { PATH: env.PATH, GOPROXY: 'off', GOSUMDB: 'off', GOTOOLCHAIN: 'local', GOWORK: 'off', GOENV: 'off', GOFLAGS: '', GOMAXPROCS: '2', host: 'linux/arm64 stock Go 1.27.1; no native qualification' };
for (const [label, cwd, command] of commands) {
  const started_at = new Date().toISOString(), start = performance.now();
  const result = spawnSync(command[0], command.slice(1), { cwd: resolve(root, cwd), env, encoding: 'utf8', maxBuffer: 64 * 1024 * 1024, timeout: 600000 });
  const events = (result.stdout || '').split('\n').flatMap(line => { try { const e = JSON.parse(line); return e.Action && e.Test ? [e] : []; } catch { return []; } });
  const terminal = events.filter(e => ['pass', 'fail', 'skip'].includes(e.Action)).map(e => [e.Package, e.Test, e.Action]);
  const run = { label, cwd: resolve(root, cwd), command, started_at, finished_at: new Date().toISOString(), elapsed_seconds: (performance.now() - start) / 1000, exit_code: result.status, signal: result.signal, error: result.error?.message, terminal, stdout: result.stdout, stderr: result.stderr };
  compact(phase, run);
  output.runs.push(run);
  writeFileSync(resolve(artifacts, phase + '.json'), JSON.stringify(output, null, 2) + '\n');
  console.log(JSON.stringify({ phase, label, exit_code: run.exit_code, elapsed_seconds: run.elapsed_seconds, counts: run.test_counts, skips: run.failures_and_skips.filter(e => e.action === 'skip') }));
}
output.after = identities();
output.unchanged_inputs = JSON.stringify(before) === JSON.stringify(output.after);
writeFileSync(resolve(artifacts, phase + '.json'), JSON.stringify(output, null, 2) + '\n');
if (!output.unchanged_inputs || output.runs.some(r => r.exit_code !== 0 && !['lint', 'full-lint', ...(phase === 'baseline' ? ['packages'] : [])].includes(r.label))) process.exitCode = 1;
