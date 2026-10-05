import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const artifacts = dirname(fileURLToPath(import.meta.url));
const go = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
const tools = ['/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0', '/tmp/fn109-lint-tools.ZdNe1t50/errortype'];
const env = { ...process.env, PATH: go + ':/usr/local/bin:/usr/bin:/bin', GOENV: 'off', GOWORK: 'off', GOTOOLCHAIN: 'local', GOPROXY: 'off', GOSUMDB: 'off', GOFLAGS: '', GOMAXPROCS: '2' };
for (const key of ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED']) delete env[key];
const [action, label, cwd, command, ...args] = process.argv.slice(2);
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
const git = args => {
  const result = spawnSync('git', args, { cwd: root, env, encoding: 'utf8' });
  if (result.status !== 0) throw new Error(result.stderr);
  return result.stdout;
};
if (action === 'manifest') {
  const paths = git(['ls-files', '-z', '--', 'tools/gomad3', 'go.mod', 'go.sum', '.github', 'Makefile', '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-42']).split('\0').filter(Boolean).sort();
  const files = Object.fromEntries(paths.map(path => [path, hash(readFileSync(resolve(root, path)))]));
  const binaries = Object.fromEntries([...tools, go + '/go', '/usr/bin/node'].map(path => [path, hash(readFileSync(path))]));
  writeFileSync(resolve(artifacts, label + '.json'), JSON.stringify({ captured_at: new Date().toISOString(), head: git(['rev-parse', 'HEAD']).trim(), status: git(['status', '--short']), files, binaries, environment: { ...Object.fromEntries(['PATH', 'GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOPROXY', 'GOSUMDB', 'GOFLAGS', 'GOMAXPROCS'].map(key => [key, env[key]])), unset: ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED'] } }, null, 2) + '\n');
  console.log(JSON.stringify({ manifest: label, files: paths.length }));
} else if (action === 'run') {
  const started_at = new Date().toISOString();
  const start = performance.now();
  const result = spawnSync(command, args, { cwd: resolve(root, cwd), env, encoding: 'utf8', maxBuffer: 64 * 1024 * 1024 });
  const elapsed_seconds = (performance.now() - start) / 1000;
  const output = { command: [command, ...args], cwd: resolve(root, cwd), started_at, finished_at: new Date().toISOString(), elapsed_seconds, exit_code: result.status, signal: result.signal, error: result.error?.message, stdout: result.stdout, stderr: result.stderr };
  const events = (result.stdout || '').split('\n').flatMap(line => { try { const event = JSON.parse(line); return event && typeof event === 'object' && event.Action ? [event] : []; } catch { return []; } });
  output.test_counts = Object.fromEntries(['pass', 'fail', 'skip'].map(action => [action, events.filter(event => event.Action === action && event.Test).length]));
  output.skips = events.filter(event => event.Action === 'skip').map(event => ({ package: event.Package, test: event.Test }));
  writeFileSync(resolve(artifacts, label + '.json'), JSON.stringify(output, null, 2) + '\n');
  console.log(JSON.stringify({ label, exit_code: result.status, elapsed_seconds, test_counts: output.test_counts, skips: output.skips }));
  if (result.status !== 0) {
    if (events.length) console.log(events.filter(event => event.OutputType === 'error').map(event => event.Test + ': ' + event.Output.trim()).join('\n'));
    else console.log((result.stdout || '') + (result.stderr || ''));
  }
} else throw new Error('expected manifest or run');
