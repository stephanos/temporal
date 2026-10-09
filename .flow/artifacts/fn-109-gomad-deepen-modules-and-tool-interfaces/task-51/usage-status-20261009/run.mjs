import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

export const repo = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
export const out = path.dirname(new URL(import.meta.url).pathname);
export const base = 'e09187751326abf393011052dd08fdfc9af61900';
export const original = '951c5516e9e7b3066e7e069adda9565cfd68844c';
export const go = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
export const lint = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0';
export const errortype = '/tmp/fn109-lint-tools.ZdNe1t50/errortype';
export const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
export function git(...args) {
  const result = spawnSync('git', args, {cwd: repo, encoding: 'utf8', maxBuffer: 64 * 1024 * 1024});
  if (result.status !== 0) throw Error(result.stderr);
  return result.stdout;
}
const predecessor = path.resolve(out, '../../task-50/generator-stderr-20261009');
const sourceBasis = path.join(predecessor, 'source-d11d5b9700d75b6cbd0d02d1d44bbaedd832fd1e34f2f02bc534af0abf31204c.json');
const toolsBasis = path.join(predecessor, 'tools-8f2aef1b9b27c90165396c6a2f15e92947df84bf05f853aec877a2ffce3a266a.json');
const basis = JSON.parse(fs.readFileSync(sourceBasis));
const added = 'tools/gomad3/cmd/gomadtool/usage_status_preservation_test.go';
export const sources = () => [...basis.map(entry => ({path: entry.path, sha256: hash(fs.readFileSync(path.join(repo, entry.path)))})),
  ...(fs.existsSync(path.join(repo, added)) ? [{path: added, sha256: hash(fs.readFileSync(path.join(repo, added)))}] : [])].sort((a, b) => a.path.localeCompare(b.path));
const tools = () => JSON.parse(fs.readFileSync(toolsBasis)).map(entry => ({path: entry.path, sha256: hash(fs.readFileSync(entry.path))}));
const controls = () => [path.join(out, 'run.mjs'), path.join(out, 'admission.md'), ...fs.readdirSync(out).filter(name => name.endsWith('.mjs') && name !== 'run.mjs').map(name => path.join(out, name)),
  ...['AGENTS.md', 'MILESTONES.md', '.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.51.md', '.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md'].map(name => path.join(repo, name))]
  .sort().map(file => ({path: file, sha256: hash(fs.readFileSync(file))}));
const env = {...process.env,
  PATH: path.dirname(go) + ':' + process.env.PATH, GOENV: 'off', GOWORK: 'off', GOTOOLCHAIN: 'local', GOFLAGS: '',
  GOCACHE: '/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r',
  TMPDIR: '/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX',
  GOTMPDIR: '/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX',
  GOMODCACHE: '/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc',
  GOLANGCI_LINT_CACHE: '/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/.lint-cache',
  GOPROXY: 'file:///Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/cache/download', GOSUMDB: 'off',
  TEST_TELEMETRY_DIR: '/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/task51-usage-telemetry'};
for (const key of ['BASH_ENV', 'GOROOT', 'GOBIN', 'GOMADSEED', 'GOMAD3_CHILD_SEED']) delete env[key];
export function run(name, command, overrides = {}) {
  if (!/^[a-z0-9-]+$/.test(name)) throw Error('invalid receipt name');
  const receiptPath = path.join(out, name + '-receipt.json');
  if (fs.existsSync(receiptPath)) throw Error('receipt already exists');
  const before = sources(), toolInputs = tools(), controlInputs = controls();
  const source = {basis: sourceBasis, basis_sha256: hash(fs.readFileSync(sourceBasis)),
    inventory_sha256: hash(JSON.stringify(before)), covered_paths: before.length,
    overrides: before.filter(entry => basis.find(old => old.path === entry.path)?.sha256 !== entry.sha256)};
  const tool = {basis: toolsBasis, basis_sha256: hash(fs.readFileSync(toolsBasis)), inventory_sha256: hash(JSON.stringify(toolInputs)), entries: toolInputs};
  if (JSON.stringify(toolInputs) !== JSON.stringify(JSON.parse(fs.readFileSync(toolsBasis)))) throw Error('predecessor tool/dependency inputs changed');
  const stdout = path.join(out, name + '.stdout'), stderr = path.join(out, name + '.stderr');
  const handles = [fs.openSync(stdout, 'wx'), fs.openSync(stderr, 'wx')];
  const childEnv = {...env, ...overrides};
  const argv = ['timeout', '600', 'bash', '-c', command];
  const started = new Date().toISOString(), start = performance.now();
  const result = spawnSync(argv[0], argv.slice(1), {cwd: repo, env: childEnv, stdio: ['ignore', ...handles]});
  handles.forEach(fd => fs.closeSync(fd));
  const events = fs.readFileSync(stdout, 'utf8').split('\n').filter(line => line.startsWith('{')).flatMap(line => {try {return [JSON.parse(line)];} catch {return [];}});
  const terminal = events.filter(event => event.Test && ['pass', 'fail', 'skip'].includes(event.Action));
  const receipt = {name, command, argv, cwd: repo, started, ended: new Date().toISOString(), elapsed_seconds: (performance.now() - start) / 1000,
    child_exit_code: result.status, signal: result.signal, error: result.error?.message ?? null, base, original_baseline: original,
    head: git('rev-parse', 'HEAD').trim(), source, tools: tool, controls: controlInputs,
    source_unchanged: JSON.stringify(before) === JSON.stringify(sources()), tools_unchanged: JSON.stringify(toolInputs) === JSON.stringify(tools()),
    controls_unchanged: JSON.stringify(controlInputs) === JSON.stringify(controls()),
    stdout: {path: path.basename(stdout), sha256: hash(fs.readFileSync(stdout))}, stderr: {path: path.basename(stderr), sha256: hash(fs.readFileSync(stderr))},
    environment: Object.fromEntries(Object.entries(childEnv).filter(([key]) => ['PATH', 'GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOFLAGS', 'GOCACHE', 'GOMODCACHE', 'GOLANGCI_LINT_CACHE', 'TMPDIR', 'GOTMPDIR', 'GOPROXY', 'GOSUMDB', 'CGO_ENABLED', 'GOOS', 'GOARCH', 'TEST_TELEMETRY_DIR', 'GIT_TRACE2_EVENT'].includes(key))),
    environment_sha256: hash(JSON.stringify(Object.entries(childEnv).sort(([a], [b]) => a.localeCompare(b)))),
    counts: Object.fromEntries(['pass', 'fail', 'skip'].map(action => [action, terminal.filter(event => event.Action === action).length])),
    top_level_counts: Object.fromEntries(['pass', 'fail', 'skip'].map(action => [action, terminal.filter(event => event.Action === action && !event.Test.includes('/')).length])),
    observations: terminal.map(event => ({Test: event.Test, Action: event.Action, Package: event.Package}))};
  if (overrides.GIT_TRACE2_EVENT && fs.existsSync(overrides.GIT_TRACE2_EVENT)) receipt.git_trace = {path: overrides.GIT_TRACE2_EVENT, sha256: hash(fs.readFileSync(overrides.GIT_TRACE2_EVENT))};
  fs.writeFileSync(receiptPath, JSON.stringify(receipt, null, 2) + '\n', {flag: 'wx'});
  console.log(JSON.stringify({name, child_exit_code: result.status, elapsed_seconds: receipt.elapsed_seconds, source, counts: receipt.counts, top_level_counts: receipt.top_level_counts}));
  if (!Number.isInteger(result.status) || result.signal || result.error || !receipt.source_unchanged || !receipt.tools_unchanged || !receipt.controls_unchanged) throw Error('inconclusive terminal/binding observation');
  return receipt;
}
if (process.argv[1] === new URL(import.meta.url).pathname) run(process.argv[2], process.argv[3], JSON.parse(process.argv[4] || '{}'));
