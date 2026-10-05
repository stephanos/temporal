import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync, existsSync, statSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = dirname(fileURLToPath(import.meta.url));
const stock = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
const hash = value => createHash('sha256').update(value).digest('hex');
function sources() {
  const result = spawnSync('git', ['ls-files', '-z'], { cwd: root, maxBuffer: 32 * 1024 * 1024 });
  if (result.status !== 0) throw Error('git source selection failed');
  return Object.fromEntries(result.stdout.toString().split('\0').filter(p => p && !p.startsWith('.flow/') && existsSync(resolve(root, p)) && statSync(resolve(root, p)).isFile()).sort().map(p => [p, hash(readFileSync(resolve(root, p)))]));
}
const env = { ...process.env };
for (const key of ['GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED', 'GOEXPERIMENT']) delete env[key];
Object.assign(env, { GOENV: 'off', GOWORK: 'off', GOTOOLCHAIN: 'local', GOPROXY: 'off', GOSUMDB: 'off', GOFLAGS: '', GOMAXPROCS: '2', GOMAD3_STOCK_GO: stock + '/go', PATH: stock + ':' + env.PATH });
const whitelist = ['GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOSUMDB', 'GOFLAGS', 'GOMAXPROCS', 'GOMAD3_STOCK_GO', 'GOEXPERIMENT', 'GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED', 'GOCACHE', 'GOMODCACHE', 'COMPACT_PHASE', 'COMPACT_CAPTURE_DIR'];
const [label, cwd, ...argv] = process.argv.slice(2);
if (!label || !cwd || !argv.length || !/^[a-z0-9-]+$/.test(label)) throw Error('usage: capture label cwd command arguments');
for (const suffix of ['.json', '.stdout', '.stderr']) if (existsSync(resolve(out, label + suffix))) throw Error('receipt already exists: ' + label);
const before = sources();
const started = new Date();
const result = spawnSync(argv[0], argv.slice(1), { cwd: resolve(root, cwd), env, timeout: 610_000, maxBuffer: 32 * 1024 * 1024 });
const ended = new Date();
const after = sources();
writeFileSync(resolve(out, label + '.stdout'), result.stdout ?? Buffer.alloc(0));
writeFileSync(resolve(out, label + '.stderr'), result.stderr ?? Buffer.alloc(0));
const changes = [...new Set([...Object.keys(before), ...Object.keys(after)])].filter(p => before[p] !== after[p]);
const receipt = { argv, cwd: resolve(root, cwd), environment: Object.fromEntries(whitelist.map(k => [k, env[k] ?? null])), exit: result.status, signal: result.signal, error: result.error?.message ?? null, started: started.toISOString(), ended: ended.toISOString(), elapsed_seconds: (ended - started) / 1000, source_count: Object.keys(before).length, sources_before_sha256: hash(JSON.stringify(before)), sources_after_sha256: hash(JSON.stringify(after)), source_changes: changes, stdout_sha256: hash(result.stdout ?? ''), stderr_sha256: hash(result.stderr ?? ''), helper_sha256: hash(readFileSync(fileURLToPath(import.meta.url))), tool_sha256: { go: hash(readFileSync(stock + '/go')), gofmt: hash(readFileSync(stock + '/gofmt')) } };
writeFileSync(resolve(out, label + '.json'), JSON.stringify(receipt, null, 2) + '\n');
console.log(label, 'exit', result.status, 'seconds', receipt.elapsed_seconds, 'source changes', changes);
console.log(result.stdout?.toString().slice(-1800) ?? '');
console.log(result.stderr?.toString().slice(-1800) ?? '');
process.exitCode = result.status ?? 1;
