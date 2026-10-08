import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {existsSync, readFileSync, writeFileSync} from 'node:fs';
import {dirname, resolve} from 'node:path';
import {fileURLToPath} from 'node:url';

export const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
export const out = dirname(fileURLToPath(import.meta.url));
export const stock = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
export const sha = value => createHash('sha256').update(value).digest('hex');
export function git(argv) {
  const result = spawnSync('git', argv, {cwd: root, maxBuffer: 32 << 20});
  assert.equal(result.status, 0, result.stderr.toString());
  return result.stdout;
}
export function sources() {
  const paths = git(['ls-files', '-z', '--cached', '--others', '--exclude-standard', '--', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'tests', 'cmd/tools/lintcode', 'Makefile', 'go.mod', 'go.sum', '.github/.golangci.yml']).toString().split('\0').filter(Boolean).sort();
  return Object.fromEntries(paths.map(path => [path, sha(readFileSync(resolve(root, path)))]));
}
if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const [label, ...argv] = process.argv.slice(2);
  assert(label && /^[a-z0-9-]+$/.test(label) && argv.length);
  for (const suffix of ['.json', '.stdout', '.stderr']) assert(!existsSync(resolve(out, label + suffix)), 'existing receipt ' + label);
  const env = {...process.env};
  for (const key of ['GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED']) delete env[key];
  Object.assign(env, {GOENV: 'off', GOWORK: 'off', GOTOOLCHAIN: 'local', GOFLAGS: '', GOEXPERIMENT: 'nogreenteagc', GOMAXPROCS: '2', GOMAD3_STOCK_GO: stock + '/go', GOLANGCI_LINT_FIX: 'false', PATH: stock + ':' + env.PATH});
  const before = sources(), headBefore = git(['rev-parse', 'HEAD']).toString().trim(), started = new Date();
  const result = spawnSync(argv[0], argv.slice(1), {cwd: root, env, timeout: 600000, maxBuffer: 32 << 20});
  const ended = new Date(), after = sources(), headAfter = git(['rev-parse', 'HEAD']).toString().trim();
  const stdout = result.stdout ?? Buffer.alloc(0), stderr = result.stderr ?? Buffer.alloc(0);
  const events = stdout.toString().split('\n').filter(line => line.startsWith('{')).flatMap(line => { try { const event = JSON.parse(line); return event.Action ? [event] : []; } catch { return []; } });
  const tests = events.filter(e => e.Test && !e.Test.includes('/') && ['pass', 'fail', 'skip'].includes(e.Action));
  const receipt = {argv, cwd: root, environment: Object.fromEntries(['GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOFLAGS', 'GOEXPERIMENT', 'GOMAXPROCS', 'GOMAD3_STOCK_GO', 'GOLANGCI_LINT_FIX', 'PATH'].map(key => [key, env[key]])), unset_environment: ['GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED'], exit: result.status, signal: result.signal, error: result.error?.message ?? null, started: started.toISOString(), ended: ended.toISOString(), elapsed_seconds: (ended - started) / 1000, head_before: headBefore, head_after: headAfter, source_count: Object.keys(before).length, source_before_sha256: sha(JSON.stringify(before)), source_after_sha256: sha(JSON.stringify(after)), source_changes: [...new Set([...Object.keys(before), ...Object.keys(after)])].filter(p => before[p] !== after[p]), stdout_sha256: sha(stdout), stderr_sha256: sha(stderr), go_sha256: sha(readFileSync(stock + '/go')), tests: tests.map(e => ({package: e.Package, test: e.Test, action: e.Action})), test_counts: Object.fromEntries(['pass', 'fail', 'skip'].map(action => [action, tests.filter(e => e.Action === action).length])), package_results: events.filter(e => !e.Test && ['pass', 'fail', 'skip'].includes(e.Action)).map(e => ({package: e.Package, action: e.Action, elapsed: e.Elapsed}))};
  writeFileSync(resolve(out, label + '.stdout'), stdout);
  writeFileSync(resolve(out, label + '.stderr'), stderr);
  writeFileSync(resolve(out, label + '.json'), JSON.stringify(receipt, null, 2) + '\n');
  console.log(JSON.stringify({label, exit: receipt.exit, error: receipt.error, elapsed_seconds: receipt.elapsed_seconds, source_changes: receipt.source_changes, test_counts: receipt.test_counts}));
  console.log(stdout.toString().slice(-1800));
  console.log(stderr.toString().slice(-1800));
  process.exitCode = result.status ?? 1;
}
