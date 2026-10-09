import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';

export const repo = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
export const out = path.dirname(new URL(import.meta.url).pathname);
export const base = '96b3974997fed4234e62786bf5717dcb863ea46e';
export const original = '951c5516e9e7b3066e7e069adda9565cfd68844c';
export const go = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go';
export const lint = '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0';
export const errortype = '/tmp/fn109-lint-tools.ZdNe1t50/errortype';
export const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
export const git = (...args) => {
  const result = spawnSync('git', args, {cwd: repo, encoding: 'utf8', maxBuffer: 64 * 1024 * 1024});
  if (result.status !== 0) throw Error(result.stderr);
  return result.stdout;
};
const env = {
  ...process.env, PATH: path.dirname(go) + ':' + process.env.PATH,
  GOENV: 'off', GOWORK: 'off', GOTOOLCHAIN: 'local', GOFLAGS: '',
  GOCACHE: '/Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r',
  TMPDIR: '/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX',
  GOTMPDIR: '/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX',
  GOMODCACHE: '/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc',
  GOLANGCI_LINT_CACHE: '/Users/stephan/Workspace/skunkworks/.gomad-fn1132-module-cache-g4UAforc/.lint-cache',
  GOPROXY: 'off', GOSUMDB: 'off',
  TEST_TELEMETRY_DIR: '/Users/stephan/Workspace/skunkworks/.gomad-source-gates-UsyTMX/task50-generator-telemetry',
};
for (const key of ['BASH_ENV', 'GOROOT', 'GOBIN', 'GOMADSEED', 'GOMAD3_CHILD_SEED']) delete env[key];
const baselinePaths = git('ls-tree', '-r', '--name-only', base, '--',
  'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'tests/gomadfunctional',
  'go.mod', 'go.sum', '.github/.golangci.yml', 'Makefile', 'cmd/tools/lintcode')
  .trim().split('\n').filter(Boolean);
const fixture = 'tools/gomad3/cmd/gomadtool/generator_diagnostic_output_test.go';
const sources = () => [...new Set([...baselinePaths, ...(fs.existsSync(path.join(repo, fixture)) ? [fixture] : [])])]
  .sort().map(relative => ({path: relative, sha256: fs.existsSync(path.join(repo, relative)) ? hash(fs.readFileSync(path.join(repo, relative))) : null}));
const tools = () => [go, path.join(path.dirname(go), 'gofmt'),
  ...['compile', 'link', 'asm', 'vet'].map(name => path.join(path.dirname(path.dirname(go)), 'pkg/tool/linux_arm64', name)),
  lint, errortype, process.execPath, '/usr/bin/bash', '/usr/bin/timeout', '/usr/bin/git', '/usr/bin/tar', '/usr/bin/make', '/usr/bin/patch']
  .map(file => ({path: file, sha256: hash(fs.readFileSync(file))}));
const controlFiles = () => [path.join(out, 'run.mjs'), path.join(out, 'admission.md'),
  path.join(repo, 'AGENTS.md'), path.join(repo, 'MILESTONES.md'),
  path.join(repo, '.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md'),
  path.join(repo, '.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.50.md'),
  path.join(repo, '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/source-acceptance-20261008/format-gate.sh'),
  ...['baseline.mjs', 'gates.mjs', 'successor.mjs', 'proof.mjs', 'inventory.mjs', 'prepare-overlay.mjs', 'baseline-overlay.json',
    'baseline-qualification_manifest.go', 'baseline-protocol.go', 'baseline-version.go', 'baseline-boundary.go'].map(name => path.join(out, name))]
  .filter(file => fs.existsSync(file)).sort().map(file => ({path: file, sha256: hash(fs.readFileSync(file))}));
const save = (kind, entries) => {
  const bytes = JSON.stringify(entries, null, 2) + '\n';
  const digest = hash(bytes), name = kind + '-' + digest + '.json', file = path.join(out, name);
  if (!fs.existsSync(file)) fs.writeFileSync(file, bytes, {flag: 'wx'});
  for (const entry of entries.filter(entry => /\.(mjs|md|sh|json|go)$/.test(entry.path))) {
    if (kind !== 'controls') break;
    const archived = path.join(out, 'control-' + entry.sha256 + path.extname(entry.path));
    if (!fs.existsSync(archived)) fs.writeFileSync(archived, fs.readFileSync(entry.path), {flag: 'wx'});
  }
  return {manifest: name, sha256: digest};
};
export function run(name, command, expected) {
  if (!/^[a-z0-9-]+$/.test(name)) throw Error('invalid receipt name');
  const receiptPath = path.join(out, name + '-receipt.json');
  if (fs.existsSync(receiptPath)) throw Error('receipt already exists: ' + name);
  const before = sources(), toolInputs = tools(), controls = controlFiles();
  const source = save('source', before), tool = save('tools', toolInputs), control = save('controls', controls);
  const stdout = path.join(out, name + '.stdout'), stderr = path.join(out, name + '.stderr');
  const handles = [fs.openSync(stdout, 'wx'), fs.openSync(stderr, 'wx')];
  const argv = ['timeout', '600', 'bash', '-c', command];
  const started = new Date().toISOString(), start = performance.now();
  const result = spawnSync(argv[0], argv.slice(1), {cwd: repo, env, stdio: ['ignore', ...handles]});
  handles.forEach(fd => fs.closeSync(fd));
  const records = fs.readFileSync(stdout, 'utf8').split('\n').filter(line => line.startsWith('{'))
    .flatMap(line => {try {return [JSON.parse(line)];} catch {return [];}});
  const receipt = {
    name, command, argv, cwd: repo, started, ended: new Date().toISOString(),
    elapsed_seconds: (performance.now() - start) / 1000, child_exit_code: result.status,
    signal: result.signal, error: result.error?.message ?? null,
    terminal_handle: 'foreground spawnSync returned a numeric child status',
    base, original_baseline: original, head: git('rev-parse', 'HEAD').trim(), source, tools: tool, controls: control,
    source_unchanged: JSON.stringify(before) === JSON.stringify(sources()),
    tools_unchanged: JSON.stringify(toolInputs) === JSON.stringify(tools()),
    controls_unchanged: JSON.stringify(controls) === JSON.stringify(controlFiles()),
    stdout: {path: path.basename(stdout), sha256: hash(fs.readFileSync(stdout))},
    stderr: {path: path.basename(stderr), sha256: hash(fs.readFileSync(stderr))},
    environment: Object.fromEntries(Object.entries(env).filter(([key]) => ['PATH', 'GOENV', 'GOWORK', 'GOTOOLCHAIN', 'GOFLAGS', 'GOCACHE', 'GOMODCACHE', 'GOLANGCI_LINT_CACHE', 'TMPDIR', 'GOTMPDIR', 'GOPROXY', 'GOSUMDB', 'CGO_ENABLED', 'GOOS', 'GOARCH', 'TEST_TELEMETRY_DIR'].includes(key))),
    environment_sha256: hash(JSON.stringify(Object.entries(env).sort(([a], [b]) => a.localeCompare(b)))),
    counts: Object.fromEntries(['pass', 'fail', 'skip'].map(action => [action, records.filter(record => record.Action === action && record.Test).length])),
    top_level_counts: Object.fromEntries(['pass', 'fail', 'skip'].map(action => [action, records.filter(record => record.Action === action && record.Test && !record.Test.includes('/')).length])),
    observations: records.filter(record => ['pass', 'fail', 'skip'].includes(record.Action) && record.Test),
  };
  fs.writeFileSync(receiptPath, JSON.stringify(receipt, null, 2) + '\n', {flag: 'wx'});
  console.log(JSON.stringify({name, child_exit_code: result.status, elapsed_seconds: receipt.elapsed_seconds, source, counts: receipt.counts, top_level_counts: receipt.top_level_counts}));
  if (!Number.isInteger(result.status) || result.signal || result.error || !receipt.source_unchanged || !receipt.tools_unchanged || !receipt.controls_unchanged) throw Error('nonterminal or changed bindings: ' + name);
  if (result.status !== expected) throw Error('unexpected child exit: ' + name + ': ' + result.status + ', expected ' + expected);
  return receipt;
}
