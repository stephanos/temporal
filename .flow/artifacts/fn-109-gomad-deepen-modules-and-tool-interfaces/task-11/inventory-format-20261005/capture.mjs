import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { closeSync, mkdirSync, openSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const artifacts = dirname(fileURLToPath(import.meta.url));
const toolchain = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
const env = { ...process.env, PATH: toolchain + ':/usr/local/bin:/usr/bin:/bin', GOENV: 'off', GOWORK: 'off', GOTOOLCHAIN: 'local', GOPROXY: 'off', GOSUMDB: 'off', GOFLAGS: '', GOMAXPROCS: '2' };
const removed = ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED'];
for (const name of removed) delete env[name];
const hash = path => createHash('sha256').update(readFileSync(path)).digest('hex');
const git = args => {
  const result = spawnSync('git', args, { cwd: root, env, encoding: 'utf8' });
  if (result.status !== 0) throw new Error(result.stderr);
  return result.stdout;
};
const selectedFiles = ['tools/gomad3/internal/sourceinventory/inventory.go', 'tools/gomad3/internal/sourceinventory/inventory_test.go', 'tools/gomad3/go.mod', 'tools/gomad3/go.sum', '.github/.golangci.yml', 'Makefile', 'tools/gomad3/Makefile', 'tools/gomad3/toolchain/version/version.json', 'tools/gomad3/internal/compatibilitypack/generation.json'];
const binaries = [toolchain + '/go', toolchain + '/gofmt', '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0', '/tmp/fn109-lint-tools.ZdNe1t50/errortype', '/usr/bin/node'];
const source = () => {
  const paths = git(['ls-files', '-z', '--', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'Makefile', '.github/.golangci.yml']).split('\0').filter(Boolean).sort();
  const digest = createHash('sha256');
  for (const path of paths) digest.update(path + '\0' + hash(resolve(root, path)) + '\0');
  return { sha256: digest.digest('hex'), tracked_path_count: paths.length };
};
const identities = () => ({ source: source(), files: Object.fromEntries(selectedFiles.map(path => [path, hash(resolve(root, path))])), tools: Object.fromEntries(binaries.map(path => [path, hash(path)])) });
const [mode, ...argv] = process.argv.slice(2);
const freezePath = resolve(artifacts, 'source-freeze.json');
if (mode === 'freeze') {
  const [stage] = argv;
  if (!['BASE', 'FINAL'].includes(stage)) throw new Error('freeze stage must be BASE or FINAL');
  const freeze = stage === 'BASE' ? {} : JSON.parse(readFileSync(freezePath));
  if (freeze[stage]) throw new Error('refusing to overwrite frozen stage');
  freeze[stage] = { head: git(['rev-parse', 'HEAD']).trim(), branch: git(['branch', '--show-current']).trim(), recorded_at: new Date().toISOString(), host: { platform: process.platform, arch: process.arch, native_qualification: false }, environment: Object.fromEntries(['PATH', 'GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOSUMDB', 'GOFLAGS', 'GOMAXPROCS'].map(name => [name, env[name]])), removed_environment: removed, ...identities() };
  writeFileSync(freezePath, JSON.stringify(freeze, null, 2) + '\n', { flag: stage === 'BASE' ? 'wx' : 'w' });
  console.log(JSON.stringify({ stage, ...freeze[stage] }));
} else if (mode === 'run') {
  const [stage, label, cwd, command, ...args] = argv;
  if (!/^[a-z0-9-]+$/.test(label)) throw new Error('invalid capture label');
  const freeze = JSON.parse(readFileSync(freezePath))[stage];
  if (!freeze) throw new Error('missing frozen stage');
  const before = identities();
  if (JSON.stringify(before) !== JSON.stringify({ source: freeze.source, files: freeze.files, tools: freeze.tools })) throw new Error('source/tool inputs differ from frozen stage');
  const outputDir = resolve(artifacts, label);
  mkdirSync(outputDir);
  const stdout = openSync(resolve(outputDir, 'stdout.log'), 'wx');
  const stderr = openSync(resolve(outputDir, 'stderr.log'), 'wx');
  const startedAt = new Date().toISOString();
  const start = performance.now();
  const result = spawnSync(command, args, { cwd: resolve(root, cwd), env, stdio: ['ignore', stdout, stderr] });
  const elapsedSeconds = (performance.now() - start) / 1000;
  closeSync(stdout);
  closeSync(stderr);
  const after = identities();
  const events = readFileSync(resolve(outputDir, 'stdout.log'), 'utf8').split('\n').flatMap(line => { try { const event = JSON.parse(line); return event?.Action ? [event] : []; } catch { return []; } });
  const counts = Object.fromEntries(['run', 'pass', 'fail', 'skip'].map(action => [action, events.filter(event => event.Action === action && event.Test).length]));
  const topLevelTests = events.filter(event => event.Action === 'run' && event.Test && !event.Test.includes('/')).map(event => event.Test);
  const receipt = { stage, freeze: '../source-freeze.json#' + stage, head: git(['rev-parse', 'HEAD']).trim(), command: [command, ...args], cwd: resolve(root, cwd), started_at: startedAt, finished_at: new Date().toISOString(), elapsed_seconds: elapsedSeconds, exit_code: result.status, signal: result.signal, error: result.error?.message, source_before: before.source, source_after: after.source, source_and_tool_inputs_stable: JSON.stringify(before) === JSON.stringify(after), test_counts: counts, top_level_tests: topLevelTests, skips: events.filter(event => event.Action === 'skip' && event.Test).map(event => ({ package: event.Package, test: event.Test })), stdout_sha256: hash(resolve(outputDir, 'stdout.log')), stderr_sha256: hash(resolve(outputDir, 'stderr.log')) };
  writeFileSync(resolve(outputDir, 'receipt.json'), JSON.stringify(receipt, null, 2) + '\n', { flag: 'wx' });
  console.log(JSON.stringify({ label, exit_code: result.status, elapsed_seconds: elapsedSeconds, source_and_tool_inputs_stable: receipt.source_and_tool_inputs_stable, test_counts: counts, top_level_tests: topLevelTests, skips: receipt.skips }));
  process.exitCode = result.status ?? 1;
} else {
  throw new Error('usage: node capture.mjs freeze BASE|FINAL; node capture.mjs run BASE|FINAL label cwd command [args...]');
}
