const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const cp = require('child_process');

const root = '/Users/stephan/Workspace/skunkworks/gomad/temporal';
const out = path.join(root, '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-26');
const goBin = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin';
const env = { ...process.env, GOENV: 'off', GOFLAGS: '', GOWORK: 'off', GOTOOLCHAIN: 'local', GOMAXPROCS: '2', PATH: goBin + ':/usr/local/bin:/usr/bin:/bin' };
const unset = ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOMAD3_SEED', 'LINT_TEST_BASE_REV'];
for (const name of unset) delete env[name];
const sha = data => crypto.createHash('sha256').update(data).digest('hex');
function snapshot() {
  const files = {};
  function walk(relative) {
    for (const entry of fs.readdirSync(path.join(root, relative), { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
      if (entry.name === '.toolchain' || entry.name === '.bin' || entry.name === '.gomad') continue;
      const name = path.join(relative, entry.name);
      if (entry.isDirectory()) walk(name);
      else if (entry.isFile()) files[name] = sha(fs.readFileSync(path.join(root, name)));
    }
  }
  walk('tools/gomad3');
  files['.github/.golangci.yml'] = sha(fs.readFileSync(path.join(root, '.github/.golangci.yml')));
  return files;
}
const [mode, freeze, name, command] = process.argv.slice(2);
const freezePath = path.join(out, freeze + '.json');
function readFreeze(file) {
  const data = JSON.parse(fs.readFileSync(file));
  if (!data.base) return data;
  const base = readFreeze(path.join(out, data.base));
  for (const name of data.removed) delete base[name];
  return Object.assign(base, data.changed);
}
if (mode === 'freeze') {
  const current = snapshot();
  const basePath = path.join(out, 'old-source.json');
  const base = JSON.parse(fs.readFileSync(basePath));
  const data = { base: 'old-source.json', base_sha256: sha(fs.readFileSync(basePath)), changed: Object.fromEntries(Object.entries(current).filter(([p, h]) => base[p] !== h)), removed: Object.keys(base).filter(p => !(p in current)) };
  fs.writeFileSync(freezePath, JSON.stringify(data, null, 2) + '\n');
  console.log(freezePath);
} else if (mode === 'run') {
  const expected = readFreeze(freezePath);
  const stable = () => {
    const current = snapshot();
    return Object.keys(current).length === Object.keys(expected).length && Object.entries(current).every(([p, h]) => expected[p] === h);
  };
  const before = stable();
  if (!before) throw new Error('sources changed before ' + name);
  const overlayMatch = command.match(/-overlay=(\S+)/);
  const overlayPaths = overlayMatch ? [overlayMatch[1], ...Object.values(JSON.parse(fs.readFileSync(overlayMatch[1])).Replace)] : [];
  const overlayInputs = Object.fromEntries(overlayPaths.map(p => [p, sha(fs.readFileSync(p))]));
  const log = path.join(out, name + '.log');
  const fd = fs.openSync(log, 'w');
  const start = new Date();
  const result = cp.spawnSync('/bin/bash', ['-c', command], { cwd: path.join(root, 'tools/gomad3'), env, stdio: ['ignore', fd, fd], timeout: 600000 });
  fs.closeSync(fd);
  const end = new Date();
  const after = stable();
  const overlayStable = overlayPaths.every(p => sha(fs.readFileSync(p)) === overlayInputs[p]);
  const tools = [path.join(goBin, 'go'), '/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0', '/tmp/fn109-lint-tools.ZdNe1t50/errortype'];
  const receipt = { command, cwd: path.join(root, 'tools/gomad3'), environment: { unset, set: Object.fromEntries(['GOENV', 'GOFLAGS', 'GOWORK', 'GOTOOLCHAIN', 'GOMAXPROCS', 'PATH'].map(k => [k, env[k]])) }, tools: Object.fromEntries(tools.map(p => [p, sha(fs.readFileSync(p))])), source_freeze: freezePath, source_freeze_sha256: sha(fs.readFileSync(freezePath)), stability_before: before, stability_after: after, overlay_inputs: overlayInputs, overlay_stability: overlayStable, start: start.toISOString(), end: end.toISOString(), elapsed_seconds: (end - start) / 1000, exit_code: result.status, signal: result.signal, error: result.error?.message, log, log_sha256: sha(fs.readFileSync(log)) };
  fs.writeFileSync(path.join(out, name + '.receipt.json'), JSON.stringify(receipt, null, 2) + '\n');
  console.log(JSON.stringify({ name, exit_code: result.status, elapsed_seconds: receipt.elapsed_seconds, stability_before: before, stability_after: after, error: receipt.error }));
  if (!after || !overlayStable || result.error) process.exitCode = 2;
} else throw new Error('usage: freeze <freeze> | run <freeze> <name> <command>');
