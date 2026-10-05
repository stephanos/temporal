import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { closeSync, existsSync, openSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const artifacts = dirname(fileURLToPath(import.meta.url));
const toolchain = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
const env = { ...process.env, PATH: toolchain + ':/usr/local/bin:/usr/bin:/bin', GOENV: 'off', GOWORK: 'off', GOTOOLCHAIN: 'local', GOPROXY: 'off', GOSUMDB: 'off', GOFLAGS: '', GOMAXPROCS: '2' };
const removed = ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED'];
for (const name of removed) delete env[name];
const hashBytes = bytes => createHash('sha256').update(bytes).digest('hex');
const hash = path => hashBytes(readFileSync(path));
const git = args => {
  const result = spawnSync('git', args, { cwd: root, env, encoding: 'utf8', maxBuffer: 32 << 20 });
  if (result.status !== 0) throw new Error(result.error?.message || result.stderr);
  return result.stdout;
};
const selectedFiles = ['tools/gomad3/internal/compatibilitypack/authoring/generate.go', 'tools/gomad3/internal/compatibilitypack/authoring/review.go', 'tools/gomad3/cmd/gomadtool/compatibility_pack_refresh.go', 'tools/gomad3/cmd/gomadtool/compatibility_pack_refresh_test.go', 'tools/gomad3/upgrade/pinimpact/pinimpact.go', 'tools/gomad3/upgrade/pinimpact/pinimpact_test.go', 'tools/gomad3/go.mod', 'tools/gomad3/go.sum', '.github/.golangci.yml', 'Makefile', 'tools/gomad3/Makefile', 'tools/gomad3/toolchain/version/version.json', 'tools/gomad3/internal/compatibilitypack/generation.json'];
const binaries = [toolchain + '/go', toolchain + '/gofmt', '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0', '/tmp/fn109-lint-tools.ZdNe1t50/errortype', '/usr/bin/node'];
const generated = () => {
  const prefix = 'tools/gomad3/internal/compatibilitypack/';
  return Object.fromEntries(git(['ls-files', '--cached', '--others', '--exclude-standard', '-z', '--', prefix]).split('\0').filter(path => path && (/(requests|packs|reports)\//.test(path) || /testdata\/.*\/(go.mod|go.sum|libc_test.go)$/.test(path) || /(?:generation.json|packs_generated_test.go|working-directories.json)$/.test(path))).sort().map(path => [path, hash(resolve(root, path))]));
};
const identities = () => {
  const paths = git(['ls-files', '--cached', '--others', '--exclude-standard', '-z', '--', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'Makefile', '.github/.golangci.yml']).split('\0').filter(Boolean).sort();
  const digest = createHash('sha256');
  for (const path of paths) digest.update(path + '\0' + hash(resolve(root, path)) + '\0');
  return { source: { sha256: digest.digest('hex'), tracked_path_count: paths.length, scope: 'All tracked tools/gomad3, tools/gomad3sim, tools/gomad3integration, root Makefile and .github/.golangci.yml paths, sorted; each path + NUL + file SHA256 + NUL' }, files: Object.fromEntries(selectedFiles.map(path => [path, hash(resolve(root, path))])), tools: Object.fromEntries(binaries.map(path => [path, hash(path)])) };
};
const receiptPath = resolve(artifacts, 'checks.json');
const receipt = existsSync(receiptPath) ? JSON.parse(readFileSync(receiptPath)) : { environment: Object.fromEntries(['PATH', 'GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOSUMDB', 'GOFLAGS', 'GOMAXPROCS'].map(name => [name, env[name]])), removed_environment: removed, host: { platform: process.platform, arch: process.arch, native_qualification: false, patched_go_present: existsSync(resolve(root, 'tools/gomad3/.toolchain/bin/go')) }, freezes: {}, runs: [] };
const save = () => writeFileSync(receiptPath, JSON.stringify(receipt, null, 2) + '\n');
const [mode, stage, label, cwd, command, ...args] = process.argv.slice(2);
if (!['BASE', 'RED', 'RESTORED', 'GENERATED', 'FINAL', 'FINAL2', 'FINAL3'].includes(stage)) throw new Error('invalid stage');
if (mode === 'freeze') {
  if (receipt.freezes[stage]) throw new Error('refusing to overwrite frozen stage');
  receipt.freezes[stage] = { head: git(['rev-parse', 'HEAD']).trim(), branch: git(['branch', '--show-current']).trim(), recorded_at: new Date().toISOString(), ...identities(), generated: generated() };
  save();
  console.log(JSON.stringify({ stage, source: receipt.freezes[stage].source }));
} else if (mode === 'run') {
  if (!/^[a-z0-9-]+$/.test(label) || receipt.runs.some(run => run.label === label)) throw new Error('invalid or duplicate label');
  const freeze = receipt.freezes[stage];
  const before = identities();
  if (!freeze || JSON.stringify(before) !== JSON.stringify({ source: freeze.source, files: freeze.files, tools: freeze.tools })) throw new Error('source/tool inputs differ from frozen stage');
  const stdoutPath = resolve(artifacts, label + '.stdout.log');
  const stderrPath = resolve(artifacts, label + '.stderr.log');
  const stdout = openSync(stdoutPath, 'wx');
  const stderr = openSync(stderrPath, 'wx');
  const startedAt = new Date().toISOString();
  const start = performance.now();
  const result = spawnSync(command, args, { cwd: resolve(root, cwd), env, stdio: ['ignore', stdout, stderr], timeout: 600000 });
  const elapsedSeconds = (performance.now() - start) / 1000;
  closeSync(stdout);
  closeSync(stderr);
  const after = identities();
  const events = readFileSync(stdoutPath, 'utf8').split('\n').flatMap(line => { try { const event = JSON.parse(line); return event?.Action ? [event] : []; } catch { return []; } });
  const counts = Object.fromEntries(['run', 'pass', 'fail', 'skip'].map(action => [action, events.filter(event => event.Action === action && event.Test).length]));
  const run = { stage, label, intentional_generation: label.startsWith('generate-'), argv: [command, ...args], cwd: resolve(root, cwd), started_at: startedAt, finished_at: new Date().toISOString(), elapsed_seconds: elapsedSeconds, exit_code: result.status, signal: result.signal, error: result.error?.message, source_before: before.source.sha256, source_after: after.source.sha256, source_and_tool_inputs_stable: JSON.stringify(before) === JSON.stringify(after), test_counts: counts, top_level_tests: events.filter(event => event.Action === 'run' && event.Test && !event.Test.includes('/')).map(event => event.Test), stdout_sha256: hash(stdoutPath), stderr_sha256: hash(stderrPath) };
  receipt.runs.push(run);
  save();
  console.log(JSON.stringify(run));
  process.exitCode = result.status ?? 1;
} else {
  throw new Error('usage: freeze BASE|FINAL; run BASE|FINAL label cwd command [args...]');
}
