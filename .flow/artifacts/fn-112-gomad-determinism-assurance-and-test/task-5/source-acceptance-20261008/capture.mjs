import {createHash} from 'node:crypto';
import {readFileSync, writeFileSync, existsSync, statSync} from 'node:fs';
import {spawnSync} from 'node:child_process';
import {dirname, resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import assert from 'node:assert/strict';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = dirname(fileURLToPath(import.meta.url));
const stock = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
const hash = value => createHash('sha256').update(value).digest('hex');
function sources() {
  const result = spawnSync('git', ['ls-files', '-z'], {cwd: root, maxBuffer: 32 << 20});
  assert.equal(result.status, 0);
  return Object.fromEntries(result.stdout.toString().split('\0').filter(p => p && !p.startsWith('.flow/') && existsSync(resolve(root, p)) && statSync(resolve(root, p)).isFile()).sort().map(p => [p, hash(readFileSync(resolve(root, p)))]));
}
const env = {...process.env};
for (const key of ['GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED']) delete env[key];
Object.assign(env, {GOENV: 'off', GOWORK: 'off', GOTOOLCHAIN: 'local', GOFLAGS: '', GOEXPERIMENT: 'nogreenteagc', GOMAXPROCS: '2', GOMAD3_STOCK_GO: stock + '/go', PATH: stock + ':' + env.PATH});
const [label, ...argv] = process.argv.slice(2);
assert(label && argv.length && /^[a-z0-9-]+$/.test(label));
for (const suffix of ['.json', '.stdout', '.stderr']) assert(!existsSync(resolve(out, label + suffix)), 'receipt exists: ' + label);
const before = sources(), started = new Date();
const result = spawnSync(argv[0], argv.slice(1), {cwd: root, env, timeout: 610000, maxBuffer: 32 << 20});
const ended = new Date(), after = sources();
writeFileSync(resolve(out, label + '.stdout'), result.stdout ?? '');
writeFileSync(resolve(out, label + '.stderr'), result.stderr ?? '');
const events = (result.stdout?.toString() ?? '').split('\n').filter(line => line.startsWith('{')).map(line => JSON.parse(line));
const top = events.filter(event => event.Test && !event.Test.includes('/') && ['pass', 'fail', 'skip'].includes(event.Action));
const receipt = {argv, cwd: root, environment: Object.fromEntries(['GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOFLAGS', 'GOEXPERIMENT', 'GOMAXPROCS', 'GOMAD3_STOCK_GO', 'PATH'].map(k => [k, env[k]])), exit: result.status, signal: result.signal, error: result.error?.message ?? null, started: started.toISOString(), ended: ended.toISOString(), elapsed_seconds: (ended - started) / 1000, source_count: Object.keys(before).length, sources_before_sha256: hash(JSON.stringify(before)), sources_after_sha256: hash(JSON.stringify(after)), source_changes: [...new Set([...Object.keys(before), ...Object.keys(after)])].filter(p => before[p] !== after[p]), stdout_sha256: hash(result.stdout ?? ''), stderr_sha256: hash(result.stderr ?? ''), tool_sha256: {go: hash(readFileSync(stock + '/go')), gofmt: hash(readFileSync(stock + '/gofmt'))}, tests: top.map(event => ({package: event.Package, test: event.Test, action: event.Action})), test_counts: Object.fromEntries(['pass', 'fail', 'skip'].map(action => [action, top.filter(event => event.Action === action).length]))};
writeFileSync(resolve(out, label + '.json'), JSON.stringify(receipt, null, 2) + '\n');
console.log(JSON.stringify({label, exit: result.status, elapsed_seconds: receipt.elapsed_seconds, source_changes: receipt.source_changes, test_counts: receipt.test_counts}));
console.log(result.stdout?.toString().slice(-1800) ?? '');
console.log(result.stderr?.toString().slice(-1800) ?? '');
process.exitCode = result.status ?? 1;
