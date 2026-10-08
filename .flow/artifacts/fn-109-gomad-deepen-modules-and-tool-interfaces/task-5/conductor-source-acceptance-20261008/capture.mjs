import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync, existsSync, statSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
export const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
export const out = dirname(fileURLToPath(import.meta.url));
export const worker = resolve(out, '../source-acceptance-20261008');
export const stock = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
export const read = p => readFileSync(resolve(root, p));
export const sha = b => createHash('sha256').update(b).digest('hex');
export function sources() {
  const listed = spawnSync('git', ['ls-files', '-z'], { cwd: root, maxBuffer: 64 << 20 });
  assert.equal(listed.status, 0);
  const paths = new Set(listed.stdout.toString().split('\0').filter(Boolean));
  const regression = 'tools/gomad3/cmd/gomad/internal/cli/explore_failure_writer_test.go';
  if (existsSync(resolve(root, regression))) paths.add(regression);
  return Object.fromEntries([...paths].filter(p => !p.startsWith('.flow/') && existsSync(resolve(root, p)) && statSync(resolve(root, p)).isFile()).sort().map(p => [p, sha(read(p))]));
}
if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const [label, ...argv] = process.argv.slice(2);
  assert(label && argv.length && /^[a-z0-9-]+$/.test(label));
  for (const suffix of ['.json', '.stdout', '.stderr']) assert(!existsSync(resolve(out, label + suffix)), 'Receipt exists: ' + label);
  const env = { ...process.env }, removed = ['GOROOT', 'GOBIN', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED'];
  for (const k of Object.keys(env)) if (k.startsWith('GOMAD') && /CAPTURE|SEED/.test(k) && !removed.includes(k)) removed.push(k);
  for (const k of removed) delete env[k];
  const temp = '/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX';
  assert(statSync(temp).isDirectory());
  Object.assign(env, { GOENV: 'off', GOWORK: 'off', GOTOOLCHAIN: 'local', GOFLAGS: '', GOEXPERIMENT: 'nogreenteagc', GOMAXPROCS: '2', GOMAD3_STOCK_GO: stock + '/go', PATH: stock + ':' + env.PATH, TMPDIR: temp, GOTMPDIR: temp });
  const before = sources(), started = new Date(), mono = process.hrtime.bigint();
  const result = spawnSync(argv[0], argv.slice(1), { cwd: root, env, timeout: 600000, maxBuffer: 128 << 20 });
  const elapsed = process.hrtime.bigint() - mono, ended = new Date(), after = sources();
  const events = (result.stdout?.toString() ?? '').split('\n').flatMap(l => { try { return [JSON.parse(l)]; } catch { return []; } });
  const terminal = events.filter(e => e.Test && ['pass', 'fail', 'skip'].includes(e.Action)), top = terminal.filter(e => !e.Test.includes('/'));
  const receipt = {
    argv, cwd: root, environment: Object.fromEntries(['GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOFLAGS', 'GOEXPERIMENT', 'GOMAXPROCS', 'GOMAD3_STOCK_GO', 'PATH', 'TMPDIR', 'GOTMPDIR', 'GOCACHE'].map(k => [k, env[k] ?? null])), removed_environment: removed,
    exit: result.status, signal: result.signal, error: result.error?.message ?? null, started: started.toISOString(), ended: ended.toISOString(), monotonic_elapsed_nanos: elapsed.toString(), elapsed_seconds: Number(elapsed) / 1e9,
    sources_before_sha256: sha(JSON.stringify(before)), sources_after_sha256: sha(JSON.stringify(after)), source_changes: [...new Set([...Object.keys(before), ...Object.keys(after)])].filter(p => before[p] !== after[p]),
    stdout_sha256: sha(result.stdout ?? ''), stderr_sha256: sha(result.stderr ?? ''),
    tool_sha256: { go: sha(read(stock + '/go')), gofmt: sha(read(stock + '/gofmt')), golangci_lint: sha(read('/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0')), errortype: sha(read('/tmp/fn109-lint-tools.ZdNe1t50/errortype')) },
    tests: terminal.map(e => ({ package: e.Package, test: e.Test, action: e.Action })), top_level_counts: Object.fromEntries(['pass', 'fail', 'skip'].map(a => [a, top.filter(e => e.Action === a).length])),
    package_results: events.filter(e => !e.Test && ['pass', 'fail', 'skip'].includes(e.Action)).map(e => ({ package: e.Package, action: e.Action })), native: false,
  };
  for (const [suffix, value] of [['.stdout', result.stdout ?? ''], ['.stderr', result.stderr ?? ''], ['.json', JSON.stringify(receipt, null, 2) + '\n']]) writeFileSync(resolve(out, label + suffix), value, { flag: 'wx' });
  console.log(JSON.stringify({ label, exit: receipt.exit, elapsed_seconds: receipt.elapsed_seconds, source_changes: receipt.source_changes, counts: receipt.top_level_counts }));
  console.log(result.stdout?.toString().slice(-1200) ?? '');
  console.log(result.stderr?.toString().slice(-1200) ?? '');
  process.exitCode = result.status ?? 1;
}
