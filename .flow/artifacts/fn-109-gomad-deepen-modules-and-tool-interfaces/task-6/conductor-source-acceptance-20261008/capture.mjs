import { spawnSync } from 'node:child_process';
import { writeFileSync, existsSync, statSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
import { root, stock, read, sha, sources } from '../../task-5/conductor-source-acceptance-20261008/capture.mjs';
export { root, stock, read, sha, sources };
export const out = dirname(fileURLToPath(import.meta.url));
export const worker = resolve(out, '../source-acceptance-20261008');
if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const [label, ...argv] = process.argv.slice(2);
  assert(label && argv.length && /^[a-z0-9-]+$/.test(label));
  for (const suffix of ['.json', '.stdout', '.stderr']) assert(!existsSync(resolve(out, label + suffix)), 'Existing receipt ' + label);
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
    argv, cwd: root,
    environment: Object.fromEntries(['GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOFLAGS', 'GOEXPERIMENT', 'GOMAXPROCS', 'GOMAD3_STOCK_GO', 'PATH', 'TMPDIR', 'GOTMPDIR', 'GOCACHE'].map(k => [k, env[k] ?? null])), removed_environment: removed,
    exit: result.status, signal: result.signal, error: result.error?.message ?? null, started: started.toISOString(), ended: ended.toISOString(), monotonic_elapsed_nanos: elapsed.toString(), elapsed_seconds: Number(elapsed) / 1e9,
    sources_before_sha256: sha(JSON.stringify(before)), sources_after_sha256: sha(JSON.stringify(after)), source_changes: [...new Set([...Object.keys(before), ...Object.keys(after)])].filter(p => before[p] !== after[p]),
    stdout_sha256: sha(result.stdout ?? ''), stderr_sha256: sha(result.stderr ?? ''),
    tool_sha256: { go: sha(read(stock + '/go')), gofmt: sha(read(stock + '/gofmt')), golangci_lint: sha(read('/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0')), errortype: sha(read('/tmp/fn109-lint-tools.ZdNe1t50/errortype')) },
    tests: terminal.map(e => ({ package: e.Package, test: e.Test, action: e.Action })), top_level_counts: Object.fromEntries(['pass', 'fail', 'skip'].map(a => [a, top.filter(e => e.Action === a).length])),
    package_results: events.filter(e => !e.Test && ['pass', 'fail', 'skip'].includes(e.Action)).map(e => ({ package: e.Package, action: e.Action })), native: false,
  };
  for (const [suffix, value] of [['.stdout', result.stdout ?? ''], ['.stderr', result.stderr ?? ''], ['.json', JSON.stringify(receipt, null, 2) + '\n']]) writeFileSync(resolve(out, label + suffix), value, { flag: 'wx' });
  console.log(JSON.stringify({ label, exit: receipt.exit, source_changes: receipt.source_changes, counts: receipt.top_level_counts, elapsed_seconds: receipt.elapsed_seconds }));
  console.log(result.stdout?.toString().slice(-2400) ?? '');
  console.log(result.stderr?.toString().slice(-1200) ?? '');
  process.exitCode = result.status ?? 1;
}
